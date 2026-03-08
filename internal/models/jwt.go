package models

type JwtPayload struct {
	Email string
	Exp   int64
}
