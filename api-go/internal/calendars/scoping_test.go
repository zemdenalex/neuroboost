// api-go/internal/calendars/scoping_test.go
package calendars

import (
	"bytes"
	"os"
	"testing"
)

// 🔴 Одно забытое место = чужие данные в чужом браузере, и откатить это нельзя:
// показанное однажды показано. Поэтому запрет проверяется механически, а не
// вниманием ревьюера.
//
// Ищется буквально `user_id = $` — то есть ограничение ВЫБОРКИ по автору.
// INSERT, который пишет user_id как колонку авторства, под шаблон не попадает
// и остаётся разрешённым.
func TestHandlersDoNotScopeQueriesByUserID(t *testing.T) {
	forbidden := []byte("user_id = $")

	for _, path := range []string{
		"../events/handlers.go",
		"../tasks/handlers.go",
	} {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read %s: %v", path, err)
		}
		if bytes.Contains(src, forbidden) {
			t.Errorf(
				"%s scopes a query by %q. Access comes from calendar membership: "+
					"use calendars.CalendarIDsFor and `calendar_id = ANY($n)`.",
				path, forbidden)
		}
	}
}
