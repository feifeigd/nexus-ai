// Package sshx 提供在远程机器上执行 SSH 命令的能力。
//
// 安全边界：浏览器不直接连接目标机器，SSH 全部由 go-zero 后端完成；
// 命令执行走 Whitelist 白名单校验，且仅 admin 可触发（由上层中间件强制）。
package sshx

import "context"

// Result 是一次 SSH 命令执行的输出。命令退出码非零时 ExitCode 非零；
// 连接失败时 Result 无意义（此时 Exec 返回的 error 非 nil）。
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner 在远程机器上执行一条命令。
type Runner interface {
	// Exec 在 host 上以 privateKey 认证登录并执行 command。
	// err 非 nil 表示未能执行（私钥解析 / 连接 / 会话失败）；
	// 命令本身退出码非零时 err 仍为 nil，由 Result.ExitCode 表达。
	Exec(ctx context.Context, host string, privateKey []byte, command string) (Result, error)
}
