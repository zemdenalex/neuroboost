package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Проход Дениса по v0.4.11.3, 18.09 03:33.
//
// 🔴 «если начать создавать например задачу, и написать другую то он пишет что-то
// пошло не так». After showTaskCard the flow is new_task/card, and «card» was in
// no case of handleNewTaskFlow — so the second line fell to default, which
// clears the flow and apologises. Typing a second thing is the most ordinary
// act there is.
func TestASecondTaskLineReplacesTheFirst(t *testing.T) {
	h, fake, chat := quickHandler(t)

	say(h, chat, "задача на завтра помыться")
	if got := fake.last(t); !strings.Contains(got.Text, "помыться") {
		t.Fatalf("первая задача не показана: %q", got.Text)
	}

	say(h, chat, "задача на завтра выкинуть мусор")
	got := fake.last(t)
	if strings.Contains(got.Text, "Что-то пошло не так") {
		t.Fatalf("вторая строка ответила ошибкой: %q", got.Text)
	}
	if !strings.Contains(got.Text, "мусор") {
		t.Errorf("вторая строка не заменила карточку: %q", got.Text)
	}
}

// 🔴 The same line twice — exactly what Denis typed — must also work. This is
// the literal reproduction from the log.
func TestTheSameTaskLineTwiceDoesNotApologise(t *testing.T) {
	h, fake, chat := quickHandler(t)

	say(h, chat, "задача на завтра помыться")
	say(h, chat, "задача на завтра помыться")

	if got := fake.last(t); strings.Contains(got.Text, "Что-то пошло не так") {
		t.Errorf("повтор той же строки ответил ошибкой: %q", got.Text)
	}
}

// 🔴 «И это работает даже если на этапе создания нажать отмена, он вернется в
// меню, но следующая задача все равно не создастся». Cancel drew the menu and
// left the flow running, so the NEXT line was still read as an answer to a
// question that was no longer on screen.
func TestCancelReallyClearsTheFlow(t *testing.T) {
	h, fake, chat := quickHandler(t)

	say(h, chat, "задача на завтра помыться")
	press(h, chat, "main_menu")

	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow != "" {
		t.Errorf("отмена оставила флоу %q — следующая строка уедет в него", flow)
	}

	say(h, chat, "задача на завтра выкинуть мусор")
	if got := fake.last(t); strings.Contains(got.Text, "Что-то пошло не так") {
		t.Errorf("после отмены следующая задача ответила ошибкой: %q", got.Text)
	}
}

// 🔴 Denis, 18.09: «на русском про заметку я имел в виду тоже поменять на
// описание, потому что заметка это другая сущность». A note is a thing
// NeuroBoost is meant to have (Obsidian, .md); calling a description a note
// promises that thing and delivers a text field.
func TestTheDescriptionIsCalledADescription(t *testing.T) {
	st := draftFrom("стоматолог завтра 15:00")
	card := renderDraft(i18n.RU, st, tuesday15())
	if strings.Contains(card, "Заметка") {
		t.Errorf("карточка всё ещё называет описание заметкой:\n%s", card)
	}
	if !strings.Contains(card, "Описание:") {
		t.Errorf("карточка не называет описание:\n%s", card)
	}
	en := renderDraft(i18n.EN, st, tuesday15())
	if strings.Contains(en, "Note:") || !strings.Contains(en, "Description:") {
		t.Errorf("english card still says Note:\n%s", en)
	}
}

// «Напомнить» is an imperative aimed at the bot; the field is a property of the
// event. Denis, 18.09: «надо поменять на нейтральное напоминания или
// уведомления, и не напоминать поменять на без напоминаний».
func TestReminderFieldIsNeutrallyNamed(t *testing.T) {
	st := draftFrom("стоматолог завтра 15:00")
	none := []int{}
	st.ReminderOffsets = &none
	card := renderDraft(i18n.RU, st, tuesday15())
	if strings.Contains(card, "Напомнить:") {
		t.Errorf("поле всё ещё называется «Напомнить»:\n%s", card)
	}
	if !strings.Contains(card, "Напоминания:") {
		t.Errorf("нет поля «Напоминания»:\n%s", card)
	}
	if strings.Contains(card, "не напоминать") || !strings.Contains(card, "без напоминаний") {
		t.Errorf("пустые напоминания названы глаголом, а не состоянием:\n%s", card)
	}
}

