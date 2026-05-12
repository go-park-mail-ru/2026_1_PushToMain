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
			Id:        email.ID,
			SenderId:  email.SenderID,
			Header:    email.Header,
			Body:      email.Body,
			CreatedAt: email.CreatedAt.String(),
		},
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
		})
	}

	return resp, nil
}
