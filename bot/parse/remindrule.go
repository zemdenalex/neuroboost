package parse

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// RemindRule is «напоминать регулярно до срока» written in the line (Denis,
// 07.11: «событие 31.11 экзамен напоминать каждый день в 12 и 16»). Same shape
// as the API's remindrule.Rule: EveryMinutes, or EveryDays with Times.
type RemindRule struct {
	EveryMinutes int      `json:"every_minutes,omitempty"`
	EveryDays    int      `json:"every_days,omitempty"`
	Times        []string `json:"times,omitempty"`
}

// DefaultRuleTime is the time of a day-rule written without one
// («напоминать каждый день»).
const DefaultRuleTime = "10:00"

// The clause needs its opener — «напоминать», «напоминай», «напомни»,
// «remind». Without it «каждый день» is a REPEAT, as it always was; with it the
// same words say how often to remind. Everything from the opener to the end of
// the rule is cut out of the line BEFORE any other pass runs, which is what
// keeps «в 12 и 16» from being read as the event's start (the class of
// aec00dc: a later pass taking words an earlier meaning owned).
var (
	ruleOpener = `(?:напоминать|напоминай|напомни(?:ть)?|remind(?:\s+me)?)`
	ruleTime   = `\d{1,2}(?:[:.]\d{2})?`
	// Without «в»/«at» only a colon time counts: a bare «12.12» after the rule
	// is a DATE, not 12:12 (review 07.11), while «каждый день 12 и 16» still
	// reads as two times. The trailing boundary (no «.») is what keeps «12» of
	// «12.12» from matching on its own.
	bareTime = `\d{1,2}(?::\d{2})?`
	// Each alternative carries its own end: after «в 12 и 16» a full stop may
	// end the sentence; after bare numbers it may not, or «12.12» would match.
	ruleTimes = `(?:\s+(?:в|at)\s+(` + ruleTime + `(?:\s*(?:,|и|and)\s*` + ruleTime + `)*)(?:\s|$|[,.])` +
		`|\s+(` + bareTime + `(?:\s*(?:,|и|and)\s*` + bareTime + `)*)(?:\s|$|,)` +
		`|(?:\s|$|[,.]))`

	// «каждые 5 минут», «каждый час», «каждые 2 часа», «every 5 minutes», «every hour»
	intervalRuleRe = regexp.MustCompile(`(?i)(?:^|\s)` + ruleOpener +
		`\s+(?:каждые|каждый|каждую|every)\s+(?:(\d{1,4})\s*)?` +
		`(минут[уы]?|мин|м|час(?:а|ов)?|ч|minutes?|mins?|hours?|h)(?:\s|$|[,.])`)

	// «каждый день [в 12 и 16]», «ежедневно», «через день», «каждые 3 дня»,
	// «every day [at 9]», «daily», «every 2 days»
	dayRuleRe = regexp.MustCompile(`(?i)(?:^|\s)` + ruleOpener +
		`\s+(каждый\s+день|ежедневно|через\s+день|каждые\s+(\d{1,2})\s+(?:дня|дней)|every\s+day|daily|every\s+(\d{1,2})\s+days)` +
		ruleTimes)

	ruleTimeRe = regexp.MustCompile(ruleTime)
)

// ExtractRemindRule finds the clause, returns the line without it and the
// rule. nil when the line has no such clause.
func ExtractRemindRule(line string) (string, *RemindRule) {
	if loc := intervalRuleRe.FindStringSubmatchIndex(line); loc != nil {
		n := 1
		if loc[2] >= 0 {
			n, _ = strconv.Atoi(line[loc[2]:loc[3]])
		}
		unit := strings.ToLower(line[loc[4]:loc[5]])
		if strings.HasPrefix(unit, "ч") || strings.HasPrefix(unit, "h") {
			n *= 60
		}
		return cut(line, loc[0], loc[1]), &RemindRule{EveryMinutes: n}
	}
	if loc := dayRuleRe.FindStringSubmatchIndex(line); loc != nil {
		r := &RemindRule{EveryDays: 1}
		word := strings.ToLower(line[loc[2]:loc[3]])
		switch {
		case loc[4] >= 0:
			r.EveryDays, _ = strconv.Atoi(line[loc[4]:loc[5]])
		case loc[6] >= 0:
			r.EveryDays, _ = strconv.Atoi(line[loc[6]:loc[7]])
		case strings.HasPrefix(word, "через"):
			r.EveryDays = 2
		}
		switch {
		case loc[8] >= 0:
			r.Times = ruleTimes2(line[loc[8]:loc[9]])
		case loc[10] >= 0:
			r.Times = ruleTimes2(line[loc[10]:loc[11]])
		}
		if len(r.Times) == 0 {
			r.Times = []string{DefaultRuleTime}
		}
		return cut(line, loc[0], loc[1]), r
	}
	return line, nil
}

func cut(line string, from, to int) string {
	return strings.Join(strings.Fields(line[:from]+" "+line[to:]), " ")
}

// ruleTimes2 reads «12 и 16», «9:30, 18» into sorted HH:MM, dropping what is
// not a clock time.
func ruleTimes2(s string) []string {
	seen := map[int]bool{}
	var mins []int
	for _, t := range ruleTimeRe.FindAllString(s, -1) {
		t = strings.ReplaceAll(t, ".", ":")
		h, m := t, "0"
		if i := strings.Index(t, ":"); i >= 0 {
			h, m = t[:i], t[i+1:]
		}
		hh, err1 := strconv.Atoi(h)
		mm, err2 := strconv.Atoi(m)
		if err1 != nil || err2 != nil || hh > 23 || mm > 59 {
			continue
		}
		if v := hh*60 + mm; !seen[v] {
			seen[v] = true
			mins = append(mins, v)
		}
	}
	sort.Ints(mins)
	out := make([]string, 0, len(mins))
	for _, v := range mins {
		out = append(out, fmt.Sprintf("%02d:%02d", v/60, v%60))
	}
	return out
}

// Describe says the rule the way the card shows it.
func (r RemindRule) Describe(ru bool) string {
	if r.EveryMinutes > 0 {
		n := r.EveryMinutes
		switch {
		case n%60 == 0 && ru:
			return fmt.Sprintf("каждые %d ч до срока", n/60)
		case n%60 == 0:
			return fmt.Sprintf("every %d h until due", n/60)
		case ru:
			return fmt.Sprintf("каждые %d мин до срока", n)
		default:
			return fmt.Sprintf("every %d min until due", n)
		}
	}
	times := strings.Join(r.Times, ", ")
	switch {
	case r.EveryDays == 1 && ru:
		return "каждый день в " + times + " до срока"
	case r.EveryDays == 1:
		return "every day at " + times + " until due"
	case ru:
		return fmt.Sprintf("раз в %d дн. в %s до срока", r.EveryDays, times)
	default:
		return fmt.Sprintf("every %d days at %s until due", r.EveryDays, times)
	}
}
