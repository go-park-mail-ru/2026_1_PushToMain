package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/minio"
)

const bucketName = "attachments"

var (
	ErrS3ClientNotInited = errors.New("s3 client not inited")
	ErrS3CreateBucket    = errors.New("failed to create bucket")
	ErrS3Err             = errors.New("failed to exec ")
)

type Repository struct {
	s3            *s3.Client
	presignClient *s3.PresignClient
}

func New(client *s3.Client) (*Repository, error) {
	if client == nil {
		return nil, ErrS3ClientNotInited
	}

	err := minio.CreateBucket(client, bucketName)
	if err != nil {
		return nil, ErrS3CreateBucket
	}

	return &Repository{
		s3:            client,
		presignClient: s3.NewPresignClient(client),
	}, nil
}

// UploadAttachment streams file bytes into object storage and returns the storage key.
func (r *Repository) UploadAttachment(
	ctx context.Context,
	emailID int64,
	filename string,
	file io.Reader,
	size int64,
	contentType string,
) (string, error) {
	key := makeAttachmentPath(emailID, filename)

	_, err := r.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(key),
		Body:          file,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

// DownloadAttachment opens a streaming reader for the given storage key.
// The caller is responsible for closing the returned ReadCloser.
func (r *Repository) DownloadAttachment(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	out, err := r.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return out.Body, nil
}

// DeleteAttachment removes the object identified by key from storage.
func (r *Repository) DeleteAttachment(
	ctx context.Context,
	key string,
) error {
	_, err := r.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	return err
}

func makeAttachmentPath(emailID int64, filename string) string {
	return fmt.Sprintf("emails/%d/%s", emailID, filename)
}
