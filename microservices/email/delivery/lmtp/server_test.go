package lmtp

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

type capturedCall struct {
	from    string
	to      []string
	subject string
	parsed  ParsedEmail
}

type mockReceiver struct {
	calls []capturedCall
	err   error
}

func (m *mockReceiver) ReceiveExternalEmail(
	_ context.Context,
	from string,
	to []string,
	subject string,
	parsed ParsedEmail,
) error {
	m.calls = append(m.calls, capturedCall{from, to, subject, parsed})
	return m.err
}

func (m *mockReceiver) last() capturedCall {
	return m.calls[len(m.calls)-1]
}

func makeSession(recv *mockReceiver) *Session {
	return &Session{backend: &Backend{receiver: recv}}
}

func encodeQP(s string) string {
	var buf bytes.Buffer
	w := quotedprintable.NewWriter(&buf)
	w.Write([]byte(s))
	w.Close()
	return buf.String()
}

func encodeB64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func TestReadDecoded_Base64(t *testing.T) {
	data := []byte("hello binary \x00\x01\x02")
	encoded := base64.StdEncoding.EncodeToString(data)
	got, err := readDecoded(strings.NewReader(encoded), "base64")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got %v, want %v", got, data)
	}
}

func TestReadDecoded_Base64_CaseInsensitive(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("x"))
	got, err := readDecoded(strings.NewReader(encoded), "BASE64")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "x" {
		t.Errorf("got %q", got)
	}
}

func TestReadDecoded_QuotedPrintable(t *testing.T) {
	got, err := readDecoded(strings.NewReader("=D0=9F=D1=80=D0=B8=D0=B2=D0=B5=D1=8210"), "quoted-printable")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "Привет") {
		t.Errorf("got %q", got)
	}
}

func TestReadDecoded_Default_PassThrough(t *testing.T) {
	input := "plain text"
	for _, cte := range []string{"", "7bit", "8bit", "binary"} {
		got, err := readDecoded(strings.NewReader(input), cte)
		if err != nil {
			t.Fatalf("cte=%q: %v", cte, err)
		}
		if string(got) != input {
			t.Errorf("cte=%q: got %q", cte, got)
		}
	}
}

func TestReadDecoded_QuotedPrintable_CTE_WithSpaces(t *testing.T) {
	qp := encodeQP("hello")
	got, err := readDecoded(strings.NewReader(qp), "  quoted-printable  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q", got)
	}
}
func TestDecodeSinglePart_PlainText(t *testing.T) {
	got, err := decodeSinglePart(strings.NewReader("hello"), "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestDecodeSinglePart_QP_Cyrillic(t *testing.T) {
	qp := encodeQP("Привет")
	got, err := decodeSinglePart(strings.NewReader(qp), "quoted-printable")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Привет" {
		t.Errorf("got %q", got)
	}
}

