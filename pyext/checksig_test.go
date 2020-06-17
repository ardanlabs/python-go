package main

import (
	"testing"
)

func TestLogs(t *testing.T) {
	logsDir := "testdata/logs"
	if err := CheckSignatures(logsDir); err == nil {
		t.Fatalf("no error no %q", logsDir)
	}
}
