package smtp

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"path/filepath"
	"strings"
)

var ErrRecipientListIsEmpty = errors.New("recipients list is empty")

type Attachment struct {
	Filename string
	Data     []byte
	MIMEType string
}

type Message struct {
	fromAddr    string
	fromDisplay string
	to          []string
	subject     string
	text        string
	html        string
	attachments []Attachment
}

func NewMessage() *Message {
	return &Message{}
}

func (m *Message) From(name, surname, addr string) *Message {
	fullName := strings.TrimSpace(name + " " + surname)
	m.fromAddr = addr
	m.fromDisplay = (&mail.Address{Name: fullName, Address: addr}).String()
	return m
}

func (m *Message) Subject(s string) *Message {
	m.subject = s
	return m
}

func (m *Message) Text(body string) *Message {
	m.text = body
	return m
}

func (m *Message) Html(body string) *Message {
	m.html = body
	return m
}

func (m *Message) To(addrs ...string) *Message {
	m.to = append(m.to, addrs...)
	return m
}

func (m *Message) Attach(filename string, data []byte, mimeType string) *Message {
	if mimeType == "" {
		mimeType = detectMimeByExtension(filepath.Ext(filename))
	}

	m.attachments = append(m.attachments, Attachment{filename, data, mimeType})
	return m
}

func (m *Message) Send(c *SmtpClient) error { return c.Send(m) }

type SmtpClient struct {
	Host string
	Port string
}

func NewSmtpClient(host, port, _, _ string) *SmtpClient {
	return &SmtpClient{Host: host, Port: port}
}

func (c *SmtpClient) SendPlainText(to []string, subject, body string) error {
	return c.Send(NewMessage().To(to...).Subject(subject).Text(body))
}

func (c *SmtpClient) SendEmail(name string, surname string, from string, to []string, subject, body string, attachments []Attachment) error {
	msg := NewMessage().From(name, surname, from).To(to...).Subject(subject).Text(body)
	for _, a := range attachments {
		msg.Attach(a.Filename, a.Data, a.MIMEType)
	}
	return c.Send(msg)
}

func (c *SmtpClient) Send(m *Message) error {
	if len(m.to) == 0 {
		return ErrRecipientListIsEmpty
	}
	raw, err := m.build()
	if err != nil {
		return fmt.Errorf("smtp: build message: %w", err)
	}
	if err := smtp.SendMail(c.Host+":"+c.Port, nil, m.fromAddr, m.to, raw); err != nil {
		return fmt.Errorf("smtp: send: %w", err)
	}
	return nil
}

func (m *Message) build() ([]byte, error) {
	var buf bytes.Buffer

	hdr := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }
	hdr("From", m.fromDisplay)
	hdr("To", strings.Join(m.to, ", "))
	hdr("Subject", encodeRFC2047(m.subject))
	hdr("MIME-Version", "1.0")

	hasHtml := m.html != ""
	hasText := m.text != ""
	hasFiles := len(m.attachments) > 0

	switch {
	case !hasHtml && !hasFiles:
		hdr("Content-Type", "text/plain; charset=UTF-8")
		hdr("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		qp := quotedprintable.NewWriter(&buf)
		if _, err := qp.Write([]byte(m.text)); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case hasHtml && !hasFiles:
		mw := multipart.NewWriter(&buf)
		hdr("Content-Type", "multipart/alternative; boundary=\""+mw.Boundary()+"\"")
		buf.WriteString("\r\n")
		if hasText {
			if err := writeTextPart(mw, m.text); err != nil {
				return nil, err
			}
		}
		if err := writeHtmlPart(mw, m.html); err != nil {
			return nil, err
		}
		if err := mw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		mw := multipart.NewWriter(&buf)
		hdr("Content-Type", "multipart/mixed; boundary=\""+mw.Boundary()+"\"")
		buf.WriteString("\r\n")

		if hasHtml {
			if err := writeAlternativeBlock(mw, m.text, m.html); err != nil {
				return nil, err
			}
		} else if hasText {
			if err := writeTextPart(mw, m.text); err != nil {
				return nil, err
			}
		}

		for _, a := range m.attachments {
			if err := writeFilePart(mw, a); err != nil {
				return nil, err
			}
		}

		if err := mw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}

func writeTextPart(mw *multipart.Writer, text string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "text/plain; charset=UTF-8")
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(pw)
	if _, err = qp.Write([]byte(text)); err != nil {
		return err
	}
	return qp.Close()
}

func writeHtmlPart(mw *multipart.Writer, html string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "text/html; charset=UTF-8")
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(pw)
	if _, err = qp.Write([]byte(html)); err != nil {
		return err
	}
	return qp.Close()
}

func writeAlternativeBlock(outer *multipart.Writer, text, html string) error {
	var altBuf bytes.Buffer
	inner := multipart.NewWriter(&altBuf)

	if text != "" {
		if err := writeTextPart(inner, text); err != nil {
			return err
		}
	}
	if err := writeHtmlPart(inner, html); err != nil {
		return err
	}
	if err := inner.Close(); err != nil {
		return err
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "multipart/alternative; boundary=\""+inner.Boundary()+"\"")
	pw, err := outer.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = pw.Write(altBuf.Bytes())
	return err
}

func writeFilePart(mw *multipart.Writer, a Attachment) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", a.MIMEType)
	h.Set("Content-Transfer-Encoding", "base64")
	h.Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, mime.QEncoding.Encode("UTF-8", a.Filename)))
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	enc := base64.NewEncoder(base64.StdEncoding, &lineBreaker{w: pw})
	if _, err := enc.Write(a.Data); err != nil {
		return err
	}
	return enc.Close()
}

type lineBreaker struct {
	w interface {
		Write([]byte) (int, error)
	}
	n int
}

func (lb *lineBreaker) Write(p []byte) (int, error) {
	const limit = 76
	written := 0
	for len(p) > 0 {
		if lb.n == limit {
			if _, err := lb.w.Write([]byte("\r\n")); err != nil {
				return written, err
			}
			lb.n = 0
		}
		chunk := p
		if rem := limit - lb.n; len(chunk) > rem {
			chunk = chunk[:rem]
		}
		n, err := lb.w.Write(chunk)
		written += n
		lb.n += n
		p = p[n:]
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func encodeRFC2047(s string) string {
	for _, r := range s {
		if r > 127 {
			return mime.BEncoding.Encode("UTF-8", s)
		}
	}
	return s
}

const (
	mimeApplicationPDF   = "application/pdf"
	mimeImagePNG         = "image/png"
	mimeImageJPEG        = "image/jpeg"
	mimeTextPlain        = "text/plain"
	mimeTextHTML         = "text/html"
	mimeApplicationOctet = "application/octet-stream"
)

func detectMimeByExtension(ext string) string {
	types := map[string]string{
		".pdf":  mimeApplicationPDF,
		".png":  mimeImagePNG,
		".jpg":  mimeImageJPEG,
		".jpeg": mimeImageJPEG,
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".txt":  mimeTextPlain,
		".csv":  "text/csv",
		".html": mimeTextHTML,
		".htm":  mimeTextHTML,
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
	}
	if v, ok := types[strings.ToLower(ext)]; ok {
		return v
	}
	return mimeApplicationOctet
}
