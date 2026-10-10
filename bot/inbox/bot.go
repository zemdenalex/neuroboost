package inbox

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Service is the bot: one owner, everything he sends becomes a note.
type Service struct {
	Owner    int64
	Token    string // only to redact it; the bot.Bot already holds it
	API      *API
	Projects []string // the project buttons; any other #word still counts
	MaxBytes int64    // files above this are recorded, not carried (nginx caps at 1 MB today)
	HTTP     *http.Client
	Col      *Collector
	b        *bot.Bot
}

// Bind gives the service its bot and its collector's flush.
func (s *Service) Bind(b *bot.Bot) {
	s.b = b
	s.Col = &Collector{Window: Window, MaxBytes: s.MaxBytes, Flush: func(chatID int64, ds []Draft) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		for _, d := range ds {
			s.save(ctx, chatID, d)
		}
	}}
}

// Handle is the bot's default handler. Anyone but the owner gets silence: the
// bot is «just for me», and an answer would confirm to a stranger it exists.
func (s *Service) Handle(ctx context.Context, b *bot.Bot, u *models.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("inbox: handler panic: %v", r)
		}
	}()
	switch {
	case u.Message != nil:
		if u.Message.From == nil || u.Message.From.ID != s.Owner {
			return
		}
		if t := strings.TrimSpace(u.Message.Text); t == "/start" || t == "/help" {
			s.help(ctx, u.Message.Chat.ID)
			return
		}
		s.Col.Add(u.Message.Chat.ID, u.Message)
	case u.CallbackQuery != nil:
		if u.CallbackQuery.From.ID != s.Owner {
			return
		}
		s.callback(ctx, u.CallbackQuery)
	case u.InlineQuery != nil:
		s.inline(ctx, u.InlineQuery)
	case u.ChosenInlineResult != nil:
		if u.ChosenInlineResult.From.ID != s.Owner {
			return
		}
		r := u.ChosenInlineResult
		typ, projects := Tags(r.Query)
		s.save(ctx, s.Owner, Draft{ClientKey: "inline:" + r.ResultID, Type: typ, Projects: projects, Text: r.Query})
	}
}

func (s *Service) save(ctx context.Context, chatID int64, d Draft) {
	via := "chat"
	if strings.HasPrefix(d.ClientKey, "inline:") {
		via = "inline"
	}
	n, created, err := s.API.Create(ctx, s.Owner, d, via)
	if err != nil {
		log.Printf("inbox: save %s: %s", d.ClientKey, Redact(err.Error(), s.Token))
		s.send(ctx, chatID, "❌ Не сохранено — сервер не ответил. Перешли ещё раз.", nil)
		return
	}
	// A redelivered update finds its note (created == false). Its files are
	// uploaded again anyway: the bot may have died between saving the note and
	// carrying them, and AddMedia is idempotent per file. The card is not
	// repeated.
	var lost []string
	for _, m := range d.Media {
		data, why := s.download(ctx, m)
		if err := s.API.AddMedia(ctx, n.ID, m, data); err != nil {
			log.Printf("inbox: media %d: %s", n.ID, Redact(err.Error(), s.Token))
			why = "не загрузился"
		}
		if why != "" {
			lost = append(lost, m.Kind+" — "+why)
		}
	}
	if !created {
		return
	}
	text := Card(n)
	if via == "inline" {
		text += "\n(из inline)"
	}
	if len(lost) > 0 {
		text += "\n⚠ файл записан без содержимого: " + strings.Join(lost, "; ")
	}
	s.send(ctx, chatID, text, s.Keyboard(n))
}

// download fetches a file from Telegram. Returns nil data and the reason when
// the file is not carried. Every error is redacted here, where it is born: the
// download URL contains the token.
func (s *Service) download(ctx context.Context, m Media) ([]byte, string) {
	if m.Size > s.MaxBytes {
		return nil, fmt.Sprintf("%.1f МБ больше лимита", float64(m.Size)/(1<<20))
	}
	f, err := s.b.GetFile(ctx, &bot.GetFileParams{FileID: m.FileID})
	if err != nil {
		log.Printf("inbox: getFile: %s", Redact(err.Error(), s.Token))
		return nil, "Telegram не отдал файл"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.b.FileDownloadLink(f), nil)
	if err != nil {
		return nil, "ошибка ссылки"
	}
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("inbox: download: %s", Redact(err.Error(), s.Token))
		return nil, "не скачался"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "не скачался"
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, s.MaxBytes+1))
	if err != nil || int64(len(data)) > s.MaxBytes {
		return nil, "не скачался целиком"
	}
	return data, ""
}

// Card is the one-line answer under a saved note.
func Card(n Note) string {
	typ := n.Type
	if typ == "" || typ == "inbox" {
		typ = "inbox"
	}
	parts := []string{fmt.Sprintf("✅ #%d", n.ID), typ}
	if len(n.Projects) > 0 {
		parts = append(parts, strings.Join(n.Projects, ", "))
	}
	if n.Status != "" && n.Status != "new" {
		parts = append(parts, statusLabel[n.Status])
	}
	return strings.Join(parts, " · ")
}

var statusLabel = map[string]string{"new": "новая", "discussed": "обсудили", "done": "сделано", "dropped": "убрано"}

