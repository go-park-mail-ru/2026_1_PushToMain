package smtp

import (
	"errors"
	"fmt"
	"net/smtp"
)

var (
	ErrRecipientListIsEmpty = errors.New("recipients list is empty")
)

type SmtpClient struct {
	Host     string
	Port     string
	Username string
	Password string
}

func NewSmtpClient(host, port, username, password string) *SmtpClient {
	return &SmtpClient{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func (c *SmtpClient) SendEmail(from string, to []string, subject, body string) error {
	if len(to) == 0 {
		return ErrRecipientListIsEmpty
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", from, to[0], subject, body)

	addr := c.Host + ":" + c.Port

	// We use NO auth
	err := smtp.SendMail(addr, nil, from, to, []byte(msg))
	if err != nil {
		return fmt.Errorf("cannot send email: %v", err)
	}

	return nil
}
