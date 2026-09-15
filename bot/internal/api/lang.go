package api

import "strings"

// The interface language lives beside the keyword vocabulary, in the settings
// blob under `bot.lang`.
//
// 🔴 Same discipline as every other write to that blob (gotcha 21): read, merge,
// write — and never write when the read failed. PATCH /api/auth/me REPLACES the
// settings object, so a language written onto a blob we could not see erases
// work hours, quiet hours, reminder presets and every keyword, and answers 200.

// BotLang reads the stored interface language, or "" when none is set.
//
// A malformed section is "" rather than an error: the language is a preference,
// and a bot that refuses to answer because it could not read a preference is
// worse than one that answers in the default.
func (c *Client) BotLang(token string) (string, error) {
	settings, err := c.MySettings(token)
	if err != nil {
		return "", err
	}
	bot, ok := settings["bot"].(map[string]any)
	if !ok {
		return "", nil
	}
	lang, _ := bot["lang"].(string)
	return strings.ToLower(strings.TrimSpace(lang)), nil
}

// SetBotLang stores the interface language.
func (c *Client) SetBotLang(token, lang string) error {
	settings, err := c.MySettings(token)
	if err != nil {
		return err
	}

	// Rebuilt key by key rather than replaced: MergeSettings is shallow, so
	// patching {"bot": …} overwrites every other key the bot section holds —
	// and the keyword vocabulary lives in there.
	bot := map[string]any{}
	if existing, ok := settings["bot"].(map[string]any); ok {
		for k, v := range existing {
			bot[k] = v
		}
	}
	bot["lang"] = strings.ToLower(strings.TrimSpace(lang))

	_, err = c.PatchSettings(token, map[string]any{"bot": bot})
	return err
}
