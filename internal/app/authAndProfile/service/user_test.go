package service

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
)

type mockUserRepo struct {
	findByEmail func(ctx context.Context, email string) (*models.User, error)
	save        func(ctx context.Context, user models.User) error
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.findByEmail(ctx, email)
}

func (m *mockUserRepo) Save(ctx context.Context, user models.User) error {
	return m.save(ctx, user)
}

type mockJWT struct {
	generate func(email string) (string, error)
}

func (m *mockJWT) GenerateJWT(email string) (string, error) {
	return m.generate(email)
}

func (m *mockJWT) ValidateJWT(token string) (*utils.JwtPayload, error) {
	return nil, nil
}

func TestAuthService_SignUp(t *testing.T) {

	tests := []struct {
		name        string
		repoFind    func(ctx context.Context, email string) (*models.User, error)
		repoSave    func(ctx context.Context, user models.User) error
		jwtGenerate func(email string) (string, error)

		expectErr bool
	}{
		{
			name: "success",
			repoFind: func(ctx context.Context, email string) (*models.User, error) {
				return nil, repository.ErrUserNotFound
			},
			repoSave: func(ctx context.Context, user models.User) error {
				return nil
			},
			jwtGenerate: func(email string) (string, error) {
				return "token123", nil
			},
		},
		{
			name: "user already exists",
			repoFind: func(ctx context.Context, email string) (*models.User, error) {
				return &models.User{}, nil
			},
			expectErr: true,
		},
		{
			name: "save error",
			repoFind: func(ctx context.Context, email string) (*models.User, error) {
				return nil, repository.ErrUserNotFound
			},
			repoSave: func(ctx context.Context, user models.User) error {
				return errors.New("db error")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			repo := &mockUserRepo{
				findByEmail: tt.repoFind,
				save:        tt.repoSave,
			}

			jwt := &mockJWT{
				generate: tt.jwtGenerate,
			}

			service := NewAuthService(repo, jwt)

			_, err := service.SignUp(context.Background(), SignUpInput{
				Email:    "test@test.com",
				Password: "123456",
				Name:     "Ivan",
				Surname:  "Ivanov",
			})

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got nil")
			}

			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_SignIn(t *testing.T) {

	hash, _ := utils.Hash("123456")

	tests := []struct {
		name        string
		repoFind    func(ctx context.Context, email string) (*models.User, error)
		jwtGenerate func(email string) (string, error)

		password  string
		expectErr bool
	}{
		{
			name: "success",
			repoFind: func(ctx context.Context, email string) (*models.User, error) {
				return &models.User{
					Email:    "test@test.com",
					Password: hash,
				}, nil
			},
			jwtGenerate: func(email string) (string, error) {
				return "token123", nil
			},
			password: "123456",
		},
		{
			name: "user not found",
			repoFind: func(ctx context.Context, email string) (*models.User, error) {
				return nil, repository.ErrUserNotFound
			},
			expectErr: true,
		},
		{
			name: "wrong password",
			repoFind: func(ctx context.Context, email string) (*models.User, error) {
				return &models.User{
					Email:    "test@test.com",
					Password: hash,
				}, nil
			},
			password:  "wrong",
			expectErr: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			repo := &mockUserRepo{
				findByEmail: tt.repoFind,
			}

			jwt := &mockJWT{
				generate: tt.jwtGenerate,
			}

			service := NewAuthService(repo, jwt)

			_, err := service.SignIn(context.Background(), SignInInput{
				Email:    "test@test.com",
				Password: tt.password,
			})

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got nil")
			}

			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
