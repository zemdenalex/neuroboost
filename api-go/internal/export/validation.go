package export

// Per-row import validation. Import must enforce the same rules the Create/Update
// handlers do, otherwise a corrupted or hand-edited export silently persists rows
// the rest of the app would reject (and which later fail to edit). These mirror:
//   - internal/events/validation.go  (event time-range: end strictly after start)
//   - internal/tasks/types.go + validation.go (task status/category/priority)
// Returning a non-empty reason marks the row invalid; "" means valid.

var validImportStatuses = map[string]bool{
	"TODO": true, "IN_PROGRESS": true, "SCHEDULED": true, "DONE": true, "CANCELLED": true,
}

var validImportCategories = map[string]bool{
	"EMERGENCY": true, "ASAP": true, "MUST_TODAY": true, "DEADLINE_SOON": true, "IF_POSSIBLE": true, "BUFFER": true,
}

// validateImportEvent returns a reason the event row is invalid, or "" if it is valid.
func validateImportEvent(ev EventRow) string {
	if !ev.EndsAt.After(ev.StartsAt) {
		return "end time must be after start time"
	}
	return ""
}

// validateImportTask returns a reason the task row is invalid, or "" if it is valid.
// Category is optional (nil = unset); status and priority are always required.
func validateImportTask(tk TaskRow) string {
	if !validImportStatuses[tk.Status] {
		return "invalid task status"
	}
	if tk.Priority < 0 || tk.Priority > 5 {
		return "priority must be between 0 and 5"
	}
	if tk.Category != nil && !validImportCategories[*tk.Category] {
		return "invalid task category"
	}
	return ""
}
