// internal/pkg/clients/folder/folder.go
package folder

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/grpcx"
	folderpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/folder"

	"google.golang.org/grpc"
)

type Client struct {
	client folderpb.FolderServiceClient
	conn   *grpc.ClientConn
}

func New(addr string) (*Client, error) {
	conn, err := grpcx.NewClient(addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: folderpb.NewFolderServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetUserFolders(
	ctx context.Context,
	userID int64,
) ([]*folderpb.Folder, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := c.client.GetUserFolders(
		ctx,
		&folderpb.GetUserFoldersRequest{
			UserId: userID,
		},
	)
	if err != nil {
		return nil, err
	}

	return resp.Folders, nil
}
