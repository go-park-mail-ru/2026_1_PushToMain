package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/service"
	pb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/folder"
)

type FolderGrpcServer struct {
	pb.UnimplementedFolderServiceServer
	folderService *service.Service
}

func New(folderService *service.Service) *FolderGrpcServer {
	return &FolderGrpcServer{folderService: folderService}
}

func (s *FolderGrpcServer) GetUserFolders(ctx context.Context, req *pb.GetUserFoldersRequest) (*pb.GetUserFoldersResponse, error) {
	folders, err := s.folderService.GetUserFolders(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get folders: %v", err)
	}

	pbFolders := make([]*pb.Folder, len(folders))
	for i, f := range folders {
		pbFolders[i] = &pb.Folder{
			Id:   f.ID,
			Name: f.Name,
		}
	}

	return &pb.GetUserFoldersResponse{Folders: pbFolders}, nil
}
