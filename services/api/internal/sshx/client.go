package sshx

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config 配置 SSH 连接。
type Config struct {
	User    string        // 登录用户，默认 root
	Port    int           // SSH 端口，默认 22
	Timeout time.Duration // 连接 + 命令总超时，默认 30s
}

// Client 是 Runner 的 crypto/ssh 实现。
type Client struct {
	user    string
	port    int
	timeout time.Duration
}

// NewClient 构造 SSH 客户端。零值字段使用默认值。
func NewClient(config Config) *Client {
	if config.User == "" {
		config.User = "root"
	}
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	return &Client{
		user:    config.User,
		port:    config.Port,
		timeout: config.Timeout,
	}
}

// Exec 在 host 上以 privateKey 认证登录并执行 command。
func (c *Client) Exec(ctx context.Context, host string, privateKey []byte, command string) (Result, error) {
	if c == nil {
		return Result{}, fmt.Errorf("ssh client is required")
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return Result{}, fmt.Errorf("parse private key: %w", err)
	}
	clientConfig := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// 参考实现暂跳过主机密钥校验；生产环境应校验 known_hosts 以防中间人。
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.timeout,
	}
	addr := net.JoinHostPort(host, strconv.Itoa(c.port))
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return Result{}, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	runErr := session.Run(command)
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if runErr == nil {
		return result, nil
	}
	// 命令退出码非零：仍返回输出，由 ExitCode 表达失败，不当作连接错误。
	if exitErr, ok := runErr.(*ssh.ExitError); ok {
		result.ExitCode = exitErr.ExitStatus()
		return result, nil
	}
	// 其余是会话执行失败（例如信号终止），同样返回已捕获的输出。
	return result, fmt.Errorf("run command: %w", runErr)
}
