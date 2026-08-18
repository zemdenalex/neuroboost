package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMergeSettingsKeepsWhatTheBotDoesNotKnowAbout(t *testing.T) {
	// The keys a bot has no idea exist — set from the web, and destroyed the
	// moment anything writes the blob without them.
	current := map[string]any{
		"header_variant":    "vertical",
		"ui_scale":          float64(120),
		"work_days":         []any{"Mon", "Tue"},
		"features":          map[string]any{"pomodoro": true},
		"quiet_hours_start": "22:00",
		"work_start":        "09:00",
	}
	merged := MergeSettings(current, map[string]any{"work_start": "08:00"})

	if merged["work_start"] != "08:00" {
		t.Errorf("the patch did not apply: work_start = %v", merged["work_start"])
	}
	for _, k := range []string{"header_variant", "ui_scale", "work_days", "features", "quiet_hours_start"} {
		if _, ok := merged[k]; !ok {
			t.Errorf("%s was dropped — PATCH /api/auth/me replaces the whole blob, so this erases it", k)
		}
	}
	if merged["header_variant"] != "vertical" || merged["ui_scale"] != float64(120) {
		t.Errorf("a surviving key was altered: %+v", merged)
	}

	// The inputs must not be mutated: the caller still holds `current` and may
	// render it after a failed write.
	if current["work_start"] != "09:00" {
		t.Error("MergeSettings wrote through to its input")
	}
}

func TestMergeSettingsHandlesAnEmptyStart(t *testing.T) {
	merged := MergeSettings(map[string]any{}, map[string]any{"work_end": "18:00"})
	if len(merged) != 1 || merged["work_end"] != "18:00" {
		t.Errorf("got %+v", merged)
	}
}

func TestPatchSettingsSendsTheWholeMergedBlob(t *testing.T) {
	var patched map[string]any
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			io.WriteString(w, `{"data":{"settings":{"ui_scale":120,"work_start":"09:00","work_end":"17:00"}}}`)
			return
		}
		gotMethod, gotPath = r.Method, r.URL.Path
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		patched, _ = body["settings"].(map[string]any)
		io.WriteString(w, `{"data":{}}`)
	}))
	defer srv.Close()

	merged, err := NewClient(srv.URL).PatchSettings("jwt", map[string]any{"work_start": "08:00"})
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if gotMethod != "PATCH" || gotPath != "/api/auth/me" {
		t.Errorf("wrote to %s %s", gotMethod, gotPath)
	}
	if patched == nil {
		t.Fatal("no settings object reached the server")
	}
	if patched["work_start"] != "08:00" {
		t.Errorf("work_start on the wire = %v", patched["work_start"])
	}
	if patched["ui_scale"] != float64(120) || patched["work_end"] != "17:00" {
		t.Errorf("the write did not carry the untouched keys: %+v", patched)
	}
	if merged["work_start"] != "08:00" {
		t.Errorf("returned blob = %+v", merged)
	}
}

func TestPatchSettingsDoesNotWriteWhenTheReadFails(t *testing.T) {
	// Without this, a failed read would merge onto an empty map and the write
	// would erase every setting the user has — the exact damage the read exists
	// to prevent, triggered by the one condition that makes it likeliest.
	var wrote bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		wrote = true
	}))
	defer srv.Close()

	if _, err := NewClient(srv.URL).PatchSettings("jwt", map[string]any{"work_start": "08:00"}); err == nil {
		t.Error("a failed read was swallowed")
	}
	if wrote {
		t.Error("it wrote settings anyway, on top of a blob it never managed to read")
	}
}
