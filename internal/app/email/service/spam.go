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

func (s *Service) Unspam(ctx context.Context, in BatchInput) error {
    if err := in.validate(); err != nil {
        return err
    }
    if err := s.repo.UnspamEmailsForReceiver(ctx, in.UserID, in.EmailIDs); err != nil {
        return MapRepositoryError(err)
    }
    if err := s.repo.RemoveSpamSendersByReceiverEmails(ctx, in.UserID, in.EmailIDs); err != nil {
        return MapRepositoryError(err)
    }
    return nil
}

func (s *Service) Spam(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.MarkSendersAsSpamBatch(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) UnmarkSpamSenders(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.UnmarkSendersAsSpamBatch(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}
