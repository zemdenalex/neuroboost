package handlers

import (
	"strings"
	"time"
)

// A recurring event does not exist once. GET /api/events EXPANDS it and stamps
// each occurrence with a synthetic id of the form `<uuid>:YYYY-MM-DD`
// (api-go/internal/events/instanceid.go).
//
// 🔴 And GET /api/events/{id} does not understand that form. It passes the id
// straight into a uuid cast, so asking for an occurrence by the id the list
// just handed out fails — which is exactly what Denis hit on 16.09:
// «С повторяющимися пишет ❌ Не удалось открыть событие».
//
// 🔴 What the bot does about it, and why this and not something cleverer.
// Every call — GET, PATCH, DELETE — uses the PARENT id, so the bot edits the
// SERIES and nothing else. The alternative is passing the instance id through,
// which the API reads as scope=occurrence, and that opens two traps at once: a
// calendar change is refused outright with a 400, and a time change written as
// an absolute value would move the series rather than the occurrence. Editing
// one occurrence of a repeat is a real feature and it needs its own screen and
// its own question; pretending to offer it from a card that says «вся серия»
// would be the button that lies.

// instanceDateLayout matches what the API stamps.
const instanceDateLayout = "2006-01-02"

// splitInstanceID separates a recurring occurrence's id from its parent.
//
// Anything not of that shape comes back unchanged with isInstance false, so
// every caller can funnel ids through this without asking first.
//
// ⚠ Split on the LAST colon: a UUID contains none, and a date does not either,
// but a future id format might.
func splitInstanceID(id string) (parentID, occurrence string, isInstance bool) {
	idx := strings.LastIndex(id, ":")
	if idx < 0 {
		return id, "", false
	}
	tail := id[idx+1:]
	if _, err := time.Parse(instanceDateLayout, tail); err != nil {
		return id, "", false
	}
	return id[:idx], tail, true
}
