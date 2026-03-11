package models


type Email struct {
	EmailID string `json:"email-id"`
	From string `json:"from"`
	To []string `json:"to"`
	Header string `json:"header"`
	Body string `json:"body"`
}