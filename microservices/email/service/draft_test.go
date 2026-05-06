package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetDrafts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		now := time.Now().UTC()

		repo.EXPECT().CountDraftsByUser(ctx, int64(1)).Return(5, nil)
		repo.EXPECT().GetDrafts(ctx, int64(1), 10, 0).Return([]models.Draft{
			{ID: 1, SenderID: 1, Header: "Draft 1", Body: "body", Receivers: []string{"a"}, CreatedAt: now, UpdatedAt: now},
		}, nil)

		res, err := svc.GetDrafts(ctx, service.GetDraftsInput{UserID: 1, Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 5, res.Total)
		assert.Len(t, res.Drafts, 1)
		assert.Equal(t, int64(1), res.Drafts[0].ID)
		assert.Equal(t, "Draft 1", res.Drafts[0].Header)
	})
}

func TestService_DeleteDrafts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().DeleteDraftsBatch(ctx, int64(1), []int64{1, 2}).Return(nil)
		err := svc.DeleteDrafts(ctx, service.DeleteDraftsInput{UserID: 1, DraftIDs: []int64{1, 2}})
		assert.NoError(t, err)
	})
}
