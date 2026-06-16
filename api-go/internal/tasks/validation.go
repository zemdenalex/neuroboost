package tasks

// validPriority reports whether p is a usable task priority
// (0 = Buffer … 5 = If Possible).
func validPriority(p int) bool {
	return p >= 0 && p <= 5
}

// validStatus reports whether s is one of the known task statuses.
func validStatus(s TaskStatus) bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusScheduled, StatusDone, StatusCancelled:
		return true
	}
	return false
}

// validCategory reports whether c is one of the known task categories.
func validCategory(c TaskCategory) bool {
	switch c {
	case CategoryEmergency, CategoryAsap, CategoryMustToday, CategoryDeadlineSoon, CategoryIfPossible, CategoryBuffer:
		return true
	}
	return false
}

// validateTaskMutation checks the optional status/priority/category fields shared by
// create and update requests. It returns an (errorCode, message) pair to respond with,
// or ("", "") when every provided field is valid. Nil fields are not validated.
func validateTaskMutation(status *TaskStatus, priority *int, category *TaskCategory) (string, string) {
	if status != nil && !validStatus(*status) {
		return "INVALID_STATUS", "Invalid task status"
	}
	if priority != nil && !validPriority(*priority) {
		return "INVALID_PRIORITY", "Priority must be between 0 and 5"
	}
	if category != nil && !validCategory(*category) {
		return "INVALID_CATEGORY", "Invalid task category"
	}
	return "", ""
}
