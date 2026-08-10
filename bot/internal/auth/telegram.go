// Package auth lets the bot obtain a JWT for the Telegram user it is talking
// to, so commands like /today act as that user instead of as nobody.
//
// It reuses the existing Telegram Login Widget endpoint (POST /api/auth/telegram)
// rather than adding a bot-only login route. The widget's signature is exactly
// the assertion the bot is entitled to make — "this Telegram user is who they
// say they are" — and the proof is an HMAC keyed by the bot token, which the
// bot already holds. A dedicated service route would have meant a second
// endpoint able to mint a JWT for any user id, guarded only by a shared secret
// that lives on a foreign host.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// LoginRequest is the payload POST /api/auth/telegram expects. The omitempty
// tags matter for more than wire size: the server rebuilds the data-check
// string from the fields it received, so a field sent as "" would be hashed by
// one side and skipped by the other.
type LoginRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// BuildLoginRequest signs the identity fields the way Telegram's Login Widget
// does: sorted "key=value" lines joined by newlines, HMAC-SHA256 under
// SHA256(botToken).
//
// 🔴 Empty optional fields are omitted, not sent blank. The server's verifier
// builds its check string only from non-empty values, so including an empty
// last_name here would produce a hash that can never match.
func BuildLoginRequest(botToken string, id int64, firstName, lastName, username, photoURL string, authDate int64) LoginRequest {
	req := LoginRequest{
		ID:        id,
		FirstName: firstName,
		LastName:  lastName,
		Username:  username,
		PhotoURL:  photoURL,
		AuthDate:  authDate,
	}
	req.Hash = sign(botToken, req)
	return req
}

func sign(botToken string, req LoginRequest) string {
	data := map[string]string{
		"id":         strconv.FormatInt(req.ID, 10),
		"first_name": req.FirstName,
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
	}
	if req.LastName != "" {
		data["last_name"] = req.LastName
	}
	if req.Username != "" {
		data["username"] = req.Username
	}
	if req.PhotoURL != "" {
		data["photo_url"] = req.PhotoURL
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, data[k]))
	}

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}
