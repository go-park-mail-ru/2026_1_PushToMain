package service

import "context"

func (s *Service) GetFavoriteEmails(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetFavoriteEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, len(emails), 0)
}
