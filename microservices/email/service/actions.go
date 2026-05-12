package service

import "context"

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

func (s *Service) batchOperation(
	ctx context.Context,
	in BatchInput,
	op func(ctx context.Context, userID int64, emailIDs []int64) error,
) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := op(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Trash(ctx context.Context, in BatchInput) error {
	return s.batchOperation(ctx, in, s.repo.TrashEmails)
}

func (s *Service) Untrash(ctx context.Context, in BatchInput) error {
	return s.batchOperation(ctx, in, s.repo.UntrashEmails)
}

func (s *Service) Favorite(ctx context.Context, in BatchInput) error {
	return s.batchOperation(ctx, in, s.repo.StarEmails)
}

func (s *Service) Unfavorite(ctx context.Context, in BatchInput) error {
	return s.batchOperation(ctx, in, s.repo.UnstarEmails)
}

func (s *Service) Spam(ctx context.Context, in BatchInput) error {
	return s.batchOperation(ctx, in, s.repo.SpamEmails)
}

func (s *Service) Unspam(ctx context.Context, in BatchInput) error {
	return s.batchOperation(ctx, in, s.repo.UnspamEmails)
}

func (s *Service) Delete(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	deletedIDs, err := s.repo.GetDeletedEmailIDs(ctx, in.UserID, in.EmailIDs)
	if err != nil {
		return MapRepositoryError(err)
	}
	deletedSet := make(map[int64]struct{}, len(deletedIDs))
	for _, id := range deletedIDs {
		deletedSet[id] = struct{}{}
	}
	var toTrash, toRemove []int64
	for _, id := range in.EmailIDs {
		if _, ok := deletedSet[id]; ok {
			toRemove = append(toRemove, id)
		} else {
			toTrash = append(toTrash, id)
		}
	}
	if len(toTrash) > 0 {
		if err := s.repo.TrashEmails(ctx, in.UserID, toTrash); err != nil {
			return MapRepositoryError(err)
		}
	}
	if len(toRemove) > 0 {
		if err := s.repo.DeleteUserEmailsBatch(ctx, in.UserID, toRemove); err != nil {
			return MapRepositoryError(err)
		}
	}
	return nil
}
