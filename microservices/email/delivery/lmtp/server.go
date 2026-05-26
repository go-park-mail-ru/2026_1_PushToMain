package lmtp

import (
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
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

	dec := new(mime.WordDecoder)
	subject, err := dec.DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		subject = msg.Header.Get("Subject")
	}

	body, err := extractBody(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.backend.receiver.ReceiveExternalEmail(ctx, s.from, s.to, subject, body)
}

func extractBody(msg *mail.Message) (string, error) {
	ct := msg.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		b, _ := io.ReadAll(msg.Body)
		return string(b), nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		return readMultipart(msg.Body, params["boundary"])
	}

	return decodeSinglePart(msg.Body, msg.Header.Get("Content-Transfer-Encoding"))
}

func readMultipart(r io.Reader, boundary string) (string, error) {
	if boundary == "" {
		b, _ := io.ReadAll(r)
		return string(b), nil
	}
	mr := multipart.NewReader(r, boundary)
	var plain, html string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		partCT := p.Header.Get("Content-Type")
		mt, partParams, _ := mime.ParseMediaType(partCT)

		if strings.HasPrefix(mt, "multipart/") {
			nested, err := readMultipart(p, partParams["boundary"])
			if err != nil {
				return "", err
			}
			if plain == "" {
				plain = nested
			}
			continue
		}

		decoded, err := decodeSinglePart(p, p.Header.Get("Content-Transfer-Encoding"))
		if err != nil {
			continue
		}
		switch mt {
		case "text/plain":
			if plain == "" {
				plain = decoded
			}
		case "text/html":
			if html == "" {
				html = decoded
			}
		}
	}
	if plain != "" {
		return plain, nil
	}
	return html, nil
}

func decodeSinglePart(r io.Reader, cte string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "quoted-printable":
		b, err := io.ReadAll(quotedprintable.NewReader(r))
		return string(b), err
	case "base64":
		b, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, r))
		return string(b), err
	default:
		b, err := io.ReadAll(r)
		return string(b), err
	}
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
