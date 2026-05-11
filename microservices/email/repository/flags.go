package repository

import "context"

func (r *Repository) SetReadBatch(ctx context.Context, userID int64, emailIDs []int64, read bool) error {
}

func (r *Repository) SetStarredBatch(ctx context.Context, userID int64, emailIDs []int64, starred bool) error {
}

func (r *Repository) SetSpamBatch(ctx context.Context, userID int64, emailIDs []int64, spam bool) error {
}

func (r *Repository) SetDeletedBatch(ctx context.Context, userID int64, emailIDs []int64, deleted bool) error {
}

func (r *Repository) DeleteUserEmailsBatch(ctx context.Context, userID int64, emailIDs []int64) error {
}
