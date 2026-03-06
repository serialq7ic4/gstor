package utils

import (
	"strings"
	"testing"
)

func TestExecShellTimeout(t *testing.T) {
	t.Setenv("GSTOR_SHELL_TIMEOUT", "50ms")

	_, err := ExecShell("sleep 1")
	if err == nil {
		t.Fatal("ExecShell should time out")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
