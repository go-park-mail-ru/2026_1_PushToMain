package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	userpb.UnimplementedUserServiceServer
	service Service
}

type Service interface {
	GetMe(ctx context.Context, userID int64) (*service.GetMeResult, error)
}

func New(svc Service) *Server {
	return &Server{
		service: svc,
	}
}

func (s *Server) GetUserById(
	ctx context.Context,
	req *userpb.GetUserByIdRequest,
) (*userpb.GetUserByIdResponse, error) {

	user, err := s.service.GetMe(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	var isMale bool
	if user.IsMale != nil {
		isMale = *user.IsMale
	}

	var birthdate string
	if user.Birthdate != nil { // если *time.Time
		birthdate = user.Birthdate.String()
	}

	return &userpb.GetUserByIdResponse{
		User: &userpb.User{
			Id:        user.UserID,
			Email:     user.Email,
			Name:      user.Name,
			Surname:   user.Surname,
			ImagePath: user.ImagePath,
			IsMale:    isMale,
			Birthdate: birthdate,
		},
	}, nil
}

func (s *Server) UserExists(
	ctx context.Context,
	req *userpb.UserExistsRequest,
) (*userpb.UserExistsResponse, error) {

	_, err := s.service.GetMe(ctx, req.UserId)

	return &userpb.UserExistsResponse{
		Exists: err == nil,
	}, nil
}
