package utils

import (
	"fmt"
	"os"
)

var debugEnabled bool

// SetDebugMode 设置 debug 模式
func SetDebugMode(enabled bool) {
	debugEnabled = enabled
}

// IsDebugEnabled 返回是否启用了 debug 模式
func IsDebugEnabled() bool {
	return debugEnabled
}

// DebugLog 输出 debug 日志
// 只有在 debug 模式启用时才会输出
func DebugLog(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", fmt.Sprintf(format, args...))
	}
}

// DebugLogCommand 输出 shell 命令的 debug 日志
func DebugLogCommand(cmd string, shell string) {
	if debugEnabled {
		if shell != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Executing command (%s): %s\n", shell, cmd)
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] Executing command: %s\n", cmd)
		}
	}
}

// DebugLogStep 输出关键步骤的 debug 日志
// 支持格式化字符串，用法类似 fmt.Printf
// 使用 fmt.Sprintf 确保格式字符串安全
func DebugLogStep(format string, args ...interface{}) {
	if debugEnabled {
		// 使用 fmt.Sprintf 先格式化，避免 go vet 警告
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "[DEBUG] Step: %s\n", msg)
	}
}
