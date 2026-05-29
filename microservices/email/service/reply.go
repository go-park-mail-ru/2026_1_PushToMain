package service

import (
	"context"
	"mime/multipart"
)

type ReplyInput struct {
	UserID        int64
	ParentEmailID int64
	Header        string
	Body          string
	IsAnonymous   bool

	Files       []multipart.File
	FileHeaders []*multipart.FileHeader
}

func (s *Service) Reply(ctx context.Context, in ReplyInput) (*SendEmailResult, error) {
	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.ParentEmailID); err != nil {
		return nil, MapRepositoryError(err)
	}

	parent, err := s.repo.GetEmailByID(ctx, in.ParentEmailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if parent == nil {
		return nil, ErrEmailNotFound
	}
	if !parent.IsAnonymous {
		return nil, ErrReplyTargetNotAnonymous
	}
	if parent.SenderID == nil {
		return nil, ErrReplyTargetNotAnonymous
	}
	if *parent.SenderID == in.UserID {
		return nil, ErrReplyByOriginalSender
	}

	author, err := s.userClient.GetUserByID(ctx, *parent.SenderID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	send := SendEmailInput{
		UserId:        in.UserID,
		Header:        in.Header,
		Body:          in.Body,
		Receivers:     []string{author.Email},
		IsAnonymous:   in.IsAnonymous,
		ParentEmailID: &in.ParentEmailID,
		Files:         in.Files,
		FileHeaders:   in.FileHeaders,
	}
	return s.SendEmail(ctx, send)
}
