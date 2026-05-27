package models

import "time"

type Folder struct {
	ID   int64
	Name string
}

type User struct {
	ID        int64
	Email     string
	Password  string
	Name      string
	Surname   string
	ImagePath string
	IsMale    *bool
	Birthdate *time.Time

	Folders []Folder
}
