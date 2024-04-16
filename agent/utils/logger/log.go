package logger

import (
	"encoding/json"
	"log"
)

// Inspired from https://cloud.google.com/run/docs/logging

// Entry defines a log entry.
type Entry struct {
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
	Trace    string `json:"logging.googleapis.com/trace,omitempty"`

	// Logs Explorer allows filtering and display of this as `jsonPayload.component`.
	Component         string `json:"component,omitempty"`
	BenchmarkID       int64  `json:"benchmark_id,omitempty"`
	BenchmarkSequence int64  `json:"benchmark_sequence,omitempty"`
	// TaskID            string `json:"task_id,omitempty"`
}

// String renders an entry structure to the JSON format expected by Cloud Logging.
func (e Entry) String() string {
	if e.Severity == "" {
		e.Severity = "INFO"
	}
	out, err := json.Marshal(e)
	if err != nil {
		log.Printf("json.Marshal: %v", err)
	}
	return string(out)
}
