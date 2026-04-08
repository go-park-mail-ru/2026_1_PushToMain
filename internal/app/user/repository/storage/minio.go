package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/minio"
)

var (
	ErrS3ClientNotInited = errors.New("s3 client not inited")
)

const (
	bucketName     = "avatars"
	presignedTTL   = 24 * time.Hour
	avatarFileType = "image/jpeg"
)

type Repository struct {
	s3            *s3.Client
	presignClient *s3.PresignClient
}

func New(client *s3.Client) *Repository {
	err := minio.CreateBucket(client, "avatars")
	if err != nil {
		return nil
	}

	return &Repository{
		s3:            client,
		presignClient: s3.NewPresignClient(client),
	}
}

func (r *Repository) UploadAvatar(ctx context.Context, userID int64, file io.Reader, size int64) (string, error) {
	if r.s3 == nil {
		return "", ErrS3ClientNotInited
	}

	key := makeAvatarPath(userID)

	_, err := r.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(key),
		Body:          file,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(avatarFileType),
	})
	if err != nil {
		return "", fmt.Errorf("save avatar for user %d: %w", userID, err)
	}
	return key, nil
}

func makeAvatarPath(userID int64) string {
	return fmt.Sprintf("users/%d/avatar", userID)
}
