package statgrid

import (
	"strconv"
	"strings"
	"time"
)

// SeriesDays counts the days a repeating task was due in [from, to) — the M of
// «Серии: N из M». The rule grammar is the bot's own (keyboards.RepeatCodes):
// DAILY/WEEKLY/MONTHLY with INTERVAL, COUNT and UNTIL. Anything else → 0: a
// guess would print a number nobody can check.
func SeriesDays(rrule, anchor string, from, to time.Time, loc *time.Location) int {
	a, err := time.Parse(time.RFC3339, anchor)
	if err != nil {
		return 0
	}
	start := midnight(a, loc)
	freq, interval, count := "", 1, 0
	var until time.Time
	for _, part := range strings.Split(rrule, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "FREQ":
			freq = kv[1]
		case "INTERVAL":
			if n, e := strconv.Atoi(kv[1]); e == nil && n > 0 {
				interval = n
			}
		case "COUNT":
			if n, e := strconv.Atoi(kv[1]); e == nil && n > 0 {
				count = n
			}
		case "UNTIL":
			if d, e := time.ParseInLocation("2006-01-02", kv[1][:min(10, len(kv[1]))], loc); e == nil {
				until = d.AddDate(0, 0, 1) // inclusive day
			}
		}
	}
	step := func(i int) time.Time {
		switch freq {
		case "DAILY":
			return start.AddDate(0, 0, i*interval)
		case "WEEKLY":
			return start.AddDate(0, 0, 7*i*interval)
		case "MONTHLY":
			return start.AddDate(0, i*interval, 0)
		}
		return time.Time{}
	}
	if step(0).IsZero() {
		return 0
	}
	n := 0
	for i := 0; ; i++ {
		d := step(i)
		if !d.Before(to) || (count > 0 && i >= count) || (!until.IsZero() && !d.Before(until)) {
			return n
		}
		if !d.Before(from) {
			n++
		}
	}
}
