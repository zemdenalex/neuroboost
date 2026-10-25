// Package inbox is the inbox bot: notes, ideas, links, feedback and
// reflections Denis sends it, kept for a session on the PC instead of acted on
// at once. No LLM anywhere — tags are typed, or chosen with a button.
// Spec: docs/team/architecture/V003-20261010-arc-inbox-bot.md.
//
// It runs on github.com/go-telegram/bot, not on the main bot's
// go-telegram-bot-api v5.5.1: that one predates Bot API 7.0 and decodes no
// forward_origin, which is half of what this bot is for.
package inbox

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
)

// Types are the five kinds Denis named, plus inbox for «not sorted yet».
var Types = []string{"idea", "link", "feedback", "reflection", "note"}

// typeAlias maps a hashtag to a type. Russian spellings too: he types both.
var typeAlias = map[string]string{
	"idea": "idea", "идея": "idea", "идеи": "idea",
	"link": "link", "ссылка": "link", "линк": "link",
	"feedback": "feedback", "фидбек": "feedback", "фидбэк": "feedback", "отзыв": "feedback",
	"reflection": "reflection", "рефлексия": "reflection", "мысль": "reflection",
	"note": "note", "заметка": "note",
}

var hashtag = regexp.MustCompile(`(?:^|\s)#([\p{L}\p{N}_-]+)`)

// Tags reads Denis's own text: the first type hashtag sets the type, every
// other hashtag is a project. Called only on text he wrote himself — hashtags
// inside a forwarded post belong to its author, not to him.
func Tags(own string) (typ string, projects []string) {
	seen := map[string]bool{}
	for _, m := range hashtag.FindAllStringSubmatch(own, -1) {
		tag := strings.ToLower(m[1])
		if t, ok := typeAlias[tag]; ok {
			if typ == "" {
				typ = t
			}
			continue
		}
		if !seen[tag] {
			seen[tag] = true
			projects = append(projects, tag)
		}
	}
	return typ, projects
}

