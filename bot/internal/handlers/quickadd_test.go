package handlers

import (
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// quickHandler is a signed-in Russian chat with nothing running: the state in
// which Denis types a line without pressing anything first.
func quickHandler(t *testing.T) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	h, fake := newTestHandler(t)
	const chat = int64(7001)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	return h, fake, chat
}

func say(h *Handler, chat int64, text string) {
	h.HandleMessage(&tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{ID: chat},
		From: &tgbotapi.User{ID: chat, FirstName: "Denis"},
	})
}

func press(h *Handler, chat int64, data string) {
	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID:      "cb",
		Data:    data,
		From:    &tgbotapi.User{ID: chat, FirstName: "Denis"},
		Message: &tgbotapi.Message{MessageID: 55, Chat: &tgbotapi.Chat{ID: chat}},
	})
}

// 🔴 Denis, 17.09: «когда пишешь что-то без предыдущей команды, он просто
// говорит, что не понял». A line typed from nowhere is a request, not noise.
func TestQuickAddAsksWhatToCreate(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "завтра в 15 стоматолог напомни за час")

	got := fake.last(t)
	if strings.Contains(got.Text, "Не понял") {
		t.Fatalf("free text still answered «не понял»: %q", got.Text)
	}
	for _, button := range []string{"qa_event", "qa_task", "qa_note", "qa_cancel"} {
		if !strings.Contains(got.Markup, button) {
			t.Errorf("the question has no %s button; markup = %s", button, got.Markup)
		}
	}
}

// And the answer uses everything the event path knows — not a second, poorer
// parser.
func TestQuickAddEventOpensTheFullCard(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "завтра в 15 стоматолог напомни за час")
	press(h, chat, "qa_event")

	card := fake.last(t).Text
	for _, want := range []string{"стоматолог", "15:00", "1 ч"} {
		if !strings.Contains(card, want) {
			t.Errorf("the card is missing %q:\n%s", want, card)
		}
	}
}

// Two days in one line → after «Событие», the one-or-list question.
func TestQuickAddTwoEventsAsksOneOrList(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "tuesday morning office work wednesday midnight movie")
	press(h, chat, "qa_event")

	got := fake.last(t)
	if !strings.Contains(got.Markup, "dr_many") || !strings.Contains(got.Text, "2") {
		t.Errorf("expected «одна запись или список из 2»; got %q / %s", got.Text, got.Markup)
	}
}

// Choosing «Событие» for a line that says «задача» makes an event, not a task.
func TestQuickAddEventOverridesTheTaskWord(t *testing.T) {
	h, _, chat := quickHandler(t)
	say(h, chat, "задача завтра в 15 позвонить в банк")
	press(h, chat, "qa_event")
	st, ok := draftOf(h, chat)
	if !ok || st.D.IsTask {
		t.Errorf("«Событие» was chosen, the draft is a task: ok=%v", ok)
	}
}

// Cancel leaves nothing running: the next line is a fresh quick add.
func TestQuickAddCancelClearsTheFlow(t *testing.T) {
	h, _, chat := quickHandler(t)
	// A line with a clock time still asks (task or event is a real choice
	// there); a plain one is saved at once and leaves nothing to cancel.
	say(h, chat, "стоматолог завтра в 15")
	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow == "" {
		t.Fatalf("no question is pending — cancelling it would prove nothing")
	}
	press(h, chat, "qa_cancel")
	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow != "" {
		t.Errorf("after cancel the flow is %q, want none", flow)
	}
}
