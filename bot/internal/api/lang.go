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
	lang = strings.ToLower(strings.TrimSpace(lang))
	bot["lang"] = lang

	// Only the bot's own language. The account's `locale` belongs to the web
	// and the Mini App (Denis 26.09: «sync web interfaces (web, miniapp) but
	// bot and later android app should stay in their own language»).
	_, err = c.PatchSettings(token, map[string]any{"bot": bot})
	return err
}

// MyTimezone reads the account's IANA timezone.
//
// It is a COLUMN on the user row, not a key in the settings blob — so writing
// it (SetTimezone) cannot erase settings the way a partial settings PATCH does.
func (c *Client) MyTimezone(token string) (string, error) {
	var resp struct {
		Data struct {
			Timezone string `json:"timezone"`
		} `json:"data"`
	}
	if err := c.get("/api/auth/me", token, nil, &resp); err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Data.Timezone), nil
}

// SetTimezone writes the account's timezone. Only the timezone field is sent:
// the API updates the columns it is given and leaves `settings` untouched
// unless that key is present (auth/handlers.go, UpdateMeHandler).
func (c *Client) SetTimezone(token, tz string) error {
	return c.patch("/api/auth/me", token, map[string]any{"timezone": tz}, nil)
}

// Onboarded reports whether this user has finished (or skipped) onboarding.
func (c *Client) Onboarded(token string) (bool, error) {
	settings, err := c.MySettings(token)
	if err != nil {
		return false, err
	}
	bot, _ := settings["bot"].(map[string]any)
	done, _ := bot["onboarded"].(bool)
	return done, nil
}

// SetOnboarded records that onboarding is over. Same read-merge-write as
// SetBotLang: the bot section also holds the keyword vocabulary and the
// language, and MergeSettings is shallow.
func (c *Client) SetOnboarded(token string) error {
	settings, err := c.MySettings(token)
	if err != nil {
		return err
	}
	bot := map[string]any{}
	if existing, ok := settings["bot"].(map[string]any); ok {
		for k, v := range existing {
			bot[k] = v
		}
	}
	bot["onboarded"] = true
	_, err = c.PatchSettings(token, map[string]any{"bot": bot})
	return err
}
