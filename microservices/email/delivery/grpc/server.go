package grpc

import (
	"context"

	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"

	emailService "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	emailpb.UnimplementedEmailServiceServer
	service *emailService.Service
}

func New(service *emailService.Service) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) SendSystemEmail(
	ctx context.Context,
	req *emailpb.SendSystemEmailRequest,
) (*emailpb.SendSystemEmailResponse, error) {
	if err := s.service.SendSystemEmail(
		ctx,
		req.RecipientUserId,
		req.RecipientEmail,
		req.SystemEmail,
		req.Header,
		req.Body,
	); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emailpb.SendSystemEmailResponse{Success: true}, nil
}

func (s *Server) SwitchIsInbox(ctx context.Context, req *emailpb.SwitchIsInboxRequest) (*emailpb.SwitchIsInboxResponse, error) {
	err := s.service.SwitchIsInbox(ctx,
		emailService.SwitchIsInboxInput{
			EmailID: req.EmailId,
			UserID:  req.UserId},
	)
	return &emailpb.SwitchIsInboxResponse{
		Success: err == nil,
	}, nil
}

func (s *Server) GetEmailById(
	ctx context.Context,
	req *emailpb.GetEmailByIdRequest,
) (*emailpb.GetEmailByIdResponse, error) {

	email, err := s.service.GetEmailByID(
		ctx,
		emailService.GetEmailInput{
			EmailID: req.EmailId,
			UserID:  req.UserId},
	)

	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			err.Error(),
		)
	}

	return &emailpb.GetEmailByIdResponse{
		Email: &emailpb.Email{
			Id:          email.ID,
			SenderId:    email.SenderID,
			Header:      email.Header,
			Body:        email.Body,
			CreatedAt:   email.CreatedAt.String(),
			IsAnonymous: email.IsAnonymous,
		},
	}, nil
}

func (s *Server) GetUserEmailID(
	ctx context.Context,
	req *emailpb.GetUserEmailIDRequest,
) (*emailpb.GetUserEmailIDResponse, error) {

	userEmailId, err := s.service.GetUserEmailID(
		ctx,
		req.EmailId,
		req.UserId,
	)

	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			err.Error(),
		)
	}

	return &emailpb.GetUserEmailIDResponse{
		UserEmailId: userEmailId,
	}, nil
}

func (s *Server) GetEmailIdsByUserEmailIds(
	ctx context.Context,
	req *emailpb.GetEmailIdsByUserEmailIdsRequest,
) (*emailpb.GetEmailIdsByUserEmailIdsResponse, error) {

	emailIds, err := s.service.GetEmailIdsByUserEmailIds(
		ctx,
		req.UserEmailIds,
	)

	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			err.Error(),
		)
	}

	return &emailpb.GetEmailIdsByUserEmailIdsResponse{
		EmailIds: emailIds,
	}, nil
}

func (s *Server) CheckEmailAccess(
	ctx context.Context,
	req *emailpb.CheckEmailAccessRequest,
) (*emailpb.CheckEmailAccessResponse, error) {

	err := s.service.CheckEmailAccess(
		ctx,
		emailService.GetEmailInput{
			EmailID: req.EmailId,
			UserID:  req.UserId},
	)

	return &emailpb.CheckEmailAccessResponse{
		HasAccess: err == nil,
	}, nil
}

func (s *Server) GetEmailsByIds(
	ctx context.Context,
	req *emailpb.GetEmailsByIdsRequest,
) (*emailpb.GetEmailsByIdsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if len(req.EmailIds) == 0 {
		return &emailpb.GetEmailsByIdsResponse{
			Emails:      []*emailpb.FolderEmail{},
			UnreadCount: 0,
		}, nil
	}

	result, err := s.service.GetEmailsByIDs(
		ctx,
		req.EmailIds,
		req.UserId,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &emailpb.GetEmailsByIdsResponse{
		UnreadCount: int32(result.UnreadCount),
		Emails:      make([]*emailpb.FolderEmail, 0, len(result.Emails)),
	}

	for _, em := range result.Emails {
		resp.Emails = append(resp.Emails, &emailpb.FolderEmail{
			Id:            em.ID,
			SenderEmail:   em.SenderEmail,
			SenderName:    em.SenderName,
			SenderSurname: em.SenderSurname,
			ReceiverList:  em.ReceiverList,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     timestamppb.New(em.CreatedAt),
			IsRead:        em.IsRead,
			IsAnonymous:   em.IsAnonymous,
		})
	}

	return resp, nil
}

// GetEmailForSupport — внутренний эндпойнт для support-сервиса. ВНИМАНИЕ:
// доступ к нему должен быть закрыт инфраструктурно (network-policy / mTLS /
// authz-interceptor); делегация в сервис без дополнительных проверок здесь —
// сознательное решение, чтобы единый источник истины для «выдать как есть»
// был один.
func (s *Server) GetEmailForSupport(
	ctx context.Context,
	req *emailpb.GetEmailForSupportRequest,
) (*emailpb.GetEmailForSupportResponse, error) {
	if req == nil || req.EmailId == 0 {
		return nil, status.Error(codes.InvalidArgument, "email_id is required")
	}

	res, err := s.service.GetEmailForSupport(ctx, req.EmailId)
	if err != nil {
		// MapRepositoryError уже превратил ErrMailNotFound в ErrEmailNotFound;
		// маппим к gRPC-кодам по типу ошибки.
		if err == emailService.ErrEmailNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbAtt := make([]*emailpb.SupportAttachment, 0, len(res.Attachments))
	for _, a := range res.Attachments {
		pbAtt = append(pbAtt, &emailpb.SupportAttachment{
			Id:          a.ID,
			Filename:    a.FileName,
			ContentType: a.ContentType,
			SizeBytes:   a.SizeBytes,
			CreatedAt:   timestamppb.New(a.CreatedAt),
		})
	}

	return &emailpb.GetEmailForSupportResponse{
		Id: res.ID,
		Sender: &emailpb.SupportSenderInfo{
			UserId:  res.Sender.UserID,
			Email:   res.Sender.Email,
			Name:    res.Sender.Name,
			Surname: res.Sender.Surname,
		},
		Recipients:  res.Recipients,
		Header:      res.Header,
		Body:        res.Body,
		CreatedAt:   timestamppb.New(res.CreatedAt),
		IsAnonymous: res.IsAnonymous,
		IsDraft:     res.IsDraft,
		Attachments: pbAtt,
	}, nil
}