// Source is where a forward came from — the same shape the API stores.
type Source struct {
	Kind     string `json:"kind"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	URL      string `json:"url,omitempty"`
	Date     int64  `json:"date,omitempty"`
}

// Origin turns forward_origin into a Source. A channel post gets a link: the
// public form by username, or t.me/c/<id>/<msg> for a private channel, which
// opens only for its members but is still the way back to the post.
func Origin(o *models.MessageOrigin) *Source {
	if o == nil {
		return nil
	}
	switch o.Type {
	case models.MessageOriginTypeUser:
		u := o.MessageOriginUser
		s := &Source{Kind: "user", Name: strings.TrimSpace(u.SenderUser.FirstName + " " + u.SenderUser.LastName),
			Username: u.SenderUser.Username, Date: int64(u.Date)}
		if u.SenderUser.Username != "" {
			s.URL = "https://t.me/" + u.SenderUser.Username
		}
		return s
	case models.MessageOriginTypeHiddenUser:
		h := o.MessageOriginHiddenUser
		return &Source{Kind: "hidden_user", Name: h.SenderUserName, Date: int64(h.Date)}
	case models.MessageOriginTypeChat:
		c := o.MessageOriginChat
		s := &Source{Kind: "chat", Name: chatName(c.SenderChat), Username: c.SenderChat.Username, Date: int64(c.Date)}
		if c.SenderChat.Username != "" {
			s.URL = "https://t.me/" + c.SenderChat.Username
		}
		return s
	case models.MessageOriginTypeChannel:
		c := o.MessageOriginChannel
		return &Source{Kind: "channel", Name: chatName(c.Chat), Username: c.Chat.Username,
			URL: postLink(c.Chat, c.MessageID), Date: int64(c.Date)}
	}
	return nil
}

func chatName(c models.Chat) string {
	if c.Title != "" {
		return c.Title
	}
	return strings.TrimSpace(c.FirstName + " " + c.LastName)
}

func postLink(c models.Chat, msgID int) string {
	if c.Username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", c.Username, msgID)
	}
	// Private channel ids are -100<internal>; the link wants <internal>.
	id := strconv.FormatInt(c.ID, 10)
	if !strings.HasPrefix(id, "-100") {
		return ""
	}
	return fmt.Sprintf("https://t.me/c/%s/%d", strings.TrimPrefix(id, "-100"), msgID)
}

// originKey says whether two forwards came from the same place, so several
// posts forwarded at once from one channel become one note.
func originKey(s *Source) string {
	if s == nil {
		return ""
	}
	return s.Kind + "|" + s.Name + "|" + s.Username
}

// Content is the message's words: text or caption, plus the URLs hidden
// behind text_link entities — an X post's «link» is often only that.
func Content(m *models.Message) string {
	text, ents := m.Text, m.Entities
	if text == "" {
		text, ents = m.Caption, m.CaptionEntities
	}
	var hidden []string
	for _, e := range ents {
		if e.Type == models.MessageEntityTypeTextLink && e.URL != "" && !strings.Contains(text, e.URL) {
			hidden = append(hidden, e.URL)
		}
	}
	if len(hidden) > 0 {
		text = strings.TrimSpace(text + "\n\n" + strings.Join(hidden, "\n"))
	}
	return text
}

// Media is one file attached to a message.
type Media struct {
	Kind     string
	FileID   string
	UniqueID string
	Size     int64
	Mime     string
	FileName string
}

// MediaOf lists the files of a message. For a photo it takes the largest size
// that still fits maxBytes — Telegram sends several sizes of one picture.
func MediaOf(m *models.Message, maxBytes int64) []Media {
	var out []Media
	if len(m.Photo) > 0 {
		best := m.Photo[0]
		for _, p := range m.Photo {
			if int64(p.FileSize) <= maxBytes && p.FileSize >= best.FileSize {
				best = p
			}
		}
		out = append(out, Media{Kind: "photo", FileID: best.FileID, UniqueID: best.FileUniqueID,
			Size: int64(best.FileSize), Mime: "image/jpeg"})
	}
	if v := m.Voice; v != nil {
		out = append(out, Media{Kind: "voice", FileID: v.FileID, UniqueID: v.FileUniqueID, Size: v.FileSize, Mime: v.MimeType})
	}
	if a := m.Audio; a != nil {
		out = append(out, Media{Kind: "audio", FileID: a.FileID, UniqueID: a.FileUniqueID, Size: a.FileSize, Mime: a.MimeType, FileName: a.FileName})
	}
	if v := m.Video; v != nil {
		out = append(out, Media{Kind: "video", FileID: v.FileID, UniqueID: v.FileUniqueID, Size: v.FileSize, Mime: v.MimeType, FileName: v.FileName})
	}
	if v := m.VideoNote; v != nil {
		out = append(out, Media{Kind: "video_note", FileID: v.FileID, UniqueID: v.FileUniqueID, Size: int64(v.FileSize), Mime: "video/mp4"})
	}
	if d := m.Document; d != nil {
		out = append(out, Media{Kind: "document", FileID: d.FileID, UniqueID: d.FileUniqueID, Size: d.FileSize, Mime: d.MimeType, FileName: d.FileName})
	}
	return out
}

var tokenShape = regexp.MustCompile(`bot[0-9]+:[A-Za-z0-9_-]+`)

// Redact removes the bot token from a string. The file download URL is
// api.telegram.org/file/bot<TOKEN>/…, so any error about a download carries
// the token (gotcha 14) — this runs where that error is born, before it is
// logged or sent anywhere.
func Redact(s, token string) string {
	if token != "" {
		s = strings.ReplaceAll(s, token, "<REDACTED>")
	}
	return tokenShape.ReplaceAllString(s, "bot<REDACTED>")
}
