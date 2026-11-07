package parse

import (
	"reflect"
	"testing"
	"time"
)

// Denis's line, 07.11, verbatim. Three claims in one line:
//   - «напоминать каждый день в 12 и 16» is the reminder rule, and its
//     «в 12» is NOT the event's start time;
//   - «31.11» does not exist: not rolled to 01.12, cut from the title, flagged;
//   - the title is what is left: «экзамен».
func TestDenisLineRuleAndAMissingDate(t *testing.T) {
	now := time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)
	p := ParseLine("событие 31.11 экзамен напоминать каждый день в 12 и 16", now)
	if p.Draft.RemindRule == nil || p.Draft.RemindRule.EveryDays != 1 ||
		!reflect.DeepEqual(p.Draft.RemindRule.Times, []string{"12:00", "16:00"}) {
		t.Errorf("rule = %+v, want every day at 12:00 and 16:00", p.Draft.RemindRule)
	}
	if p.Draft.HasTime {
		t.Errorf("«в 12» of the rule became the start time %v", p.Draft.Start)
	}
	if p.Draft.HasDay {
		t.Errorf("31.11 became the day %v — a date that does not exist was guessed", p.Draft.Day)
	}
	if p.Draft.BadDate != "31.11" {
		t.Errorf("BadDate = %q, want 31.11", p.Draft.BadDate)
	}
	if p.Title != "экзамен" {
		t.Errorf("title = %q, want «экзамен»", p.Title)
	}
}

func TestRuleWordings(t *testing.T) {
	cases := map[string]RemindRule{
		"экзамен 30.11 напоминать каждые 5 минут":   {EveryMinutes: 5},
		"отчёт напоминай каждый час":                {EveryMinutes: 60},
		"отчёт remind every 2 hours":                {EveryMinutes: 120},
		"полить цветы напоминать через день":        {EveryDays: 2, Times: []string{DefaultRuleTime}},
		"виза напоминать каждые 3 дня в 9:30":       {EveryDays: 3, Times: []string{"09:30"}},
		"essay remind me every day at 18 and 9":     {EveryDays: 1, Times: []string{"09:00", "18:00"}},
		"отчёт 20.11 напоминать ежедневно в 10, 14": {EveryDays: 1, Times: []string{"10:00", "14:00"}},
	}
	for line, want := range cases {
		_, got := ExtractRemindRule(line)
		if got == nil || !reflect.DeepEqual(*got, want) {
			t.Errorf("%q → %+v, want %+v", line, got, want)
		}
	}
	// Without the opener «каждый день» stays a repeat, as before.
	p := ParseLine("каждый день пить таблетки", time.Now())
	if p.Draft.RemindRule != nil || p.Draft.Repeat == "" {
		t.Errorf("a plain repeat became a reminder rule: rule %+v, repeat %q", p.Draft.RemindRule, p.Draft.Repeat)
	}
}

// The task path reads the same clause, and keeps its due.
func TestTaskLineCarriesTheRule(t *testing.T) {
	now := time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)
	r := ParseTask("сдать отчёт 20.11 напоминать каждый день в 9", now)
	if r.Title != "сдать отчёт" || r.DueDate == nil || r.DueDate.Day() != 20 {
		t.Errorf("title %q due %v", r.Title, r.DueDate)
	}
	if r.RemindRule == nil || !reflect.DeepEqual(r.RemindRule.Times, []string{"09:00"}) {
		t.Errorf("rule = %+v", r.RemindRule)
	}
	if bad := ParseTask("экзамен 31.11", now); bad.DueDate != nil || bad.BadDate != "31.11" {
		t.Errorf("31.11 on a task: due %v, bad %q", bad.DueDate, bad.BadDate)
	}
}

// Review 07.11, both lines verbatim from the probe.
//   - «напомни каждый день …» with no date stays the daily REPEAT it was before
//     rules existed; a day rule with nothing to run up to would be dropped.
//   - a date right after the rule is a date: «12.12» is not 12:12.
func TestRuleNeverEatsARepeatOrADate(t *testing.T) {
	now := time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)
	for _, line := range []string{"напомни каждый день пить таблетки"} {
		task := ParseTask(line, now)
		if task.RemindRule != nil || task.Rrule != "FREQ=DAILY" {
			t.Errorf("task %q: rule %+v, repeat %q, want a daily repeat", line, task.RemindRule, task.Rrule)
		}
		ev := ParseLine(line, now)
		if ev.Draft.RemindRule != nil || ev.Draft.Repeat == "" {
			t.Errorf("event %q: rule %+v, repeat %q, want a repeat", line, ev.Draft.RemindRule, ev.Draft.Repeat)
		}
	}
	task := ParseTask("напоминать каждый день 12.12 экзамен", now)
	if task.DueDate == nil || task.DueDate.Day() != 12 || task.DueDate.Month() != 12 {
		t.Errorf("12.12 was not the due: %v (rule %+v)", task.DueDate, task.RemindRule)
	}
	if task.RemindRule == nil || len(task.RemindRule.Times) != 1 || task.RemindRule.Times[0] != DefaultRuleTime {
		t.Errorf("rule = %+v, want every day at the default time", task.RemindRule)
	}
}

// Without «в» a bare number is still a time, a dotted one is a date — checked
// on the clause itself, where the no-date fallback cannot hide it.
func TestBareTimesWithoutVButNeverADate(t *testing.T) {
	rest, r := ExtractRemindRule("экзамен 30.11 напоминать каждый день 12 и 16 до обеда")
	if r == nil || !reflect.DeepEqual(r.Times, []string{"12:00", "16:00"}) || rest != "экзамен 30.11 до обеда" {
		t.Errorf("bare times: rule %+v, rest %q", r, rest)
	}
	rest, r = ExtractRemindRule("напоминать каждый день 12.12 экзамен")
	if r == nil || !reflect.DeepEqual(r.Times, []string{DefaultRuleTime}) || rest != "12.12 экзамен" {
		t.Errorf("12.12 was taken as a time: rule %+v, rest %q", r, rest)
	}
}