func TestExtractContent_NoContentType_PlainFallback(t *testing.T) {
	msg, _ := mail.ReadMessage(strings.NewReader("From: a@b.com\r\n\r\nhello"))
	out, err := extractContent(msg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != "hello" {
		t.Errorf("body: %q", out.Body)
	}
	if len(out.Attachments) != 0 {
		t.Errorf("expected no attachments")
	}
}

func TestExtractContent_TextPlain_7bit(t *testing.T) {
	raw := "Content-Type: text/plain; charset=UTF-8\r\n\r\nHello world"
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	out, err := extractContent(msg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != "Hello world" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestExtractContent_TextPlain_QP(t *testing.T) {
	body := encodeQP("Тест контент")
	raw := "Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n" + body
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	out, err := extractContent(msg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != "Тест контент" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestExtractContent_TextPlain_Base64(t *testing.T) {
	body := encodeB64([]byte("base64 body"))
	raw := "Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" + body
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	out, err := extractContent(msg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != "base64 body" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestExtractContent_InvalidContentType_Fallback(t *testing.T) {
	raw := "Content-Type: not/valid/at/all\r\n\r\nfallback"
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	out, err := extractContent(msg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != "fallback" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestWalkMultipart_EmptyBoundary_Fallback(t *testing.T) {
	var out ParsedEmail
	err := walkMultipart(strings.NewReader("raw content"), "", &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != "raw content" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestWalkMultipart_TextPlainOnly(t *testing.T) {
	body := buildMultipart("boundary123",
		part{"Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable", encodeQP("Привет")},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "boundary123", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "Привет" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestWalkMultipart_HTMLFallback_WhenNoPlain(t *testing.T) {
	body := buildMultipart("b1",
		part{"Content-Type: text/html; charset=UTF-8", "<p>html</p>"},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "b1", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "<p>html</p>" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestWalkMultipart_PlainPreferredOverHTML(t *testing.T) {
	body := buildMultipart("bnd",
		part{"Content-Type: text/plain", "plain text"},
		part{"Content-Type: text/html", "<b>html</b>"},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "plain text" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestWalkMultipart_FirstPlainWins(t *testing.T) {
	body := buildMultipart("bnd",
		part{"Content-Type: text/plain", "first"},
		part{"Content-Type: text/plain", "second"},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "first" {
		t.Errorf("body: %q", out.Body)
	}
}

func TestWalkMultipart_BodyNotOverwritten(t *testing.T) {
	out := ParsedEmail{Body: "already set"}
	body := buildMultipart("bnd",
		part{"Content-Type: text/plain", "new"},
	)
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "already set" {
		t.Errorf("body should not be overwritten: %q", out.Body)
	}
}

func TestWalkMultipart_AttachmentByDisposition(t *testing.T) {
	data := []byte("PDF content")
	encoded := encodeB64(data)
	body := buildMultipart("bnd",
		part{"Content-Type: text/plain", "see attachment"},
		part{
			"Content-Type: application/pdf\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"report.pdf\"",
			encoded,
		},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "see attachment" {
		t.Errorf("body: %q", out.Body)
	}
	if len(out.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(out.Attachments))
	}
	a := out.Attachments[0]
	if a.Filename != "report.pdf" {
		t.Errorf("filename: %q", a.Filename)
	}
	if a.ContentType != "application/pdf" {
		t.Errorf("content type: %q", a.ContentType)
	}
	if !bytes.Equal(a.Data, data) {
		t.Errorf("data mismatch")
	}
}

func TestWalkMultipart_AttachmentByNameParam(t *testing.T) {
	data := []byte("image data")
	body := buildMultipart("bnd",
		part{
			"Content-Type: image/png; name=\"photo.png\"\r\nContent-Transfer-Encoding: base64",
			encodeB64(data),
		},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(out.Attachments))
	}
	if out.Attachments[0].Filename != "photo.png" {
		t.Errorf("filename: %q", out.Attachments[0].Filename)
	}
}

func TestWalkMultipart_AttachmentEmptyFilename_Default(t *testing.T) {
	body := buildMultipart("bnd",
		part{
			"Content-Type: application/octet-stream\r\nContent-Disposition: attachment",
			encodeB64([]byte("data")),
		},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(out.Attachments))
	}
	if out.Attachments[0].Filename != "attachment" {
		t.Errorf("filename: %q", out.Attachments[0].Filename)
	}
}

func TestWalkMultipart_MultipleAttachments(t *testing.T) {
	body := buildMultipart("bnd",
		part{"Content-Type: text/plain", "body"},
		part{
			"Content-Type: application/pdf\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"a.pdf\"",
			encodeB64([]byte("pdf")),
		},
		part{
			"Content-Type: image/png\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"b.png\"",
			encodeB64([]byte("png")),
		},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(out.Attachments))
	}
	if out.Attachments[0].Filename != "a.pdf" {
		t.Errorf("first: %q", out.Attachments[0].Filename)
	}
	if out.Attachments[1].Filename != "b.png" {
		t.Errorf("second: %q", out.Attachments[1].Filename)
	}
}

func TestWalkMultipart_CyrillicFilename_RFC2047(t *testing.T) {
	encodedFilename := "=?UTF-8?B?0LTQvtC60YPQvNC10L3RgjIucGRm?="
	body := buildMultipart("bnd",
		part{
			"Content-Type: application/pdf\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"" + encodedFilename + "\"",
			encodeB64([]byte("data")),
		},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(body), "bnd", &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(out.Attachments))
	}
	if out.Attachments[0].Filename != "документ2.pdf" {
		t.Errorf("filename: %q", out.Attachments[0].Filename)
	}
}

func TestWalkMultipart_Nested_MixedWithAlternative(t *testing.T) {
	innerBody := buildMultipart("inner",
		part{"Content-Type: text/plain", "plain"},
		part{"Content-Type: text/html", "<b>html</b>"},
	)
	outerBody := buildMultipart("outer",
		part{"Content-Type: multipart/alternative; boundary=\"inner\"", innerBody},
		part{
			"Content-Type: image/png\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"pic.png\"",
			encodeB64([]byte("imgdata")),
		},
	)
	var out ParsedEmail
	if err := walkMultipart(strings.NewReader(outerBody), "outer", &out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "plain" {
		t.Errorf("body: %q", out.Body)
	}
	if len(out.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(out.Attachments))
	}
	if out.Attachments[0].Filename != "pic.png" {
		t.Errorf("filename: %q", out.Attachments[0].Filename)
	}
}

func TestSession_Mail(t *testing.T) {
	s := makeSession(&mockReceiver{})
	if err := s.Mail("sender@example.com", nil); err != nil {
		t.Fatal(err)
	}
	if s.from != "sender@example.com" {
		t.Errorf("from: %q", s.from)
	}
}

func TestSession_Rcpt_Accumulates(t *testing.T) {
	s := makeSession(&mockReceiver{})
	s.Rcpt("a@a.com", nil)
	s.Rcpt("b@b.com", nil)
	if len(s.to) != 2 {
		t.Errorf("to len: %d", len(s.to))
	}
}

func TestSession_Reset(t *testing.T) {
	s := makeSession(&mockReceiver{})
	s.from = "x@x.com"
	s.to = []string{"a@a.com"}
	s.Reset()
	if s.from != "" {
		t.Errorf("from after reset: %q", s.from)
	}
	if len(s.to) != 0 {
		t.Errorf("to after reset: %v", s.to)
	}
}

func TestSession_Logout(t *testing.T) {
	s := makeSession(&mockReceiver{})
	if err := s.Logout(); err != nil {
		t.Errorf("logout: %v", err)
	}
}

func TestSession_Data_PlainText(t *testing.T) {
	recv := &mockReceiver{}
	s := makeSession(recv)
	s.from = "ext@gmail.com"
	s.to = []string{"user@e-smail.ru"}

	raw := "From: ext@gmail.com\r\nTo: user@e-smail.ru\r\nSubject: Test\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nHello!"
	if err := s.Data(strings.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	if len(recv.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(recv.calls))
	}
	c := recv.last()
	if c.from != "ext@gmail.com" {
		t.Errorf("from: %q", c.from)
	}
	if len(c.to) != 1 || c.to[0] != "user@e-smail.ru" {
		t.Errorf("to: %v", c.to)
	}
	if c.subject != "Test" {
		t.Errorf("subject: %q", c.subject)
	}
	if c.parsed.Body != "Hello!" {
		t.Errorf("body: %q", c.parsed.Body)
	}
}

func TestSession_Data_SubjectRFC2047(t *testing.T) {
	recv := &mockReceiver{}
	s := makeSession(recv)
	s.from = "a@b.com"
	s.to = []string{"u@e-smail.ru"}

	encodedSubj := "=?UTF-8?B?0J/RgNC40LLQtdGCLCDQvNC40YA=?="
	raw := "Subject: " + encodedSubj + "\r\nContent-Type: text/plain\r\n\r\nbody"
	if err := s.Data(strings.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	c := recv.last()
	if !strings.Contains(c.subject, "Привет") && !strings.Contains(c.subject, "мир") {
		t.Errorf("subject not decoded: %q", c.subject)
	}
}

func TestSession_Data_MultipartWithAttachment(t *testing.T) {
	recv := &mockReceiver{}
	s := makeSession(recv)
	s.from = "ext@gmail.com"
	s.to = []string{"u@e-smail.ru"}

	fileData := []byte("PDF content bytes")
	inner := buildMultipart("bnd",
		part{"Content-Type: text/plain", "see file"},
		part{
			"Content-Type: application/pdf\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"doc.pdf\"",
			encodeB64(fileData),
		},
	)
	raw := "Subject: Files\r\nContent-Type: multipart/mixed; boundary=\"bnd\"\r\n\r\n" + inner
	if err := s.Data(strings.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	c := recv.last()
	if c.parsed.Body != "see file" {
		t.Errorf("body: %q", c.parsed.Body)
	}
	if len(c.parsed.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(c.parsed.Attachments))
	}
	if c.parsed.Attachments[0].Filename != "doc.pdf" {
		t.Errorf("filename: %q", c.parsed.Attachments[0].Filename)
	}
	if !bytes.Equal(c.parsed.Attachments[0].Data, fileData) {
		t.Errorf("attachment data mismatch")
	}
}

func TestSession_Data_FromAndTo_PassedCorrectly(t *testing.T) {
	recv := &mockReceiver{}
	s := makeSession(recv)
	s.from = "sender@outer.com"
	s.to = []string{"one@e-smail.ru", "two@e-smail.ru"}

	raw := "Subject: s\r\nContent-Type: text/plain\r\n\r\nbody"
	if err := s.Data(strings.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	c := recv.last()
	if c.from != "sender@outer.com" {
		t.Errorf("from: %q", c.from)
	}
	if len(c.to) != 2 || c.to[0] != "one@e-smail.ru" || c.to[1] != "two@e-smail.ru" {
		t.Errorf("to: %v", c.to)
	}
}

func TestSession_Data_ReceiverError_Propagated(t *testing.T) {
	recv := &mockReceiver{err: io.ErrUnexpectedEOF}
	s := makeSession(recv)
	s.from = "a@b.com"
	s.to = []string{"u@e-smail.ru"}

	raw := "Subject: s\r\nContent-Type: text/plain\r\n\r\nbody"
	if err := s.Data(strings.NewReader(raw)); err != io.ErrUnexpectedEOF {
		t.Errorf("expected receiver error, got %v", err)
	}
}

func TestSession_Data_InvalidMIME_Fallback(t *testing.T) {
	recv := &mockReceiver{}
	s := makeSession(recv)
	s.from = "a@b.com"
	s.to = []string{"u@e-smail.ru"}

	raw := "Subject: s\r\nContent-Type: !!!bad\r\n\r\nfallback body"
	if err := s.Data(strings.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	if recv.last().parsed.Body != "fallback body" {
		t.Errorf("body: %q", recv.last().parsed.Body)
	}
}

type part struct {
	headers string
	body    string
}

func buildMultipart(boundary string, parts ...part) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString("--" + boundary + "\r\n")
		sb.WriteString(p.headers + "\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(p.body + "\r\n")
	}
	sb.WriteString("--" + boundary + "--\r\n")
	return sb.String()
}
