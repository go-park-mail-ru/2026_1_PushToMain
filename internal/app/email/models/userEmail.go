package models

type UserEmail struct {
	ID         int64
	EmailID    int64
	ReceiverID int64
	IsRead     bool
}
