package release

import (
	"strconv"
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

// Newest-first is checked on the numbers, not trusted to the order they were
// typed in: /broadcast sends Latest(), i.e. notes[0].
//
// ⚠ What this does NOT catch — the 22.09 case. A «v0.4.11.5» written on 18.09
// sat on top, correctly ordered, describing account linking that had in fact
// shipped silently inside v0.4.11.3. Whether the top note is the release being
// cut is a question for the person cutting it; the spec says so (§D).
func TestVersionsStrictlyDescend(t *testing.T) {
	all := All()
	for i := 1; i < len(all); i++ {
		if !versionLess(all[i].Version, all[i-1].Version) {
			t.Errorf("%s is listed below %s but is not older", all[i].Version, all[i-1].Version)
		}
	}
}

// Only one release offers the priority symbol: the one that made it a choice.
func TestOnlyOneReleaseOffersThePrioritySymbol(t *testing.T) {
	var n int
	for _, note := range All() {
		if note.OfferPriority {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d releases offer the priority symbol, want exactly 1", n)
	}
}

func versionLess(a, b string) bool {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x < y
		}
	}
	return false
}
