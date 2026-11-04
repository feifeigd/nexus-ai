package auth

import (
	"testing"
	"time"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("secret-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "secret-password" {
		t.Fatal("hash should not equal the plaintext")
	}
	if !CheckPassword("secret-password", hash) {
		t.Fatal("expected the correct password to match")
	}
	if CheckPassword("wrong-password", hash) {
		t.Fatal("expected a wrong password not to match")
	}
}

func TestTokenSignAndParse(t *testing.T) {
	now := time.Now()
	token, err := SignToken("test-secret", "user-1", "alice", "staff", now, time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	claims, err := ParseToken("test-secret", token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != "user-1" || claims.Username != "alice" || claims.Role != "staff" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenRejectsWrongSecret(t *testing.T) {
	token, err := SignToken("secret-a", "user-1", "alice", "staff", time.Now(), time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if _, err := ParseToken("secret-b", token); err == nil {
		t.Fatal("expected a token signed with a different secret to fail")
	}
}
