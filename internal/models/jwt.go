package models

type JwtPayload struct {
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}
