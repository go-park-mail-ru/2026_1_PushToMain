package service

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
)

type BatchInput struct {
	UserID   int64
	EmailIDs []int64
}

func (in BatchInput) validate() error {
	if len(in.EmailIDs) == 0 {
		return ErrEmptyIDs
	}
	return nil
}

func (s *Service) Favorite(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.SetStarredBatch(ctx, in.UserID, in.EmailIDs, true); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Unfavorite(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.SetStarredBatch(ctx, in.UserID, in.EmailIDs, false); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Trash(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}

	var toSoft, toHard []int64
	for _, id := range in.EmailIDs {
		flags, err := s.findUserEmailFlags(ctx, id, in.UserID)
		if err != nil {
			return MapRepositoryError(err)
		}
		if flags == nil {
			// Письма у юзера нет ни как у получателя, ни как у отправителя — пропускаем.
			continue
		}
		if flags.IsDeleted {
			toHard = append(toHard, id)
		} else {
			toSoft = append(toSoft, id)
		}
	}

	if len(toSoft) > 0 {
		if err := s.repo.SetTrashedBatch(ctx, in.UserID, toSoft, true); err != nil {
			return MapRepositoryError(err)
		}
	}
	if len(toHard) > 0 {
		if err := s.repo.HardDeleteBatch(ctx, in.UserID, toHard); err != nil {
			return MapRepositoryError(err)
		}
	}
	return nil
}

func (s *Service) findUserEmailFlags(ctx context.Context, emailID, userID int64) (*flagsView, error) {
	receiver, receiverErr := s.repo.GetUserEmailFlags(ctx, emailID, userID, false)
	if receiverErr != nil && !errors.Is(receiverErr, repository.ErrMailNotFound) {
		return nil, receiverErr
	}

	sender, senderErr := s.repo.GetUserEmailFlags(ctx, emailID, userID, true)
	if senderErr != nil && !errors.Is(senderErr, repository.ErrMailNotFound) {
		return nil, senderErr
	}

	if receiver == nil && sender == nil {
		return nil, nil
	}

	isDeleted := (receiver != nil && receiver.IsDeleted) || (sender != nil && sender.IsDeleted)
	return &flagsView{IsDeleted: isDeleted}, nil
}

type flagsView struct {
	IsDeleted bool
}
