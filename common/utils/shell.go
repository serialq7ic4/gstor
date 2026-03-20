package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultShellTimeout = 30 * time.Second

type ShellResult struct {
	Output   string
	Stderr   string
	ExitCode int
}

// ExecShell 执行 shell 命令并返回输出和错误
// 这是推荐的函数，因为它允许调用者处理错误
func ExecShell(cmd string) (string, error) {
	return ExecShellWithShell(cmd, "/bin/sh")
}

func getShellTimeout() time.Duration {
	value := strings.TrimSpace(os.Getenv("GSTOR_SHELL_TIMEOUT"))
	if value == "" {
		return defaultShellTimeout
	}

	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return defaultShellTimeout
	}
	return timeout
}

// ExecShellWithShell 使用指定的 shell 执行命令
// shell 可以是 "/bin/sh" 或 "/bin/bash"
// 如果启用了 debug 模式，会打印执行的命令
func ExecShellWithShell(cmd string, shell string) (string, error) {
	result, err := ExecShellResultWithShell(cmd, shell)
	return result.Output, err
}

func ExecShellResult(cmd string) (ShellResult, error) {
	return ExecShellResultWithShell(cmd, "/bin/sh")
}

func ExecShellResultWithShell(cmd string, shell string) (ShellResult, error) {
	// Debug 模式：打印执行的命令
	DebugLogCommand(cmd, shell)

	timeout := getShellTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmdjob := exec.CommandContext(ctx, shell, "-c", cmd)
	var stdout, stderr bytes.Buffer
	cmdjob.Stdout = &stdout
	cmdjob.Stderr = &stderr

	err := cmdjob.Run()
	outStr := stdout.String()
	errStr := stderr.String()
	result := ShellResult{
		Output: strings.TrimRight(outStr, "\n"),
		Stderr: strings.TrimRight(errStr, "\n"),
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ShellResult{}, fmt.Errorf("command '%s' timed out after %s", cmd, timeout)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		// Debug 模式：打印错误信息
		if debugEnabled {
			fmt.Fprintf(os.Stderr, "[DEBUG] Command failed: %v, stderr: %s\n", err, errStr)
		}
		return result, fmt.Errorf("command '%s' failed: %w, stderr: %s", cmd, err, errStr)
	}
	result.ExitCode = 0

	// Debug 模式：打印输出长度（不打印完整输出，可能很长）
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] Command succeeded, output length: %d bytes\n", len(outStr))
	}

	return result, nil
}

// ExecShellSafe 执行 shell 命令，错误时返回空字符串
// 这个函数是为了向后兼容，新代码应该使用 ExecShell
func ExecShellSafe(cmd string) string {
	result, _ := ExecShell(cmd)
	return result
}

// ExecShellSafeWithShell 使用指定的 shell 执行命令，错误时返回空字符串
func ExecShellSafeWithShell(cmd string, shell string) string {
	result, _ := ExecShellWithShell(cmd, shell)
	return result
}
