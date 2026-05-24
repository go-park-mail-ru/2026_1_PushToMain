package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const bucketName = "attachments"

type Repository struct {
	s3 *s3.Client
}

func New(client *s3.Client) *Repository {
	return &Repository{s3: client}
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
