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

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type ParsedEmail struct {
	Body        string
	Attachments []Attachment
}

type Receiver interface {
	ReceiveExternalEmail(ctx context.Context, from string, to []string, subject string, parsed ParsedEmail) error
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

	parsed, err := extractContent(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.backend.receiver.ReceiveExternalEmail(ctx, s.from, s.to, subject, parsed)
}

func extractContent(msg *mail.Message) (ParsedEmail, error) {
	var out ParsedEmail
	ct := msg.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		b, _ := io.ReadAll(msg.Body)
		out.Body = string(b)
		return out, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		err := walkMultipart(msg.Body, params["boundary"], &out)
		return out, err
	}

	body, err := decodeSinglePart(msg.Body, msg.Header.Get("Content-Transfer-Encoding"))
	out.Body = body
	return out, err
}

func walkMultipart(r io.Reader, boundary string, out *ParsedEmail) error {
	if boundary == "" {
		b, _ := io.ReadAll(r)
		if out.Body == "" {
			out.Body = string(b)
		}
		return nil
	}

	var plain, html string
	mr := multipart.NewReader(r, boundary)
	dec := new(mime.WordDecoder)

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		partCT := p.Header.Get("Content-Type")
		if partCT == "" {
			partCT = "text/plain"
		}
		mt, partParams, _ := mime.ParseMediaType(partCT)

		disposition, dispParams, _ := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
		filename := dispParams["filename"]
		if filename == "" {
			filename = partParams["name"]
		}

		isAttachment := disposition == "attachment" || filename != ""
		if isAttachment {
			data, err := readDecoded(p, p.Header.Get("Content-Transfer-Encoding"))
			if err != nil {
				continue
			}
			if decoded, err := dec.DecodeHeader(filename); err == nil {
				filename = decoded
			}
			if filename == "" {
				filename = "attachment"
			}
			out.Attachments = append(out.Attachments, Attachment{
				Filename:    filename,
				ContentType: mt,
				Data:        data,
			})
			continue
		}

		if strings.HasPrefix(mt, "multipart/") {
			if err := walkMultipart(p, partParams["boundary"], out); err != nil {
				return err
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

	if out.Body == "" {
		if plain != "" {
			out.Body = plain
		} else {
			out.Body = html
		}
	}
	return nil
}

func readDecoded(r io.Reader, cte string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		return io.ReadAll(base64.NewDecoder(base64.StdEncoding, r))
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(r))
	default:
		return io.ReadAll(r)
	}
}

func decodeSinglePart(r io.Reader, cte string) (string, error) {
	b, err := readDecoded(r, cte)
	return string(b), err
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
