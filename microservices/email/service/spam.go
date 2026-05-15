package service

import "context"

func (s *Service) GetSpamEmails(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetSpamEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.GetSpamEmailsCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	unread, err := s.repo.GetUnreadSpamCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, total, unread)
}
