package repository

import "errors"

var ErrUserExists = errors.New("user exists")
var ErrUserNotFound = errors.New("user not found")
