package keyboards

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// One symbol, one meaning, across every keyboard in the bot.
//
// 🔴 Denis, 18.09: «отмена тоже где-то корзина, где-то крестик, давай лучше
// тогда везде отмена это красный крест, а удалить это корзина, и там где отмена
// удаляет черновик или что-то тогда и писать удалить, а не отменить».
//
// The rule is not decoration. A bin that sometimes means "step back" and
// sometimes means "destroy" has to be read every time; a symbol that means one
// thing can be obeyed without reading. And the dangerous direction is
// asymmetric: a cross that quietly deletes is worse than a bin that quietly
// goes back, so both pairings are refused here rather than only one.
//
// This is a scan of the source, which is the only form that can cover every
// button — there are forty-odd across nine files, and a test that pressed them
// would need a Telegram for each.
func TestCancelIsACrossAndDeleteIsABin(t *testing.T) {
	label := regexp.MustCompile(`"([^"]*)"`)

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(body), "\n") {
			if !strings.Contains(line, "NewInlineKeyboardButtonData") &&
				!strings.Contains(line, "i18n.T(lang,") {
				continue
			}
			for _, m := range label.FindAllStringSubmatch(line, -1) {
				text := m[1]
				bin := strings.Contains(text, "🗑")
				cross := strings.Contains(text, "❌")
				if !bin && !cross {
					continue
				}
				cancels := strings.Contains(strings.ToLower(text), "отмен") ||
					strings.Contains(strings.ToLower(text), "cancel")
				deletes := strings.Contains(strings.ToLower(text), "удал") ||
					strings.Contains(strings.ToLower(text), "delete")

				if bin && cancels {
					t.Errorf("%s:%d — 🗑 on a button that says «отмена»: %q\n"+
						"    a bin means destroy; if it destroys, the word must say so", f, i+1, text)
				}
				if cross && deletes {
					t.Errorf("%s:%d — ❌ on a button that deletes: %q\n"+
						"    a cross means step back; destroying wants the bin", f, i+1, text)
				}
			}
		}
	}
}
