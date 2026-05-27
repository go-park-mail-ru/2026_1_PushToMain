package email

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/grpcx"
	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"

	"google.golang.org/grpc"
)

type Client struct {
	client emailpb.EmailServiceClient
	conn   *grpc.ClientConn
}

func New(addr string) (*Client, error) {

	conn, err := grpcx.NewClient(addr)
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
func (c *Client) GetUserEmailID(
	ctx context.Context,
	emailID,
	userID int64,
) (int64, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.GetUserEmailID(
		ctx,
		&emailpb.GetUserEmailIDRequest{
			EmailId: emailID,
			UserId:  userID,
		},
	)

	if err != nil {
		return 0, err
	}

	return resp.UserEmailId, nil
}

func (c *Client) GetEmailIdsByUserEmailIds(
	ctx context.Context,
	userEmailIDs []int64,
) ([]int64, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.GetEmailIdsByUserEmailIds(
		ctx,
		&emailpb.GetEmailIdsByUserEmailIdsRequest{
			UserEmailIds: userEmailIDs,
		},
	)

	if err != nil {
		return nil, err
	}

	return resp.EmailIds, nil
}

func (c *Client) SwitchIsInbox(
	ctx context.Context,
	emailID,
	userID int64,
) (bool, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		3*time.Second,
	)

	defer cancel()

	resp, err := c.client.SwitchIsInbox(
		ctx,
		&emailpb.SwitchIsInboxRequest{
			EmailId: emailID,
			UserId:  userID,
		},
	)

	if err != nil {
		return false, err
	}

	return resp.Success, nil
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
