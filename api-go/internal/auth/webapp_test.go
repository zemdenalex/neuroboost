package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const webAppToken = "123456:TEST-token-for-webapp"

// signInitData builds initData the way Telegram does (core.telegram.org/bots/webapps):
// secret = HMAC-SHA256(key "WebAppData", bot token); hash = HMAC-SHA256(secret,
// every field but hash, sorted, "k=v" joined by "\n"). Written independently of
// the code under test so a shared mistake cannot make both agree.
func signInitData(t *testing.T, token string, fields map[string]string) string {
	t.Helper()
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(lines, "\n")))
	q := url.Values{}
	for k, v := range fields {
		q.Set(k, v)
	}
	q.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return q.Encode()
}

func freshFields(now time.Time) map[string]string {
	return map[string]string{
		"auth_date": strconv.FormatInt(now.Unix(), 10),
		"query_id":  "AAH-test",
		"user":      `{"id":4242,"first_name":"Ann","last_name":"Lee","username":"ann_l","language_code":"ru","photo_url":"https://t.me/i/userpic/320/a.jpg"}`,
	}
}

func TestVerifyWebAppInitDataAcceptsARealSignature(t *testing.T) {
	now := time.Unix(1_790_000_000, 0)
	u, err := verifyWebAppInitData(signInitData(t, webAppToken, freshFields(now)), webAppToken, now)
	if err != nil {
		t.Fatalf("valid initData refused: %v", err)
	}
	if u.ID != 4242 || u.FirstName != "Ann" || u.LastName != "Lee" || u.Username != "ann_l" {
		t.Fatalf("user = %+v", u)
	}
	if u.PhotoURL != "https://t.me/i/userpic/320/a.jpg" {
		t.Fatalf("photo = %q", u.PhotoURL)
	}
	if u.AuthDate != now.Unix() {
		t.Fatalf("auth_date = %d", u.AuthDate)
	}
}

func TestVerifyWebAppInitDataRefuses(t *testing.T) {
	now := time.Unix(1_790_000_000, 0)
	good := signInitData(t, webAppToken, freshFields(now))

	tampered := strings.Replace(good, "Ann", "Bob", 1)
	if tampered == good {
		t.Fatal("tamper did not change the string")
	}

	// The Login Widget scheme (secret = SHA256(token)) must not pass here.
	widget := func() string {
		f := freshFields(now)
		keys := []string{"auth_date", "query_id", "user"}
		lines := []string{}
		for _, k := range keys {
			lines = append(lines, k+"="+f[k])
		}
		s := sha256.Sum256([]byte(webAppToken))
		mac := hmac.New(sha256.New, s[:])
		mac.Write([]byte(strings.Join(lines, "\n")))
		q := url.Values{}
		for k, v := range f {
			q.Set(k, v)
		}
		q.Set("hash", hex.EncodeToString(mac.Sum(nil)))
		return q.Encode()
	}()

	noUser := freshFields(now)
	delete(noUser, "user")

	cases := []struct {
		name, data, token string
		at                time.Time
	}{
		{"one field changed", tampered, webAppToken, now},
		{"another bot's token", good, "999:other", now},
		{"signed with the Login Widget scheme", widget, webAppToken, now},
		{"older than a day", good, webAppToken, now.Add(25 * time.Hour)},
		{"no hash", strings.Split(good, "&hash=")[0], webAppToken, now},
		{"empty token on the server", good, "", now},
		{"no user field", signInitData(t, webAppToken, noUser), webAppToken, now},
		{"not a query string", "%zz", webAppToken, now},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := verifyWebAppInitData(c.data, c.token, c.at); err == nil {
				t.Fatal("accepted; want refused")
			}
		})
	}
}
