// Package machinekey 负责机器 SSH 私钥的认证加密，避免私钥明文落库。
package machinekey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	// Prefix 标识当前密文格式，便于将来轮换算法并区分历史明文数据。
	Prefix = "v1:"
	aad    = "nexusai/machine-ssh-private-key"
)

// Box 使用从 NEXUS_AUTH_SECRET 派生的独立密钥执行 AES-256-GCM。
// 派生时加入用途前缀，避免把 JWT 签名密钥直接复用于数据加密。
type Box struct {
	aead cipher.AEAD
}

func NewBox(secret string) (*Box, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("machine key encryption secret is required")
	}
	key := sha256.Sum256([]byte("nexusai/machine-key/v1\x00" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create machine key cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create machine key GCM: %w", err)
	}
	return &Box{aead: aead}, nil
}

// IsEncrypted 判断数据库值是否已经是受支持的密文；历史明文由 Store 兼容读取。
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, Prefix)
}

// Seal 加密私钥。每次使用随机 nonce，同一私钥重复写入也会得到不同密文。
func (b *Box) Seal(plaintext string) (string, error) {
	if b == nil || b.aead == nil {
		return "", errors.New("machine key cipher is not configured")
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate machine key nonce: %w", err)
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), []byte(aad))
	return Prefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Open 解密并校验私钥密文。篡改、密钥变化和格式错误都会返回错误。
func (b *Box) Open(encoded string) (string, error) {
	if b == nil || b.aead == nil {
		return "", errors.New("machine key cipher is not configured")
	}
	if !IsEncrypted(encoded) {
		return "", errors.New("machine key ciphertext version is missing")
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(encoded, Prefix))
	if err != nil {
		return "", fmt.Errorf("decode machine key ciphertext: %w", err)
	}
	nonceSize := b.aead.NonceSize()
	if len(payload) < nonceSize {
		return "", errors.New("machine key ciphertext is too short")
	}
	plaintext, err := b.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], []byte(aad))
	if err != nil {
		return "", errors.New("decrypt machine key: authentication failed")
	}
	return string(plaintext), nil
}
