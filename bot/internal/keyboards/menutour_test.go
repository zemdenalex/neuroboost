package keyboards

import (
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// N1 of pass 3 (docs/tasks-prohod3-2026-09-23.md, Denis 23.09): onboarding
// explains the main menu, «или хотя бы кнопку что это». Explained by buttons,
// not by prose that sends people to look for one (TestNoScreenPointsAtAReplyButton):
// each entrance of the reply keyboard, with a line on what it is, as a button
// that opens it.
func TestTheMenuTourCoversEveryEntranceInBothLanguages(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.RU, i18n.EN} {
		text, kb := MenuTour(lang)
		screens := map[string]bool{}
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil && strings.HasPrefix(*b.CallbackData, MenuTourPrefix) {
					screens[strings.TrimPrefix(*b.CallbackData, MenuTourPrefix)] = true
				}
			}
		}
		for _, row := range menuRows {
			for _, e := range row {
				if !screens[e.Screen] {
					t.Errorf("%s: no button opens %q", lang, e.Screen)
				}
				if !strings.Contains(text, e.label(lang)) {
					t.Errorf("%s: the tour does not name %q", lang, e.label(lang))
				}
				if strings.TrimSpace(i18n.T(lang, e.WhatRU, e.WhatEN)) == "" {
					t.Errorf("%s: %q has no line saying what it is", lang, e.label(lang))
				}
			}
		}
		if len(screens) != 6 {
			t.Errorf("%s: %d entrance buttons, want the menu's 6", lang, len(screens))
		}
	}
}

func TestOnboardingEndsWithAWayIntoTheMenuTour(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.RU, i18n.EN} {
		found := false
		for _, row := range OnboardNext(lang).InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil && *b.CallbackData == MenuTourOpen {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%s: the last onboarding screen has no «what is in the menu» button", lang)
		}
	}
}
