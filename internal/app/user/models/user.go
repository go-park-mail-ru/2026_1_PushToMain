package models

import "time"

type User struct {
	ID        int64
	Email     string
	Password  string
	Name      string
	Surname   string
	ImagePath string
	IsMale    *bool
	Birthdate *time.Time
}
