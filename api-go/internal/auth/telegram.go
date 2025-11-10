package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

// VerifyTelegramAuth implements https://core.telegram.org/widgets/login#checking-authorization
// authData must include fields from Telegram login (as strings), excluding "hash" which is provided separately.
func VerifyTelegramAuth(authData map[string]string, botToken string) (bool, error) {
	hashProvided, ok := authData["hash"]
	if !ok || hashProvided == "" {
		return false, errors.New("missing hash")
	}

	// Build data-check-string from all keys except "hash", sorted lexicographically.
	keys := make([]string, 0, len(authData))
	for k := range authData {
		if k == "hash" {
			continue
		}
		if authData[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		// Note: use exact values as received, do not URL-encode
		pairs = append(pairs, k+"="+authData[k])
	}
	dataCheckString := strings.Join(pairs, "\n")

	// Secret key = SHA256(bot_token), HMAC-SHA256(data-check-string, secret key)
	h := sha256.New()
	h.Write([]byte(botToken))
	secretKey := h.Sum(nil)

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	calculated := hex.EncodeToString(mac.Sum(nil))

	// Constant-time compare
	if !hmac.Equal([]byte(calculated), []byte(strings.ToLower(hashProvided))) &&
		!hmac.Equal([]byte(calculated), []byte(hashProvided)) {
		return false, errors.New("invalid hash")
	}

	// Recency check: auth_date < 24h
	adStr, ok := authData["auth_date"]
	if !ok || adStr == "" {
		return false, errors.New("missing auth_date")
	}
	ad, err := strconv.ParseInt(adStr, 10, 64)
	if err != nil {
		return false, errors.New("invalid auth_date")
	}
	if time.Since(time.Unix(ad, 0)) > 24*time.Hour {
		return false, errors.New("auth_date too old")
	}

	return true, nil
}
