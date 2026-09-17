package parse

import (
	"regexp"
	"strconv"
	"strings"
)

// Reminders in words. Denis, item 4: «пресеты уведомлений тоже должна быть
// возможность выбрать с помощью кнопки или просто словом».
//
// Two forms, and they are separate functions because they need different
// things: an offset is readable from the line alone, a preset name exists only
// in the user's settings.

var remindOpeners = map[string]bool{
	"напомнить": true, "напоминание": true, "напомни": true,
	"remind": true, "reminder": true,
}

// remindOffset matches «15м», «30мин», «1ч», «2часа», «1д», «1день».
var remindOffset = regexp.MustCompile(`^(\d{1,4})(м|мин|минут|минуты|ч|час|часа|часов|д|день|дня|дней|m|min|h|hour|d|day)$`)

// RecogniseReminderOffset claims «напомнить за 15м» and writes the offset in
// minutes.
//
// 🔴 The result is a pointer. POST /api/events applies the user's default
// preset when reminder_offsets is ABSENT and stays silent forever when it is an
// empty array — "not stated" and "stated as none" are different events, and a
// plain slice cannot hold the difference.
func RecogniseReminderOffset(toks []Token, d *Draft) bool {
	for i, t := range toks {
		if t.Field != FieldNone || !remindOpeners[t.Norm] {
			continue
		}

		j := i + 1
		// «за» is optional: «напомнить за 15м» and «напомнить 15м» both read.
		if j < len(toks) && toks[j].Field == FieldNone && (toks[j].Norm == "за" || toks[j].Norm == "in") {
			j++
		}
		if j >= len(toks) || toks[j].Field != FieldNone {
			continue
		}

		minutes, ok := offsetMinutes(toks[j].Norm)
		// «за 2 часа» — the number and the unit typed as two words.
		if !ok && j+1 < len(toks) && toks[j+1].Field == FieldNone {
			if m, joined := offsetMinutes(toks[j].Norm + toks[j+1].Norm); joined {
				minutes, ok = m, true
				j++
			}
		}
		// «за час», «за день» — the unit alone means one of it.
		if !ok {
			minutes, ok = offsetWords[toks[j].Norm]
		}
		if !ok {
			continue
		}
		offsets := []int{minutes}
		d.ReminderOffsets = &offsets
		for k := i; k <= j; k++ {
			toks[k].Field = FieldReminder
		}
		return true
	}
	return false
}

// offsetWords are units written without a number: «напомни за час».
var offsetWords = map[string]int{
	"час": 60, "полчаса": 30, "день": 1440, "сутки": 1440, "неделю": 10080,
	"hour": 60, "day": 1440, "week": 10080,
}

func offsetMinutes(norm string) (int, bool) {
	m := remindOffset.FindStringSubmatch(norm)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return 0, false
	}
	switch {
	case strings.HasPrefix(m[2], "ч"), strings.HasPrefix(m[2], "h"):
		n *= 60
	case strings.HasPrefix(m[2], "д"), strings.HasPrefix(m[2], "d"):
		n *= 60 * 24
	}
	return n, true
}

// RecogniseReminderPreset claims a word that names one of the user's own
// reminder presets.
//
// Like the calendar pass, it lives outside the pipeline because it needs data
// the parser cannot have, and it matches exactly for the same reason: a preset
// called «важное» must not turn every «важное» in a title into a setting.
func RecogniseReminderPreset(toks []Token, presets map[string][]int, d *Draft) bool {
	if len(presets) == 0 {
		return false
	}
	norm := make(map[string][]int, len(presets))
	for name, offsets := range presets {
		if n := strings.ToLower(strings.TrimSpace(name)); n != "" {
			norm[n] = offsets
		}
	}

	for i, t := range toks {
		if t.Field != FieldNone {
			continue
		}
		offsets, ok := norm[t.Norm]
		if !ok {
			continue
		}
		// A preset naming an empty list is a real answer — «без» means silent
		// — so the slice is copied rather than tested for emptiness.
		chosen := append([]int(nil), offsets...)
		d.ReminderOffsets = &chosen
		toks[i].Field = FieldReminder
		return true
	}
	return false
}

// ReminderOffsetText reads a reminder typed on its own — the «✏️ Своё время»
// step: «2ч», «45 минут», «за 2 часа», «за день», «in 1h».
//
// The same vocabulary as «напомнить за …» inside a line, so a time that works
// in one place works in the other.
func ReminderOffsetText(text string) (int, bool) {
	toks := Tokenize("напомнить " + text)
	var d Draft
	if !RecogniseReminderOffset(toks, &d) || d.ReminderOffsets == nil || len(*d.ReminderOffsets) != 1 {
		return 0, false
	}
	// Anything left over means the text was more than a reminder time, and a
	// guess at which part was meant is not an answer.
	if Title(toks) != "" {
		return 0, false
	}
	return (*d.ReminderOffsets)[0], true
}
