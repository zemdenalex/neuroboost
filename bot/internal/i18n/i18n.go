// Package i18n carries the bot's interface language.
//
// 🔴 Translations are PAIRS at the call site, not keys in a catalogue:
//
//	i18n.T(lang, "Что создаём?", "What are we creating?")
//
// With 346 strings, a catalogue means 346 keys that can go missing, and the
// only thing that would find a missing one is a test somebody has to write and
// keep complete. A pair cannot go missing: `T` takes two strings, so leaving
// out the English is a compile error. The enforcement is the signature.
//
// It also keeps the two languages side by side, where a reader can see that
// they still say the same thing. A catalogue puts them in different files and
// lets them drift for months.
package i18n

// Lang is a bot interface language. There are two.
type Lang string

const (
	RU Lang = "ru"
	EN Lang = "en"
)

// Default is what a chat gets before it has chosen, and what an unreadable
// setting falls back to.
//
// ⚠ Russian, because this bot's one user reads Russian and a bot that greets
// him in English on first contact is broken from his side even though every
// string is present.
const Default = RU

// Parse reads a stored setting.
func Parse(s string) Lang {
	switch Lang(s) {
	case EN:
		return EN
	case RU:
		return RU
	default:
		return Default
	}
}

// T picks the string for this language.
func T(l Lang, ru, en string) string {
	if l == EN {
		return en
	}
	return ru
}

// Name is how the language calls itself, for the settings screen. A language
// list written in the CURRENT language is useless to the person who cannot
// read it — which is the person choosing.
func Name(l Lang) string {
	if l == EN {
		return "English"
	}
	return "Русский"
}
