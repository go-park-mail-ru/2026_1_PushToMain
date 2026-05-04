package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestS3Client(t *testing.T, handler http.Handler) *s3.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return s3.New(s3.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(server.URL),
		Credentials:  aws.AnonymousCredentials{},
	})
}

func TestRepository_New(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := newTestS3Client(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut && r.URL.Path == "/avatars" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}))

		repo, err := New(client)
		require.NoError(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, repo.s3)
	})

	t.Run("nil client", func(t *testing.T) {
		repo, err := New(nil)
		assert.Nil(t, repo)
		assert.ErrorIs(t, err, ErrS3ClientNotInited)
	})

	t.Run("bucket creation fails", func(t *testing.T) {
		client := newTestS3Client(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "access denied", http.StatusForbidden)
		}))

		repo, err := New(client)
		assert.Nil(t, repo)
		assert.ErrorIs(t, err, ErrS3CreateBucket)
	})
}

func TestRepository_UploadAvatar(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := newTestS3Client(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut && r.URL.Path == "/avatars" {
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.Method == http.MethodPut && r.URL.Path == "/avatars/users/123/avatar" {
				body, _ := io.ReadAll(r.Body)
				assert.Equal(t, "image data", string(body))
				assert.Equal(t, "image/jpeg", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}))

		repo, err := New(client)
		require.NoError(t, err)

		file := bytes.NewReader([]byte("image data"))
		key, err := repo.UploadAvatar(context.Background(), 123, file, int64(file.Len()))
		assert.NoError(t, err)
		assert.Equal(t, "users/123/avatar", key)
	})

	t.Run("upload fails", func(t *testing.T) {
		client := newTestS3Client(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut && r.URL.Path == "/avatars" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))

		repo, err := New(client)
		require.NoError(t, err)

		file := bytes.NewReader([]byte("image data"))
		key, err := repo.UploadAvatar(context.Background(), 123, file, int64(file.Len()))
		assert.Empty(t, key)
		assert.ErrorIs(t, err, ErrS3Err)
	})
}

func TestRepository_DeleteAvatar(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := newTestS3Client(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut && r.URL.Path == "/avatars" {
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.Method == http.MethodDelete && r.URL.Path == "/avatars/users/123/avatar" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.NotFound(w, r)
		}))

		repo, err := New(client)
		require.NoError(t, err)

		err = repo.DeleteAvatar(context.Background(), 123)
		assert.NoError(t, err)
	})

	t.Run("delete fails", func(t *testing.T) {
		client := newTestS3Client(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut && r.URL.Path == "/avatars" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}))

		repo, err := New(client)
		require.NoError(t, err)

		err = repo.DeleteAvatar(context.Background(), 123)
		assert.ErrorIs(t, err, ErrS3Err)
	})
}

func TestMakeAvatarPath(t *testing.T) {
	assert.Equal(t, "users/123/avatar", makeAvatarPath(123))
	assert.Equal(t, "users/0/avatar", makeAvatarPath(0))
}
