package tools

import (
	"strings"
	"testing"
	"time"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	secret := "supersecret"
	expire := 2 * time.Hour

	jwtMgr := NewJWTManager(secret, expire)

	email := "test@example.com"
	token, err := jwtMgr.GenerateJWT(email)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateJWT returned empty token")
	}

	payload, err := jwtMgr.ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT returned error: %v", err)
	}

	if payload.Email != email {
		t.Fatalf("expected email %s, got %s", email, payload.Email)
	}
}

func TestJWTManager_Validate_InvalidToken(t *testing.T) {
	jwtMgr := NewJWTManager("secret", 2*time.Hour)

	// поддельный токен
	_, err := jwtMgr.ValidateJWT("invalid.token.parts")
	if err == nil {
		t.Fatal("expected error for invalid token format")
	}
}

func TestJWTManager_Validate_TamperedToken(t *testing.T) {
	jwtMgr := NewJWTManager("secret", 2*time.Hour)

	token, _ := jwtMgr.GenerateJWT("user@test.com")
	// изменяем часть токена
	parts := strings.Split(token, ".")
	parts[1] = "tampered"
	tamperedToken := strings.Join(parts, ".")

	_, err := jwtMgr.ValidateJWT(tamperedToken)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestJWTManager_Validate_ExpiredToken(t *testing.T) {
	jwtMgr := NewJWTManager("secret", -1*time.Hour)

	token, _ := jwtMgr.GenerateJWT("user@test.com")

	_, err := jwtMgr.ValidateJWT(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if err.Error() != "token expired" {
		t.Fatalf("expected 'token expired', got %v", err)
	}
}
