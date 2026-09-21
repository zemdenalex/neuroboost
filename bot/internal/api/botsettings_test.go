package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 🔴 PATCH /api/auth/me replaces the whole settings blob (gotcha 21) and
// MergeSettings is shallow — writing {"bot": {key: value}} alone would erase
// the language and the keyword vocabulary that live beside it.
func TestSetBotSettingKeepsTheRestOfTheBotSection(t *testing.T) {
	var written string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch {
			b, _ := io.ReadAll(r.Body)
			written = string(b)
			_, _ = w.Write([]byte(`{"data":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"settings":{"work_start":"09:00","bot":{"lang":"ru","keywords":{"x":1},"stats_scale":"peak"}}}}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	if v, err := c.BotSetting("tok", "stats_scale"); err != nil || v != "peak" {
		t.Errorf("read = %q, %v", v, err)
	}
	if err := c.SetBotSetting("tok", "stats_scale", "work"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"stats_scale":"work"`, `"lang":"ru"`, `"keywords"`, `"work_start":"09:00"`} {
		if !strings.Contains(written, want) {
			t.Errorf("PATCH %s lost %s", written, want)
		}
	}
}
