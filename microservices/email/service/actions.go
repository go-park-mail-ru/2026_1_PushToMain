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

type SwitchIsInboxInput struct {
	UserID  int64
	EmailID int64
}

func (s *Service) SwitchIsInbox(ctx context.Context, input SwitchIsInboxInput) error {
	if err := s.repo.SwitchIsInbox(ctx, input.EmailID, input.UserID); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Trash(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}

	spamIDs, err := s.repo.GetSpamEmailIDs(ctx, in.UserID, in.EmailIDs)
	if err != nil {
		return MapRepositoryError(err)
	}
	spamSet := make(map[int64]struct{}, len(spamIDs))
	for _, id := range spamIDs {
		spamSet[id] = struct{}{}
	}

	var nonSpam []int64
	for _, id := range in.EmailIDs {
		if _, ok := spamSet[id]; !ok {
			nonSpam = append(nonSpam, id)
		}
	}

	deletedIDs, err := s.repo.GetDeletedEmailIDs(ctx, in.UserID, nonSpam)
	if err != nil {
		return MapRepositoryError(err)
	}
	deletedSet := make(map[int64]struct{}, len(deletedIDs))
	for _, id := range deletedIDs {
		deletedSet[id] = struct{}{}
	}

	var toTrash, toRemove []int64
	toRemove = append(toRemove, spamIDs...)
	for _, id := range nonSpam {
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

func (s *Service) Untrash(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.UntrashEmails(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Favorite(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.StarEmails(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Unfavorite(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.UnstarEmails(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Spam(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.SpamEmails(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Unspam(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.UnspamEmails(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) BlockSenders(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.BlockSendersBatch(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}

	spamIDs, err := s.repo.GetSpamEmailIDs(ctx, in.UserID, in.EmailIDs)
	if err != nil {
		return MapRepositoryError(err)
	}
	spamSet := make(map[int64]struct{}, len(spamIDs))
	for _, id := range spamIDs {
		spamSet[id] = struct{}{}
	}

	var nonSpam []int64
	for _, id := range in.EmailIDs {
		if _, ok := spamSet[id]; !ok {
			nonSpam = append(nonSpam, id)
		}
	}

	deletedIDs, err := s.repo.GetDeletedEmailIDs(ctx, in.UserID, nonSpam)
	if err != nil {
		return MapRepositoryError(err)
	}
	deletedSet := make(map[int64]struct{}, len(deletedIDs))
	for _, id := range deletedIDs {
		deletedSet[id] = struct{}{}
	}

	var toTrash, toRemove []int64
	toRemove = append(toRemove, spamIDs...)
	for _, id := range nonSpam {
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
