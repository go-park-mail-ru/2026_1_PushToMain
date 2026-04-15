package minio

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBucket(t *testing.T) {
	tests := []struct {
		name       string
		bucket     string
		handler    http.HandlerFunc
		wantErr    bool
		errMessage string
	}{
		{
			name:   "bucket already exists",
			bucket: "existing-bucket",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusOK)
					return
				}
				http.NotFound(w, r)
			},
			wantErr: false,
		},
		{
			name:   "bucket does not exist, create succeeds",
			bucket: "new-bucket",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodPut && r.URL.Path == "/new-bucket" {
					w.WriteHeader(http.StatusOK)
					return
				}
				http.NotFound(w, r)
			},
			wantErr: false,
		},
		{
			name:   "create bucket fails",
			bucket: "fail-bucket",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodPut {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				http.NotFound(w, r)
			},
			wantErr:    true,
			errMessage: "create bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := s3.New(s3.Options{
				Region:       "us-east-1",
				BaseEndpoint: aws.String(server.URL),
				Credentials:  aws.AnonymousCredentials{},
				UsePathStyle: true,
			})

			err := CreateBucket(client, tt.bucket)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name       string
		bucket     string
		handler    http.HandlerFunc
		wantErr    bool
		errMessage string
	}{
		{
			name:   "successful ping",
			bucket: "my-bucket",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead && r.URL.Path == "/my-bucket" {
					w.WriteHeader(http.StatusOK)
					return
				}
				http.NotFound(w, r)
			},
			wantErr: false,
		},
		{
			name:   "bucket not found",
			bucket: "missing-bucket",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				http.NotFound(w, r)
			},
			wantErr:    true,
			errMessage: "minio ping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := s3.New(s3.Options{
				Region:       "us-east-1",
				BaseEndpoint: aws.String(server.URL),
				Credentials:  aws.AnonymousCredentials{},
				UsePathStyle: true,
			})

			err := Ping(client, tt.bucket)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
