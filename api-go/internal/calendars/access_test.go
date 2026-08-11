package calendars

import "testing"

func TestAccessibleIDsKeepsOnlyActiveMemberships(t *testing.T) {
	// 🔴 Приглашённый не видит ничего до принятия. Если это правило ошибётся,
	// человек увидит чужой календарь, которого ещё не принимал.
	got := AccessibleIDs([]Membership{
		{CalendarID: "a", Status: StatusActive},
		{CalendarID: "b", Status: StatusInvited},
		{CalendarID: "c", Status: StatusActive},
	})
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("got %v, want [a c]", got)
	}
}

func TestAccessibleIDsOnEmptyInputIsEmptyNotNil(t *testing.T) {
	// Пустой срез уходит в `calendar_id = ANY($1)`. nil там означал бы
	// «нет условия», то есть выдачу чужих строк.
	got := AccessibleIDs(nil)
	if got == nil {
		t.Fatal("AccessibleIDs(nil) returned nil; must be an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestAccessibleIDsIgnoresUnknownStatus(t *testing.T) {
	// Неизвестный статус трактуется как «нет доступа», а не как «есть».
	got := AccessibleIDs([]Membership{{CalendarID: "a", Status: "revoked"}})
	if len(got) != 0 {
		t.Fatalf("got %v, want empty for an unrecognised status", got)
	}
}
