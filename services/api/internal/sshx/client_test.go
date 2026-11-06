package sshx

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

// startTestSSHServer 启动一个内存内 SSH 服务器：接受 clientSigner 的公钥，
// 对每条 exec 命令按 handler 返回输出与退出码。返回监听端口与客户端私钥 PEM。
func startTestSSHServer(t *testing.T, clientSigner ssh.Signer, handler func(command string) (stdout, stderr string, exitCode int)) int {
	t.Helper()
	hostSigner, _ := generateKeyPair(t)
	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if string(key.Marshal()) != string(clientSigner.PublicKey().Marshal()) {
				return nil, fmt.Errorf("unknown public key")
			}
			return &ssh.Permissions{}, nil
		},
	}
	serverConfig.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveTestConn(conn, serverConfig, handler)
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port
}

func serveTestConn(netConn net.Conn, config *ssh.ServerConfig, handler func(string) (string, string, int)) {
	defer netConn.Close()
	serverConn, chans, reqs, err := ssh.NewServerConn(netConn, config)
	if err != nil {
		return
	}
	defer serverConn.Close()
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go serveTestSession(channel, requests, handler)
	}
}

func serveTestSession(channel ssh.Channel, requests <-chan *ssh.Request, handler func(string) (string, string, int)) {
	defer channel.Close()
	for req := range requests {
		if req.Type != "exec" {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)
		var execMsg struct{ Command string }
		if err := ssh.Unmarshal(req.Payload, &execMsg); err != nil {
			return
		}
		stdout, stderr, exitCode := handler(execMsg.Command)
		_, _ = channel.Write([]byte(stdout))
		_, _ = channel.Stderr().Write([]byte(stderr))
		_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{Status: uint32(exitCode)}))
		return
	}
}

func generateKeyPair(t *testing.T) (ssh.Signer, []byte) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	return signer, pem.EncodeToMemory(block)
}

func TestClientExec(t *testing.T) {
	clientSigner, clientKeyPEM := generateKeyPair(t)
	port := startTestSSHServer(t, clientSigner, func(command string) (string, string, int) {
		switch command {
		case "hostname":
			return "test-host\n", "", 0
		case "boom":
			return "", "boom: no such thing\n", 1
		default:
			return "", "unknown command\n", 127
		}
	})
	client := NewClient(Config{User: "root", Port: port})

	t.Run("exit zero", func(t *testing.T) {
		result, err := client.Exec(context.Background(), "127.0.0.1", clientKeyPEM, "hostname")
		if err != nil {
			t.Fatalf("exec: %v", err)
		}
		if result.Stdout != "test-host\n" || result.Stderr != "" || result.ExitCode != 0 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("non-zero exit", func(t *testing.T) {
		result, err := client.Exec(context.Background(), "127.0.0.1", clientKeyPEM, "boom")
		if err != nil {
			t.Fatalf("exec: %v", err)
		}
		if result.Stderr != "boom: no such thing\n" || result.ExitCode != 1 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
}

func TestClientExecErrors(t *testing.T) {
	clientSigner, clientKeyPEM := generateKeyPair(t)
	port := startTestSSHServer(t, clientSigner, func(string) (string, string, int) { return "", "", 0 })
	client := NewClient(Config{User: "root", Port: port})

	t.Run("bad private key", func(t *testing.T) {
		if _, err := client.Exec(context.Background(), "127.0.0.1", []byte("not-a-key"), "hostname"); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		unused, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		closedPort := unused.Addr().(*net.TCPAddr).Port
		_ = unused.Close()
		closedClient := NewClient(Config{User: "root", Port: closedPort})
		if _, err := closedClient.Exec(context.Background(), "127.0.0.1", clientKeyPEM, "hostname"); err == nil {
			t.Fatal("expected connection error")
		}
	})
}
