package keyboards

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// 🔴 Spec 21.09 §C1: every screen has an explanation; the screens that do not
// yet are named here, and this list may only get SHORTER.
//
// A «screen» is an exported constructor in this package that returns an inline
// keyboard — the set that grows by itself when someone adds a screen, which a
// hand-kept list of screen names would not. A constructor «has help» when its
// body calls HelpButton, directly or through another function of this package
// that does (ConvertCancel → convertCancelOn → HelpButton).
//
// The pin: len(screensWithoutHelp) must equal maxWithoutHelp exactly. Adding a
// line makes it longer and fails; giving a listed screen its «ℹ️» fails until
// the line is removed and the number lowered — so the list tracks the code.
const maxWithoutHelp = 65

var screensWithoutHelp = []string{
	"AgendaActions", "BackToList", "BackToMenu", "BackToTasks", "BroadcastConfirm",
	"CalendarCard", "CalendarColours", "CalendarConfirm", "CalendarInvite", "CalendarList",
	"CalendarPicker", "ColourPicker", "CreateMenu", "DatePicker", "DayActions",
	"DraftBack", "DraftCard", "DraftCardInList", "DraftDay", "DraftEditMenu",
	"EventCard", "EventDeleteConfirm", "EventEditor", "EventPicker", "FeedbackKinds",
	"FreqPicker", "GuideMore", "HelpBack", "HomeInlineFor", "Linking",
	"ListCard", "ListConfirm", "ListPick", "ManyDates", "MonthGrid",
	"None", "OnboardNext", "QuickAddKind", "ReminderPicker", "RepeatEndPicker",
	"RestartNewEvent", "SettingsMenu", "SpanFix", "StatsNav", "StatsScale",
	"TaskActions", "TaskCard", "TaskDue", "TaskEstimate", "TaskListEmpty",
	"TaskListItem", "TaskScheduleDuration", "TaskScheduleWhen", "TodayScreen", "TriggerCancel",
	"TriggerColourPicker", "TriggerFieldPicker", "TriggerFreqPicker", "WhatsNew", "WizardDue",
	"WizardEstimate", "WizardPriority", "WizardRepeat", "WorkHoursEnd", "WorkHoursStart",
}

func TestEveryScreenHasHelpOrIsAKnownException(t *testing.T) {
	screens, hasHelp := scanScreens(t)

	listed := map[string]bool{}
	for _, name := range screensWithoutHelp {
		if listed[name] {
			t.Errorf("%s is listed twice", name)
		}
		listed[name] = true
	}

	var missing []string
	for name := range screens {
		if !hasHelp[name] && !listed[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		t.Errorf("%s is a screen with no «ℹ️ Что это?» — give it one (HelpButton + a text in handlers.helpText)", name)
	}
	for _, name := range screensWithoutHelp {
		switch {
		case !screens[name]:
			t.Errorf("%s is listed as a screen without help, but no such screen exists — remove the line", name)
		case hasHelp[name]:
			t.Errorf("%s has its «ℹ️» now — remove it from screensWithoutHelp and lower maxWithoutHelp", name)
		}
	}
	if len(screensWithoutHelp) != maxWithoutHelp {
		t.Errorf("screensWithoutHelp has %d lines, the pin says %d — the list may only shrink, and the pin with it",
			len(screensWithoutHelp), maxWithoutHelp)
	}
}

// scanScreens parses this package and returns its screens and which of the
// package's functions reach HelpButton.
func scanScreens(t *testing.T) (screens, hasHelp map[string]bool) {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no source to scan: %v", err)
	}
	fset := token.NewFileSet()
	calls := map[string]map[string]bool{}
	screens = map[string]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			name := fn.Name.Name
			called := map[string]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					if id, ok := c.Fun.(*ast.Ident); ok {
						called[id.Name] = true
					}
				}
				return true
			})
			calls[name] = called
			if ast.IsExported(name) && returnsInlineKeyboard(fn) {
				screens[name] = true
			}
		}
	}
	if len(screens) < 40 {
		t.Fatalf("found only %d screens — the scan is not reading this package", len(screens))
	}

	hasHelp = map[string]bool{"HelpButton": true}
	for changed := true; changed; {
		changed = false
		for name, called := range calls {
			if hasHelp[name] {
				continue
			}
			for c := range called {
				if hasHelp[c] {
					hasHelp[name], changed = true, true
					break
				}
			}
		}
	}
	return screens, hasHelp
}

func returnsInlineKeyboard(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, r := range fn.Type.Results.List {
		if sel, ok := r.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "InlineKeyboardMarkup" {
			return true
		}
	}
	return false
}
