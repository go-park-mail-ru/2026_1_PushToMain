package lmtp

import (
	"context"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
)

type Receiver interface {
	ReceiveExternalEmail(ctx context.Context, from string, to []string, header, body string) error
}

type Backend struct {
	receiver Receiver
}

func NewBackend(r Receiver) *Backend {
	return &Backend{receiver: r}
}

func (b *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{backend: b}, nil
}

type Session struct {
	backend *Backend
	from    string
	to      []string
}

func (s *Session) Mail(from string, _ *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}

	subject := msg.Header.Get("Subject")

	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		return err
	}
	body := strings.TrimSpace(string(bodyBytes))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.backend.receiver.ReceiveExternalEmail(ctx, s.from, s.to, subject, body)
}

func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *Session) Logout() error {
	return nil
}

func NewServer(r Receiver, addr string) *smtp.Server {
	be := NewBackend(r)
	srv := smtp.NewServer(be)

	srv.Addr = addr
	srv.LMTP = true
	srv.Domain = "email-service"
	srv.ReadTimeout = 30 * time.Second
	srv.WriteTimeout = 30 * time.Second
	srv.MaxMessageBytes = 26 * 1024 * 1024
	srv.MaxRecipients = 100
	srv.AllowInsecureAuth = true

	return srv
}
