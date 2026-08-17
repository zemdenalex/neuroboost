// Package logsafe keeps secrets out of anything this process writes down.
//
// The bot's token is part of every Telegram API URL, and the client library
// returns transport failures as *url.Error, whose Error() prints that URL in
// full: `Post "https://api.telegram.org/bot<TOKEN>/getMe": dial tcp …`. Any
// `log.Printf("...: %v", err)` therefore writes the token to disk.
//
// That is not a hypothetical path on this deployment. The comment in
// cmd/main.go says plainly that Russian hosting blocks Telegram IPv4 — a
// refused connection is an EXPECTED state here, which is why TELEGRAM_PROXY
// exists at all. And the notifier ticks once a minute, so a network outage
// wrote the token to the log every sixty seconds.
//
// The project's rule for READING bot logs already applies the same regexp
// through sed. That protects the reader; it does nothing for the file on disk,
// for `docker logs` shipped elsewhere, or for a log aggregator. This does.
package logsafe

import "regexp"

// tokenPattern matches a Telegram bot token as it appears inside an API URL:
// numeric bot id, colon, then the secret. Deliberately anchored on the `bot`
// prefix from the URL path so ordinary text containing a colon is untouched.
var tokenPattern = regexp.MustCompile(`bot[0-9]+:[A-Za-z0-9_-]+`)

// Redact returns err's message with any bot token replaced.
//
// Returns "<nil>" for a nil error rather than panicking: this is called on
// logging paths, and a logging helper that can crash the process is worse than
// the leak it prevents.
func Redact(err error) string {
	if err == nil {
		return "<nil>"
	}
	return String(err.Error())
}

// String redacts an arbitrary message. Used where the token could reach a log
// through something other than an error value.
func String(s string) string {
	return tokenPattern.ReplaceAllString(s, "bot<REDACTED>")
}
