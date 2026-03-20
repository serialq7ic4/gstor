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

func TestExecShellResultCapturesStdoutAndExitCode(t *testing.T) {
	result, err := ExecShellResult("printf 'smart output'; exit 4")
	if err == nil {
		t.Fatal("ExecShellResult should return an error for non-zero exit status")
	}
	if result.Output != "smart output" {
		t.Fatalf("stdout = %q, want %q", result.Output, "smart output")
	}
	if result.ExitCode != 4 {
		t.Fatalf("exit code = %d, want %d", result.ExitCode, 4)
	}
}
