package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecShell 执行 shell 命令并返回输出和错误
// 这是推荐的函数，因为它允许调用者处理错误
func ExecShell(cmd string) (string, error) {
	return ExecShellWithShell(cmd, "/bin/sh")
}

// ExecShellWithShell 使用指定的 shell 执行命令
// shell 可以是 "/bin/sh" 或 "/bin/bash"
// 如果启用了 debug 模式，会打印执行的命令
func ExecShellWithShell(cmd string, shell string) (string, error) {
	// Debug 模式：打印执行的命令
	DebugLogCommand(cmd, shell)

	cmdjob := exec.Command(shell, "-c", cmd)
	var stdout, stderr bytes.Buffer
	cmdjob.Stdout = &stdout
	cmdjob.Stderr = &stderr

	err := cmdjob.Run()
	outStr := stdout.String()
	errStr := stderr.String()

	if err != nil {
		// Debug 模式：打印错误信息
		if debugEnabled {
			fmt.Fprintf(os.Stderr, "[DEBUG] Command failed: %v, stderr: %s\n", err, errStr)
		}
		return "", fmt.Errorf("command '%s' failed: %w, stderr: %s", cmd, err, errStr)
	}

	// Debug 模式：打印输出长度（不打印完整输出，可能很长）
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] Command succeeded, output length: %d bytes\n", len(outStr))
	}

	// 移除末尾的换行符，保持与原有行为一致
	return strings.TrimRight(outStr, "\n"), nil
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
