package api

// BotSetting reads one key of the settings' "bot" section; "" when absent.
func (c *Client) BotSetting(token, key string) (string, error) {
	settings, err := c.MySettings(token)
	if err != nil {
		return "", err
	}
	bot, _ := settings["bot"].(map[string]any)
	v, _ := bot[key].(string)
	return v, nil
}

// SetBotSetting writes one key of the "bot" section and keeps every other.
//
// Rebuilt key by key, the way SetBotLang does it: MergeSettings is shallow,
// so patching {"bot": …} replaces the whole section — language and keyword
// vocabulary included.
func (c *Client) SetBotSetting(token, key, value string) error {
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
	bot[key] = value
	_, err = c.PatchSettings(token, map[string]any{"bot": bot})
	return err
}
