package sshx

import "strings"

// Whitelist 校验命令是否允许执行。它是「逐步开放命令」的权威：只有命中的命令才能
// 进入 SSH 执行，且先拒绝 shell 元字符，防止用拼接/替换绕过白名单。
type Whitelist struct {
	prefixes []string
}

// NewWhitelist 用允许的命令前缀构造白名单。空前缀会被忽略。
func NewWhitelist(prefixes []string) *Whitelist {
	trimmed := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix = strings.TrimSpace(prefix); prefix != "" {
			trimmed = append(trimmed, prefix)
		}
	}
	return &Whitelist{prefixes: trimmed}
}

// Prefixes 返回允许的命令前缀（用于前端展示提示）。
func (w *Whitelist) Prefixes() []string {
	if w == nil {
		return nil
	}
	return append([]string(nil), w.prefixes...)
}

// Allowed 判断命令是否允许执行。规则：
//  1. 拒绝空命令；
//  2. 拒绝包含 shell 元字符的命令（; | & > < ` $ \ 换行等），防命令拼接/替换绕过；
//  3. 命令必须等于某个前缀，或以「前缀 + 空格」开头（如 "systemctl status nginx"）。
func (w *Whitelist) Allowed(command string) bool {
	if w == nil {
		return false
	}
	command = strings.TrimSpace(command)
	if command == "" || containsShellMetachar(command) {
		return false
	}
	for _, prefix := range w.prefixes {
		if command == prefix || strings.HasPrefix(command, prefix+" ") {
			return true
		}
	}
	return false
}

// containsShellMetachar 报告命令是否包含可用于拼接或替换命令的 shell 元字符。
func containsShellMetachar(command string) bool {
	return strings.ContainsAny(command, ";|&><`$\\\n\r")
}
