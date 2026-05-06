package service

import "context"

func (s *Service) GetTrashEmails(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetTrashEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.GetTrashEmailsCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	unread, err := s.repo.GetUnreadTrashCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, total, unread)
}

func (s *Service) Untrash(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.SetTrashedBatch(ctx, in.UserID, in.EmailIDs, false); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}
