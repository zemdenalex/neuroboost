package keyboards

import "testing"

// 🔴 The rule: every entrance on the reply keyboard is also reachable from the
// home screen.
//
// Denis, 23.08: «странно что в меню обычном нету тех же кнопок или невозможно
// добраться до кнопок, которые есть в клавиатурном меню». Календарь, События and
// Задачи existed ONLY as reply buttons — and a reply keyboard can be collapsed,
// at which point half the product had no way in.
//
// The rule that produced that hole was satisfied literally: "no screen tells the
// user to press a reply button" says nothing about a screen quietly DEPENDING on
// one. A rule stated as a prohibition needs its positive half written down too,
// which is what this test is.
//
// entranceRoutes is the decision, stated literally rather than derived from the
// code under test: a test that computes its expectation from its subject cannot
// disagree with it.
var entranceRoutes = map[string]string{
	"🗓 Календарь":  "cal_open",
	"📅 События":    "agenda_open",
	"📋 Задачи":     "top_tasks",
	"➕ Создать":    "create_menu",
	"⚙️ Настройки": "settings_menu",
}

// homeEntrance is the one reply button with no inline twin: it opens the very
// screen the inline buttons live on.
const homeEntrance = "🏠 Меню"

func TestEveryReplyEntranceHasAnInlineWayIn(t *testing.T) {
	home := map[string]bool{}
	for _, row := range HomeInline().InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != nil {
				home[*b.CallbackData] = true
			}
		}
	}
	if len(home) == 0 {
		t.Fatal("HomeInline carries no callback buttons — the test is checking nothing")
	}

	seen := 0
	for _, row := range MainMenu().Keyboard {
		for _, b := range row {
			if b.Text == homeEntrance {
				continue
			}
			seen++
			route, known := entranceRoutes[b.Text]
			if !known {
				// A new reply button with no declared route: whoever added it
				// has to say where the inline twin leads.
				t.Errorf("reply button %q has no route in entranceRoutes — "+
					"add it, and give HomeInline a button carrying that callback", b.Text)
				continue
			}
			if !home[route] {
				t.Errorf("reply button %q leads somewhere the home screen cannot reach "+
					"(no inline button with callback_data %q). Collapse the reply keyboard "+
					"and that entrance is gone.", b.Text, route)
			}
		}
	}
	if seen != len(entranceRoutes) {
		t.Errorf("checked %d reply entrances but entranceRoutes declares %d — "+
			"the two lists have drifted apart", seen, len(entranceRoutes))
	}
}
