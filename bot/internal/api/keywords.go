package api

import "strings"

// The user's own keyword vocabulary, Denis's ask of 15.09, in two halves:
// «также должна быть возможность создавать свои слова для тегов», and then,
// after the first pass shipped: «Ключевые слова не обязательно теги, они должны
// быть как триггеры… выбирается что оно обозначает/заменяет то есть какую
// характеристику (тег/дата/цвет/календарь и тд)».
//
// It lives in the settings blob under `bot.keywords`, as word → {field, value}.
//
// 🔴 Everything here obeys gotcha 21. PATCH /api/auth/me REPLACES the whole
// settings blob rather than merging into it, so a naive write of
// {"bot": {...}} erases work hours, quiet hours and reminder presets, and
// answers 200 while doing it. Two rules follow, and both are load-bearing:
// read-merge-write, and never write when the read failed — merging onto an
// empty map is how everything disappears at once.

// Keyword is one defined word: which characteristic it sets, and to what.
//
// Field is the stored NAME of the characteristic ("tag", "colour", "day"…),
// not a parse.Field: this package must not depend on the parser, and a string
// in the settings blob survives a renumbering of the enum.
type Keyword struct {
	Field string
	Value string
}

// BotKeywords reads the user's own vocabulary.
//
// A missing or malformed section is an empty vocabulary, not an error: the bot
// must keep working for someone who never defined a word, and for someone whose
// blob was written by a client that shaped it differently.
func (c *Client) BotKeywords(token string) (map[string]Keyword, error) {
	settings, err := c.MySettings(token)
	if err != nil {
		return nil, err
	}
	return keywordsFrom(settings), nil
}

// keywordsFrom reads both shapes the blob may hold.
//
// ⚠ A bare string is the FIRST shape this feature shipped with, where every
// word was a tag. It is still read as one. Dropping it would silently empty the
// vocabulary of anyone who used the version that stored it — and "silently" is
// the part that makes it unacceptable, not "empty".
func keywordsFrom(settings map[string]any) map[string]Keyword {
	out := map[string]Keyword{}
	bot, ok := settings["bot"].(map[string]any)
	if !ok {
		return out
	}
	words, ok := bot["keywords"].(map[string]any)
	if !ok {
		return out
	}

	for word, raw := range words {
		w := strings.ToLower(strings.TrimSpace(word))
		if w == "" {
			continue
		}
		switch v := raw.(type) {
		case string:
			if s := strings.ToLower(strings.TrimSpace(v)); s != "" {
				out[w] = Keyword{Field: "tag", Value: s}
			}
		case map[string]any:
			field, _ := v["field"].(string)
			value, _ := v["value"].(string)
			field = strings.ToLower(strings.TrimSpace(field))
			if field == "" {
				continue
			}
			out[w] = Keyword{Field: field, Value: strings.TrimSpace(value)}
		}
	}
	return out
}

// SetBotKeyword adds or replaces one word. An empty field removes it.
//
// 🔴 The read is not a formality. If MySettings fails this returns without
// writing anything: the alternative is patching a blob we could not see, which
// merges the new word onto an empty map and deletes the rest of the user's
// settings.
func (c *Client) SetBotKeyword(token, word, field, value string) error {
	settings, err := c.MySettings(token)
	if err != nil {
		return err
	}

	words := map[string]any{}
	if bot, ok := settings["bot"].(map[string]any); ok {
		if existing, ok := bot["keywords"].(map[string]any); ok {
			for k, v := range existing {
				words[k] = v
			}
		}
	}

	key := strings.ToLower(strings.TrimSpace(word))
	if key == "" {
		return nil
	}
	if strings.TrimSpace(field) == "" {
		delete(words, key)
	} else {
		words[key] = map[string]any{
			"field": strings.ToLower(strings.TrimSpace(field)),
			"value": strings.TrimSpace(value),
		}
	}

	// The `bot` section is rebuilt from what was there, not replaced:
	// MergeSettings is shallow, so patching {"bot": …} overwrites every other
	// key the bot section may hold.
	bot := map[string]any{}
	if existing, ok := settings["bot"].(map[string]any); ok {
		for k, v := range existing {
			bot[k] = v
		}
	}
	bot["keywords"] = words

	_, err = c.PatchSettings(token, map[string]any{"bot": bot})
	return err
}

// ReminderPresets reads the user's named reminder presets — `reminders.presets`
// in the settings blob, as name → offsets in minutes.
//
// ⚠ A preset naming an EMPTY list is a real answer, not a missing one: «без»
// means stay silent. So an empty slice is kept, and only a value of the wrong
// shape is skipped.
func (c *Client) ReminderPresets(token string) (map[string][]int, error) {
	settings, err := c.MySettings(token)
	if err != nil {
		return nil, err
	}
	return presetsFrom(settings), nil
}

func presetsFrom(settings map[string]any) map[string][]int {
	out := map[string][]int{}
	reminders, ok := settings["reminders"].(map[string]any)
	if !ok {
		return out
	}
	presets, ok := reminders["presets"].(map[string]any)
	if !ok {
		return out
	}
	for name, raw := range presets {
		list, ok := raw.([]any)
		if !ok {
			continue
		}
		offsets := make([]int, 0, len(list))
		bad := false
		for _, v := range list {
			n, ok := v.(float64) // every number out of encoding/json is a float64
			if !ok {
				bad = true
				break
			}
			offsets = append(offsets, int(n))
		}
		if bad {
			continue
		}
		if n := strings.ToLower(strings.TrimSpace(name)); n != "" {
			out[n] = offsets
		}
	}
	return out
}