// Keyboard: type buttons, project buttons (toggle), and drop. The chosen type
// and projects are marked, so the buttons change what they show, not only
// what is stored.
func (s *Service) Keyboard(n Note) *models.InlineKeyboardMarkup {
	mark := func(on bool, label string) string {
		if on {
			return "• " + label
		}
		return label
	}
	var types, projects []models.InlineKeyboardButton
	for _, t := range Types {
		types = append(types, models.InlineKeyboardButton{Text: mark(n.Type == t, t), CallbackData: fmt.Sprintf("t:%d:%s", n.ID, t)})
	}
	has := map[string]bool{}
	for _, p := range n.Projects {
		has[p] = true
	}
	for _, p := range s.Projects {
		projects = append(projects, models.InlineKeyboardButton{Text: mark(has[p], p), CallbackData: fmt.Sprintf("p:%d:%s", n.ID, p)})
	}
	rows := [][]models.InlineKeyboardButton{types[:3], types[3:]}
	for i := 0; i < len(projects); i += 3 {
		rows = append(rows, projects[i:min(i+3, len(projects))])
	}
	drop := models.InlineKeyboardButton{Text: "🗑 убрать", CallbackData: fmt.Sprintf("s:%d:dropped", n.ID)}
	if n.Status == "dropped" {
		drop = models.InlineKeyboardButton{Text: "↩ вернуть", CallbackData: fmt.Sprintf("s:%d:new", n.ID)}
	}
	rows = append(rows, []models.InlineKeyboardButton{drop})
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ParseCallback reads «t|p|s:<id>:<value>».
func ParseCallback(data string) (kind byte, id int64, value string, ok bool) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 || len(parts[0]) != 1 {
		return 0, 0, "", false
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || parts[2] == "" {
		return 0, 0, "", false
	}
	return parts[0][0], id, parts[2], true
}

func (s *Service) callback(ctx context.Context, q *models.CallbackQuery) {
	answer := func(text string) {
		_, _ = s.b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID, Text: text})
	}
	kind, id, value, ok := ParseCallback(q.Data)
	if !ok {
		answer("")
		return
	}
	var fields map[string]any
	switch kind {
	case 't':
		fields = map[string]any{"type": value}
	case 's':
		fields = map[string]any{"status": value}
	case 'p':
		cur, err := s.API.Get(ctx, id)
		if err != nil {
			answer("Сервер не ответил")
			return
		}
		next := []string{}
		found := false
		for _, p := range cur.Projects {
			if p == value {
				found = true
				continue
			}
			next = append(next, p)
		}
		if !found {
			next = append(next, value)
		}
		fields = map[string]any{"projects": next}
	default:
		answer("")
		return
	}
	n, err := s.API.Patch(ctx, id, fields)
	if err != nil {
		log.Printf("inbox: patch %d: %s", id, Redact(err.Error(), s.Token))
		answer("Не сохранилось")
		return
	}
	answer("")
	if q.Message.Message != nil {
		_, _ = s.b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: q.Message.Message.Chat.ID, MessageID: q.Message.Message.ID,
			Text: Card(n), ReplyMarkup: s.Keyboard(n)})
	}
}

// inline answers «@bot text #idea» with one result. Saving happens on
// chosen_inline_result, not here: an inline query fires on every keystroke.
// ⚠ Choosing the result posts the text into the chat it was typed in.
func (s *Service) inline(ctx context.Context, q *models.InlineQuery) {
	results := []models.InlineQueryResult{}
	text := strings.TrimSpace(q.Query)
	if q.From.ID == s.Owner && text != "" {
		typ, projects := Tags(text)
		desc := "inbox"
		if typ != "" {
			desc = typ
		}
		if len(projects) > 0 {
			desc += " · " + strings.Join(projects, ", ")
		}
		results = append(results, &models.InlineQueryResultArticle{
			ID:                  strconv.FormatInt(time.Now().UnixNano(), 36),
			Title:               "Сохранить в inbox",
			Description:         desc + " — " + text,
			InputMessageContent: &models.InputTextMessageContent{MessageText: text},
		})
	}
	_, err := s.b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: q.ID, Results: results, CacheTime: 0, IsPersonal: true})
	if err != nil {
		log.Printf("inbox: inline: %s", Redact(err.Error(), s.Token))
	}
}

func (s *Service) help(ctx context.Context, chatID int64) {
	text := "Присылай что угодно — текст, ссылку, пересылку, фото, голос. Всё станет заметкой.\n\n" +
		"Теги в тексте: #idea #link #feedback #reflection #note — тип; любое другое #слово — проект.\n" +
		"Без тегов — выбери кнопками или оставь: попадёт в inbox.\n" +
		"Пересылка + своё сообщение рядом = одна заметка с твоим комментарием.\n" +
		"Из любого чата: @" + s.botName(ctx) + " текст #idea"
	if counts, err := s.API.CountNew(ctx); err == nil {
		var parts []string
		total := 0
		for _, t := range append([]string{"inbox"}, Types...) {
			if c := counts[t]; c > 0 {
				parts = append(parts, fmt.Sprintf("%s %d", t, c))
				total += c
			}
		}
		text += fmt.Sprintf("\n\nНовых: %d", total)
		if len(parts) > 0 {
			text += " (" + strings.Join(parts, " · ") + ")"
		}
	}
	s.send(ctx, chatID, text, nil)
}

func (s *Service) botName(ctx context.Context) string {
	if me, err := s.b.GetMe(ctx); err == nil {
		return me.Username
	}
	return "бот"
}

func (s *Service) send(ctx context.Context, chatID int64, text string, kb *models.InlineKeyboardMarkup) {
	p := &bot.SendMessageParams{ChatID: chatID, Text: text,
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: bot.True()}}
	if kb != nil {
		p.ReplyMarkup = kb
	}
	if _, err := s.b.SendMessage(ctx, p); err != nil {
		log.Printf("inbox: send: %s", Redact(err.Error(), s.Token))
	}
}
