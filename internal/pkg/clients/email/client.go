package email

import (
	"context"
	"time"

	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client emailpb.EmailServiceClient
	conn   *grpc.ClientConn
}

func New(addr string) (*Client, error) {

	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		return nil, err
	}

	return &Client{
		client: emailpb.NewEmailServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetEmailByID(
	ctx context.Context,
	emailID,
	userID int64,
) (*emailpb.Email, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.GetEmailById(
		ctx,
		&emailpb.GetEmailByIdRequest{
			EmailId: emailID,
			UserId:  userID,
		},
	)

	if err != nil {
		return nil, err
	}

	return resp.Email, nil
}

func (c *Client) CheckEmailAccess(
	ctx context.Context,
	emailID,
	userID int64,
) (bool, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.CheckEmailAccess(
		ctx,
		&emailpb.CheckEmailAccessRequest{
			EmailId: emailID,
			UserId:  userID,
		},
	)

	if err != nil {
		return false, err
	}

	return resp.HasAccess, nil
}

func (c *Client) GetEmailsByIDs(
	ctx context.Context,
	emailIDs []int64,
	userID int64,
) (*emailpb.GetEmailsByIdsResponse, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	return c.client.GetEmailsByIds(
		ctx,
		&emailpb.GetEmailsByIdsRequest{
			EmailIds: emailIDs,
			UserId:   userID,
		},
	)
}