// 🔴 «надо сделать лимит символов в целом в заметке (но обязательно если есть
// возможность чтобы оно не обрывалось на середине слова)». The first version
// counted runes of the whole string but four short lines came to 55 and were
// printed in full, taking four lines of a card.
func TestLongDescriptionIsCutAtAWordAndOnOneLine(t *testing.T) {
	st := draftFrom("доделать сайт завтра 15:00")
	st.Description = "фывралоыфдв\nывоафдловад\nыфвоалдовыдла\nыфволафдо\nещё строка\nи ещё одна"
	card := renderDraft(i18n.RU, st, tuesday15())

	line := lineWith(card, "Описание:")
	if line == "" {
		t.Fatalf("описания нет вовсе:\n%s", card)
	}
	if strings.Count(card, "\n") > 12 {
		t.Errorf("описание разложило карточку на %d строк:\n%s", strings.Count(card, "\n"), card)
	}
	if !strings.Contains(line, "…") {
		t.Errorf("длинное описание не укорочено: %q", line)
	}
	// Not cut in the middle of a word: what precedes the ellipsis is a whole word.
	cut := strings.TrimSuffix(strings.TrimSpace(line), "…")
	if strings.HasSuffix(cut, "ыфвоалдо") {
		t.Errorf("описание обрезано посреди слова: %q", line)
	}
}

// 🔴 Denis, 18.09: «календарь как и напомнить у стандартного события в личном
// календаре» — a new event goes to the personal calendar, so the card saying
// «Календарь: нет» is simply wrong, and it is wrong in the direction that
// teaches people the field does not work.
func TestCardNamesThePersonalCalendarByDefault(t *testing.T) {
	st := draftFrom("стоматолог завтра 15:00")
	st.CalendarName = "Личный"
	card := renderDraft(i18n.RU, st, tuesday15())
	if strings.Contains(card, "Календарь: нет") {
		t.Errorf("карточка говорит «Календарь: нет», хотя календарь известен:\n%s", card)
	}
}

// 🔴 Denis, 18.09, on «Карточка задачи тоже называет все поля»: «Нет». The task
// card was a separate renderer that printed only what was filled — the exact
// defect the event card had just been cured of, living one file away.
func TestTaskCardNamesEveryField(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "задача на завтра доделать сайт")

	got := fake.last(t)
	for _, label := range []string{"Срок:", "Приоритет:", "Оценка:", "Теги:", "Календарь:", "Напоминания:", "Описание:"} {
		if !strings.Contains(got.Text, label) {
			t.Errorf("карточка задачи не называет %q:\n%s", label, got.Text)
		}
	}
	if !strings.Contains(got.Text, "нет") {
		t.Errorf("пустые поля не сказали «нет»:\n%s", got.Text)
	}
	// The value that WAS given must still be printed, or the card passes by
	// saying «нет» to everything.
	if !strings.Contains(got.Text, "19.09") {
		t.Errorf("срок из строки потерялся:\n%s", got.Text)
	}
}

// Denis, 18.09: «для времени использовать соответствующее часу эмодзи часов».
func TestTheClockFaceMatchesTheHour(t *testing.T) {
	for _, c := range []struct {
		when time.Duration
		want string
	}{
		{15 * time.Hour, "🕒"},
		{15*time.Hour + 30*time.Minute, "🕞"},
		{9 * time.Hour, "🕘"},
		{0, "🕛"},
		{12 * time.Hour, "🕛"},
		{23*time.Hour + 50*time.Minute, "🕛"}, // rounds up into the next hour
	} {
		if got := clockFace(c.when); got != c.want {
			t.Errorf("clockFace(%s) = %s, want %s", c.when, got, c.want)
		}
	}
}

func TestTheCardUsesTheMatchingClock(t *testing.T) {
	card := renderDraft(i18n.RU, draftFrom("стоматолог завтра 15:00"), tuesday15())
	if !strings.Contains(card, "🕒 15:00") {
		t.Errorf("карточка не использует часы, соответствующие времени:\n%s", card)
	}
}

// 🔴 Denis, 18.09: «if completed for the day it should stop being in the task
// list, but appear somewhere gray, and on the next day it pops up again».
//
// A repeating task answered today is not outstanding. Leaving it in the list
// would make the list lie; deleting it outright would hide that it exists.
func TestAnsweredTodayIsNotOutstanding(t *testing.T) {
	for _, c := range []struct {
		name string
		task api.Task
		want bool
	}{
		{"repeating and done today", api.Task{Rrule: "FREQ=DAILY", OccurrenceState: "done"}, true},
		{"repeating and skipped today", api.Task{Rrule: "FREQ=DAILY", OccurrenceState: "skipped"}, true},
		{"repeating and untouched", api.Task{Rrule: "FREQ=DAILY"}, false},
		// 🔴 The control that matters: a ONE-OFF task is never "answered
		// today", whatever stray state arrives. Its status is the truth, and
		// treating it as a series would make ordinary tasks vanish.
		{"one-off with a stray state", api.Task{OccurrenceState: "done"}, false},
		{"plain one-off", api.Task{}, false},
	} {
		if got := c.task.AnsweredToday(); got != c.want {
			t.Errorf("%s: AnsweredToday() = %v, want %v", c.name, got, c.want)
		}
	}
}
