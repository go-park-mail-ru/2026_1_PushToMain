package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/models"
)

func TestUserRepo_SaveAndFind(t *testing.T) {
	repo := NewMemoryUserRepo()
	ctx := context.Background()

	user := models.User{
		Email:   "test@mail.com",
		Name:    "Test",
		Surname: "User",
	}

	err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	foundUser, err := repo.FindByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("FindByEmail returned error: %v", err)
	}

	if foundUser.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, foundUser.Email)
	}

	if foundUser.Name != user.Name {
		t.Fatalf("expected name %s, got %s", user.Name, foundUser.Name)
	}
}

func TestUserRepo_FindUserNotFound(t *testing.T) {
	repo := NewMemoryUserRepo()
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "notfound@mail.com")

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepo_SaveOverwrite(t *testing.T) {
	repo := NewMemoryUserRepo()
	ctx := context.Background()

	user := models.User{
		Email:   "test@mail.com",
		Name:    "First",
		Surname: "User",
	}

	err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	updatedUser := models.User{
		Email:   "test@mail.com",
		Name:    "Updated",
		Surname: "User",
	}

	err = repo.Save(ctx, updatedUser)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	foundUser, err := repo.FindByEmail(ctx, "test@mail.com")
	if err != nil {
		t.Fatalf("FindByEmail returned error: %v", err)
	}

	if foundUser.Name != "Updated" {
		t.Fatalf("expected updated name, got %s", foundUser.Name)
	}
}
