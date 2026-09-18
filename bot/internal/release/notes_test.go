package release

import (
	"strings"
	"testing"
)

// 🔴 «Что нового» is added in the very release it must describe. An empty note
// for the newest version would be the feature failing on its first use — and
// failing quietly, because an empty screen looks like a quiet release rather
// than a bug.
func TestLatestNoteExistsInBothLanguages(t *testing.T) {
	n := Latest()
	if n.Version == "" {
		t.Fatal("no release notes at all")
	}
	if len([]rune(n.RU)) < 20 || len([]rune(n.EN)) < 20 {
		t.Errorf("%s has an empty or stub note: ru=%q en=%q", n.Version, n.RU, n.EN)
	}
}

// Every entry, not just the newest: a version added without its English half
// shows a Russian screen to an English user and nothing warns anyone.
func TestEveryNoteHasBothLanguages(t *testing.T) {
	for _, n := range All() {
		if strings.TrimSpace(n.RU) == "" || strings.TrimSpace(n.EN) == "" {
			t.Errorf("%s is missing a language: ru=%q en=%q", n.Version, n.RU, n.EN)
		}
		if n.RU == n.EN {
			t.Errorf("%s has the same text in both languages — one of them was never written", n.Version)
		}
	}
}

func TestAllNotesAreNewestFirst(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("no notes")
	}
	if all[0].Version != Latest().Version {
		t.Errorf("All() is not newest-first: starts with %s, Latest is %s", all[0].Version, Latest().Version)
	}
}
