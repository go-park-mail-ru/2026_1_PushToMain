package service

import "context"

type Repository interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	return s.repo.IsAdmin(ctx, userID)
}
