package inbox

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestTags(t *testing.T) {
	cases := []struct {
		in, typ, projects string
	}{
		{"#idea #avocar статья про кэш", "idea", "avocar"},
		{"#Идея #AvoCar #avocar #graph", "idea", "avocar,graph"},
		{"#link #reflection про лупы", "link", ""}, // first type wins, the second is not a project
		{"просто мысль", "", ""},
		{"#harness", "", "harness"},
		{"цена#123 не тег, а #рефлексия да", "reflection", ""},
	}
	for _, c := range cases {
		typ, ps := Tags(c.in)
		if typ != c.typ || strings.Join(ps, ",") != c.projects {
			t.Errorf("Tags(%q) = %q %v, want %q [%s]", c.in, typ, ps, c.typ, c.projects)
		}
	}
}

func TestOriginCoversAllFourKinds(t *testing.T) {
	cases := []struct {
		name string
		o    *models.MessageOrigin
		want Source
	}{
		{"user with username", &models.MessageOrigin{Type: models.MessageOriginTypeUser, MessageOriginUser: &models.MessageOriginUser{
			Date: 10, SenderUser: models.User{FirstName: "Ann", LastName: "Lee", Username: "ann"}}},
			Source{Kind: "user", Name: "Ann Lee", Username: "ann", URL: "https://t.me/ann", Date: 10}},
		{"hidden user", &models.MessageOrigin{Type: models.MessageOriginTypeHiddenUser, MessageOriginHiddenUser: &models.MessageOriginHiddenUser{
			Date: 11, SenderUserName: "Кто-то"}},
			Source{Kind: "hidden_user", Name: "Кто-то", Date: 11}},
		{"group as sender", &models.MessageOrigin{Type: models.MessageOriginTypeChat, MessageOriginChat: &models.MessageOriginChat{
			Date: 12, SenderChat: models.Chat{ID: -100500, Title: "Группа"}}},
			Source{Kind: "chat", Name: "Группа", Date: 12}},
		{"public channel", &models.MessageOrigin{Type: models.MessageOriginTypeChannel, MessageOriginChannel: &models.MessageOriginChannel{
			Date: 13, MessageID: 77, Chat: models.Chat{ID: -1001234, Title: "Канал", Username: "chan"}}},
			Source{Kind: "channel", Name: "Канал", Username: "chan", URL: "https://t.me/chan/77", Date: 13}},
		{"private channel", &models.MessageOrigin{Type: models.MessageOriginTypeChannel, MessageOriginChannel: &models.MessageOriginChannel{
			Date: 14, MessageID: 5, Chat: models.Chat{ID: -1009876543, Title: "Закрытый"}}},
			Source{Kind: "channel", Name: "Закрытый", URL: "https://t.me/c/9876543/5", Date: 14}},
	}
	for _, c := range cases {
		got := Origin(c.o)
		if got == nil || *got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
	if Origin(nil) != nil {
		t.Error("own message must have no source")
	}
}

func TestContentKeepsHiddenLinks(t *testing.T) {
	m := &models.Message{Text: "look at this thread", Entities: []models.MessageEntity{
		{Type: models.MessageEntityTypeTextLink, Offset: 8, Length: 4, URL: "https://x.com/a/status/1"},
	}}
	if got := Content(m); got != "look at this thread\n\nhttps://x.com/a/status/1" {
		t.Errorf("got %q", got)
	}
	cap := &models.Message{Caption: "фото https://e.com", CaptionEntities: []models.MessageEntity{
		{Type: models.MessageEntityTypeTextLink, URL: "https://e.com"}}}
	if got := Content(cap); got != "фото https://e.com" {
		t.Errorf("a link already in the text is not repeated: %q", got)
	}
}

func TestMediaOfPicksTheLargestPhotoThatFits(t *testing.T) {
	m := &models.Message{Photo: []models.PhotoSize{
		{FileID: "s", FileUniqueID: "us", FileSize: 10_000},
		{FileID: "m", FileUniqueID: "um", FileSize: 300_000},
		{FileID: "l", FileUniqueID: "ul", FileSize: 2_000_000},
	}}
	got := MediaOf(m, 1_000_000)
	if len(got) != 1 || got[0].FileID != "m" {
		t.Fatalf("got %+v, want the 300 KB size", got)
	}
	if got := MediaOf(m, 5_000_000); got[0].FileID != "l" {
		t.Errorf("with room, the largest: %+v", got)
	}
	v := MediaOf(&models.Message{Voice: &models.Voice{FileID: "v", FileUniqueID: "uv", FileSize: 9, MimeType: "audio/ogg"}}, 1)
	if len(v) != 1 || v[0].Kind != "voice" {
		t.Errorf("voice: %+v", v)
	}
}

func TestRedactRemovesTheToken(t *testing.T) {
	token := "123456:AAH-real_shaped_token"
	err := fmt.Errorf("download: %w", errors.New(`Get "https://api.telegram.org/file/bot`+token+`/voice/file_1.oga": dial tcp: timeout`))
	got := Redact(err.Error(), token)
	if strings.Contains(got, "AAH-real_shaped_token") || strings.Contains(got, "123456:") {
		t.Errorf("token survived: %s", got)
	}
	// A token this process does not know (another bot's) is caught by shape.
	if got := Redact("bot987:XYZ_abc-1 failed", ""); strings.Contains(got, "XYZ") {
		t.Errorf("shape redaction missed: %s", got)
	}
}
