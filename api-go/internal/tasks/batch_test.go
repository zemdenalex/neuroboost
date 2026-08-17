package tasks

import "testing"

func batchStrPtr(s string) *string { return &s }
func batchIntPtr(i int) *int       { return &i }

func TestValidateBatchRow(t *testing.T) {
	tests := []struct {
		name     string
		req      CreateTaskRequest
		wantCode string
	}{
		{"valid minimal", CreateTaskRequest{Title: "купить хлеб"}, ""},
		{"empty title", CreateTaskRequest{Title: ""}, "MISSING_TITLE"},
		{"whitespace title", CreateTaskRequest{Title: "   "}, "MISSING_TITLE"},
		{"bad due date", CreateTaskRequest{Title: "x", DueDate: batchStrPtr("28-07-2026")}, "INVALID_DUE_DATE"},
		{"good due date", CreateTaskRequest{Title: "x", DueDate: batchStrPtr("2026-07-28T21:00:00Z")}, ""},
		{"empty due date is not a date", CreateTaskRequest{Title: "x", DueDate: batchStrPtr("")}, ""},
		{"priority out of range", CreateTaskRequest{Title: "x", Priority: batchIntPtr(9)}, "INVALID_PRIORITY"},
		{"priority in range", CreateTaskRequest{Title: "x", Priority: batchIntPtr(0)}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := validateBatchRow(tt.req)
			if code != tt.wantCode {
				t.Errorf("validateBatchRow() code = %q, want %q", code, tt.wantCode)
			}
			if code != "" && msg == "" {
				t.Error("validateBatchRow() returned a code with an empty message")
			}
		})
	}
}

// A batch is capped so an accidental paste of a large document cannot turn
// into thousands of inserts.
func TestBatchSizeLimit(t *testing.T) {
	if MaxBatchTasks != 100 {
		t.Errorf("MaxBatchTasks = %d, want 100", MaxBatchTasks)
	}
}

// Row errors carry their index so the client can point at the offending line.
func TestBatchRowErrorCarriesIndex(t *testing.T) {
	e := BatchRowError{Index: 7, Code: "MISSING_TITLE", Message: "Title is required"}
	if e.Index != 7 {
		t.Errorf("Index = %d, want 7", e.Index)
	}
}
