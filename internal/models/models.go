package models

type ID string
type EmailName string

type Email struct {
	EmailID ID `json:"email-id"`
	From EmailName `json:"from"`
	To []EmailName `json:"to"`
	Header string `json:"header"`
	Body string `json:"body"`
}