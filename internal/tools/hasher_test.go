package tools

import "testing"

func TestHashAndComparePassword(t *testing.T) {
	password := "my-secret-password"

	hash, err := Hash(password)
	if err != nil {
		t.Fatalf("hash returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	err = ComparePasswordAndHash(hash, password)
	if err != nil {
		t.Fatalf("password should match hash, got error: %v", err)
	}
}

func TestComparePassword_WrongPassword(t *testing.T) {
	password := "correct-password"
	wrongPassword := "wrong-password"

	hash, err := Hash(password)
	if err != nil {
		t.Fatalf("hash returned error: %v", err)
	}

	err = ComparePasswordAndHash(hash, wrongPassword)
	if err == nil {
		t.Fatal("expected error for wrong password but got nil")
	}
}
