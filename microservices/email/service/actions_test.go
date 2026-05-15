package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// ---------- Trash ----------
func TestService_Trash(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Trash(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().SetTrashedBatch(ctx, int64(1), []int64{10, 20}, true).Return(nil)
		err := svc.Trash(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{10, 20}})
		assert.NoError(t, err)
	})
	t.Run("repo error", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().SetTrashedBatch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("fail"))
		err := svc.Trash(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{1}})
		assert.Error(t, err)
	})
}

// ---------- Untrash ----------
func TestService_Untrash(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Untrash(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().SetTrashedBatch(ctx, int64(1), []int64{5}, false).Return(nil)
		err := svc.Untrash(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{5}})
		assert.NoError(t, err)
	})
}

// ---------- Favorite ----------
func TestService_Favorite(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Favorite(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().SetStarredBatch(ctx, int64(2), []int64{7}, true).Return(nil)
		err := svc.Favorite(ctx, service.BatchInput{UserID: 2, EmailIDs: []int64{7}})
		assert.NoError(t, err)
	})
}

// ---------- Unfavorite ----------
func TestService_Unfavorite(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Unfavorite(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().SetStarredBatch(ctx, int64(3), []int64{99}, false).Return(nil)
		err := svc.Unfavorite(ctx, service.BatchInput{UserID: 3, EmailIDs: []int64{99}})
		assert.NoError(t, err)
	})
}

// ---------- Spam ----------
func TestService_Spam(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Spam(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().MarkSendersAsSpamBatch(ctx, int64(10), []int64{1, 2}).Return(nil)
		err := svc.Spam(ctx, service.BatchInput{UserID: 10, EmailIDs: []int64{1, 2}})
		assert.NoError(t, err)
	})
}

// ---------- Unspam ----------
func TestService_Unspam(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Unspam(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().SetSpamBatch(ctx, int64(4), []int64{8}, false).Return(nil)
		err := svc.Unspam(ctx, service.BatchInput{UserID: 4, EmailIDs: []int64{8}})
		assert.NoError(t, err)
	})
}

// ---------- UnmarkSpamSenders ----------
func TestService_UnmarkSpamSenders(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.UnmarkSpamSenders(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().UnmarkSendersAsSpamBatch(ctx, int64(5), []int64{3, 4}).Return(nil)
		err := svc.UnmarkSpamSenders(ctx, service.BatchInput{UserID: 5, EmailIDs: []int64{3, 4}})
		assert.NoError(t, err)
	})
}

// ---------- Delete ----------
func TestService_Delete(t *testing.T) {
	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		err := svc.Delete(context.Background(), service.BatchInput{UserID: 1})
		assert.ErrorIs(t, err, service.ErrEmptyIDs)
	})

	t.Run("all soft delete (not deleted)", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUserEmailFlags(ctx, int64(10), int64(1), false).
			Return(&models.UserEmail{IsDeleted: false}, nil)
		repo.EXPECT().GetUserEmailFlags(ctx, int64(20), int64(1), false).
			Return(&models.UserEmail{IsDeleted: false}, nil)
		repo.EXPECT().SetTrashedBatch(ctx, int64(1), []int64{10, 20}, true).Return(nil)
		err := svc.Delete(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{10, 20}})
		assert.NoError(t, err)
	})

	t.Run("all hard delete (already deleted)", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUserEmailFlags(ctx, int64(100), int64(2), false).
			Return(&models.UserEmail{IsDeleted: true}, nil)
		repo.EXPECT().HardDeleteBatch(ctx, int64(2), []int64{100}).Return(nil)
		err := svc.Delete(ctx, service.BatchInput{UserID: 2, EmailIDs: []int64{100}})
		assert.NoError(t, err)
	})

	t.Run("mixed", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		// email 1: not deleted (soft)
		repo.EXPECT().GetUserEmailFlags(ctx, int64(1), int64(3), false).
			Return(&models.UserEmail{IsDeleted: false}, nil)
		// email 2: already deleted (hard)
		repo.EXPECT().GetUserEmailFlags(ctx, int64(2), int64(3), false).
			Return(&models.UserEmail{IsDeleted: true}, nil)
		// email 3: not found as receiver, found as sender and not deleted (soft)
		repo.EXPECT().GetUserEmailFlags(ctx, int64(3), int64(3), false).
			Return(nil, errors.New("not found"))
		repo.EXPECT().GetUserEmailFlags(ctx, int64(3), int64(3), true).
			Return(&models.UserEmail{IsDeleted: false}, nil)
		// email 4: not found as receiver, not found as sender (skip)
		repo.EXPECT().GetUserEmailFlags(ctx, int64(4), int64(3), false).
			Return(nil, errors.New("not found"))
		repo.EXPECT().GetUserEmailFlags(ctx, int64(4), int64(3), true).
			Return(nil, errors.New("not found"))

		repo.EXPECT().SetTrashedBatch(ctx, int64(3), []int64{1, 3}, true).Return(nil)
		repo.EXPECT().HardDeleteBatch(ctx, int64(3), []int64{2}).Return(nil)

		err := svc.Delete(ctx, service.BatchInput{UserID: 3, EmailIDs: []int64{1, 2, 3, 4}})
		assert.NoError(t, err)
	})

	t.Run("GetUserEmailFlags error propagates via batch", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUserEmailFlags(ctx, int64(10), int64(1), false).
			Return(&models.UserEmail{IsDeleted: false}, nil)
		repo.EXPECT().SetTrashedBatch(ctx, int64(1), []int64{10}, true).
			Return(errors.New("fail"))
		repo.EXPECT().GetUserEmailFlags(ctx, int64(99), int64(1), false).
			Return(nil, errors.New("db fail"))
		repo.EXPECT().GetUserEmailFlags(ctx, int64(99), int64(1), true).
			Return(nil, errors.New("db fail"))

		err := svc.Delete(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{10, 99}})
		assert.Error(t, err)
	})

	t.Run("SetTrashedBatch fails", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUserEmailFlags(ctx, int64(1), int64(1), false).
			Return(&models.UserEmail{IsDeleted: false}, nil)
		repo.EXPECT().SetTrashedBatch(ctx, int64(1), []int64{1}, true).
			Return(errors.New("trash error"))
		err := svc.Delete(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{1}})
		assert.Error(t, err)
	})

	t.Run("HardDeleteBatch fails", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUserEmailFlags(ctx, int64(2), int64(1), false).
			Return(&models.UserEmail{IsDeleted: true}, nil)
		repo.EXPECT().HardDeleteBatch(ctx, int64(1), []int64{2}).
			Return(errors.New("hard delete error"))
		err := svc.Delete(ctx, service.BatchInput{UserID: 1, EmailIDs: []int64{2}})
		assert.Error(t, err)
	})
}
