package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// What the user is allowed to see when the API says no.
//
// 🔴 Denis, 21.09, pressing «✅ На сегодня» in a Telegram chat:
//
//	❌ Не получилось: API error 400: {"error":{"code":"NOT_AN_OCCURRENCE",
//	"message":"That day is not in the series"}}
//
// Twenty handlers print `err.Error()` straight to the chat, so the JSON was
// never going to be fixed screen by screen — it had to stop being produced.
// The envelope is parsed HERE, where it arrives, and what travels onward is a
// value with a code a handler can branch on and a sentence a person can read.
//
// ⚠ The API's own `message` is English by contract. Rendering for a Russian
// chat is the handlers' job (`errorText`), not this package's: `api` knows
// nothing about who is reading.

// Error is a refusal the API explained.
type Error struct {
	Status  int    // HTTP status
	Code    string // the API's machine code, e.g. NOT_AN_OCCURRENCE
	Message string // the API's English sentence
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Code != "" {
		return e.Code
	}
	return fmt.Sprintf("%d %s", e.Status, http.StatusText(e.Status))
}

// CodeOf reports the API code behind an error, or "" when it was not one of
// ours — a timeout, a DNS failure, a proxy page. Callers branch on this rather
// than on message text, which changes whenever someone improves the wording.
func CodeOf(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ""
}

// errorFromBody reads the `{"error":{"code","message"}}` envelope every handler
// in api-go produces through util.RespondError.
//
// A body that is not that envelope — a proxy's HTML, an empty 502 — must not
// become the user's error text: it would put a page of markup into a chat
// message. Such a body is dropped and the status speaks alone.
func errorFromBody(status int, body []byte) error {
	e := &Error{Status: status}

	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if len(body) > 0 && json.Unmarshal(body, &envelope) == nil {
		e.Code = envelope.Error.Code
		e.Message = envelope.Error.Message
	}
	return e
}
