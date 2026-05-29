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

	// AcceptAnonymous — указатель, чтобы различать "не задано в запросе"
	// и "явно установлено false" в UpdateProfile (PATCH-семантика, как у IsMale).
	// На уровне БД колонка NOT NULL DEFAULT false, так что nil тут только
	// для UpdateProfile-инпута, а Read-операции всегда возвращают конкретное значение.
	AcceptAnonymous *bool

	Folders []Folder
}
