package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/config"
)

// Through the door, not around it (CLAUDE.md, v0.4.11.4 lesson): the Mini App
// sign-in is tested as an HTTP request on the route, against a real database.
func TestMiniAppSignInCreatesThenFindsTheSameUser(t *testing.T) {
	d := linkingDB(t)
	ctx := context.Background()
	tgID := time.Now().UnixNano() % 1_000_000_000_000
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE tg_id = $1`, tgID) })

	r := chi.NewRouter()
	r.Post("/api/auth/telegram-webapp", NewHandler(d, &config.Config{TelegramBotToken: webAppToken, JWTSecret: "s"}).TelegramWebApp)

	signIn := func(initData string) (int, AuthResponse) {
		body, _ := json.Marshal(TelegramWebAppLoginRequest{InitData: initData})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/telegram-webapp", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		var env struct {
			Data AuthResponse `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &env)
		return rec.Code, env.Data
	}
	fields := map[string]string{
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
		"user":      `{"id":` + strconv.FormatInt(tgID, 10) + `,"first_name":"Mini"}`,
	}

	code, first := signIn(signInitData(t, webAppToken, fields))
	if code != http.StatusOK || first.Token == "" || first.User.ID == "" {
		t.Fatalf("first sign-in: %d %+v", code, first)
	}
	if first.User.TgID == nil || *first.User.TgID != tgID {
		t.Fatal("user not tied to the Telegram id")
	}
	code, second := signIn(signInitData(t, webAppToken, fields))
	if code != http.StatusOK || second.User.ID != first.User.ID {
		t.Fatalf("second sign-in made another user: %d, %s vs %s", code, second.User.ID, first.User.ID)
	}

	// Negative control on the same route: a forged string gets 401 and no user.
	if code, _ := signIn(signInitData(t, "999:other", fields)); code != http.StatusUnauthorized {
		t.Fatalf("forged initData: %d, want 401", code)
	}
}

// Review M3 (25.09): the bot and the Mini App can both see "no such tg_id" and
// both INSERT; the loser hit the unique index and answered 500. Creating a
// user who already exists must hand back that user instead.
func TestCreatingATelegramUserWhoAlreadyExistsReturnsThem(t *testing.T) {
	d := linkingDB(t)
	ctx := context.Background()
	tgID := time.Now().UnixNano()%1_000_000_000_000 + 7
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE tg_id = $1`, tgID) })
	var existing string
	if err := d.Pool.QueryRow(ctx, `INSERT INTO "user" (tg_id, settings) VALUES ($1, '{}') RETURNING id`, tgID).Scan(&existing); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(d, &config.Config{TelegramBotToken: webAppToken, JWTSecret: "s"})
	u, err := h.createTelegramUserOrFind(ctx, TelegramLoginRequest{ID: tgID, FirstName: "Race", AuthDate: time.Now().Unix()})
	if err != nil {
		t.Fatalf("lost the race with an error: %v", err)
	}
	if u.ID != existing {
		t.Fatalf("got user %s, want the existing %s", u.ID, existing)
	}
}

// An account born in the Mini App takes
// Telegram's language by the bot's own rule — «ru*» Russian, anything else
// English — so a person who never wrote to the bot does not open it in Russian.
func TestAMiniAppAccountStartsInTelegramsLanguage(t *testing.T) {
	d := linkingDB(t)
	ctx := context.Background()
	r := chi.NewRouter()
	r.Post("/api/auth/telegram-webapp", NewHandler(d, &config.Config{TelegramBotToken: webAppToken, JWTSecret: "s"}).TelegramWebApp)

	for i, c := range []struct{ code, want string }{{"en", "en"}, {"ru-RU", "ru"}, {"ar", "en"}, {"", "ru"}} {
		tgID := time.Now().UnixNano()%1_000_000_000_000 + int64(100+i)
		t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE tg_id = $1`, tgID) })
		user := `{"id":` + strconv.FormatInt(tgID, 10) + `,"first_name":"Lang"`
		if c.code != "" {
			user += `,"language_code":"` + c.code + `"`
		}
		body, _ := json.Marshal(TelegramWebAppLoginRequest{InitData: signInitData(t, webAppToken, map[string]string{
			"auth_date": strconv.FormatInt(time.Now().Unix(), 10), "user": user + "}",
		})})
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/telegram-webapp", bytes.NewReader(body)))
		var env struct {
			Data AuthResponse `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &env)
		if rec.Code != http.StatusOK || env.Data.User.Locale != c.want {
			t.Errorf("language_code %q: %d, locale %q, want %q", c.code, rec.Code, env.Data.User.Locale, c.want)
		}
	}
}
