package user

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/grpcx"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"

	"google.golang.org/grpc"
)

type Client struct {
	client userpb.UserServiceClient
	conn   *grpc.ClientConn
}

func New(addr string) (*Client, error) {

	conn, err := grpcx.NewClient(addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: userpb.NewUserServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetUserByID(
	ctx context.Context,
	userID int64,
) (*userpb.User, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.GetUserById(
		ctx,
		&userpb.GetUserByIdRequest{
			UserId: userID,
		},
	)

	if err != nil {
		return nil, err
	}

	return resp.User, nil
}

func (c *Client) UserExists(
	ctx context.Context,
	userID int64,
) (bool, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.UserExists(
		ctx,
		&userpb.UserExistsRequest{
			UserId: userID,
		},
	)

	if err != nil {
		return false, err
	}

	return resp.Exists, nil
}
