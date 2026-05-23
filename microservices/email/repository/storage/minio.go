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
	return &Repository{
		s3: client,
	}
}

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
	return fmt.Sprintf(
		"emails/%d/%s",
		emailID,
		filename,
	)
}
