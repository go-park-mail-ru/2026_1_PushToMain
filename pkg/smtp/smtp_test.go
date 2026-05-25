package smtp

import (
	"encoding/base64"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

func parseMessage(t *testing.T, raw []byte) *mail.Message {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	return msg
}

func mediaType(t *testing.T, ct string) (string, map[string]string) {
	t.Helper()
	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("parse media type %q: %v", ct, err)
	}
	return mt, params
}

func readQP(t *testing.T, r interface{ Read([]byte) (int, error) }) string {
	t.Helper()
	var b strings.Builder
	buf := make([]byte, 4096)
	qp := quotedprintable.NewReader(r.(interface{ Read([]byte) (int, error) }))
	for {
		n, err := qp.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return b.String()
}

func collectParts(t *testing.T, boundary string, body interface{ Read([]byte) (int, error) }) []*multipart.Part {
	t.Helper()
	mr := multipart.NewReader(body.(interface {
		Read([]byte) (int, error)
	}), boundary)
	var parts []*multipart.Part
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		parts = append(parts, p)
	}
	return parts
}

func TestNewSmtpClient(t *testing.T) {
	c := NewSmtpClient("localhost", "587", "user", "pass")
	if c.Host != "localhost" {
		t.Errorf("host: got %q want %q", c.Host, "localhost")
	}
	if c.Port != "587" {
		t.Errorf("port: got %q want %q", c.Port, "587")
	}
}

func TestNewMessage_Defaults(t *testing.T) {
	m := NewMessage()
	if m.from != "" || m.subject != "" || m.text != "" || m.html != "" {
		t.Error("expected zero-value Message")
	}
	if len(m.to) != 0 || len(m.attachments) != 0 {
		t.Error("expected empty slices")
	}
}

func TestMessage_Builder(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru", "c@c.ru").
		Subject("Hi").
		Text("plain").
		Html("<b>bold</b>").
		Attach("f.txt", []byte("data"), "text/plain")

	if m.from != "a@a.ru" {
		t.Errorf("from: %q", m.from)
	}
	if len(m.to) != 2 {
		t.Errorf("to len: %d", len(m.to))
	}
	if m.subject != "Hi" {
		t.Errorf("subject: %q", m.subject)
	}
	if m.text != "plain" {
		t.Errorf("text: %q", m.text)
	}
	if m.html != "<b>bold</b>" {
		t.Errorf("html: %q", m.html)
	}
	if len(m.attachments) != 1 {
		t.Fatalf("attachments len: %d", len(m.attachments))
	}
	if m.attachments[0].Filename != "f.txt" {
		t.Errorf("attachment filename: %q", m.attachments[0].Filename)
	}
}

func TestMessage_ToAccumulates(t *testing.T) {
	m := NewMessage().To("a@a.ru").To("b@b.ru")
	if len(m.to) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(m.to))
	}
}

func TestSend_EmptyRecipients(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := c.Send(NewMessage().From("a@a.ru").Text("hi"))
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty, got %v", err)
	}
}

func TestSend_EmptyRecipientsViaSendPlainText(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := c.SendPlainText("a@a.ru", nil, "subj", "body")
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty, got %v", err)
	}
}

func TestBuild_PlainText(t *testing.T) {
	m := NewMessage().
		From("sender@e-smail.ru").
		To("recv@mail.ru").
		Subject("Hello").
		Text("Hello world")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	msg := parseMessage(t, raw)

	if msg.Header.Get("From") != "sender@e-smail.ru" {
		t.Errorf("From: %q", msg.Header.Get("From"))
	}
	if msg.Header.Get("To") != "recv@mail.ru" {
		t.Errorf("To: %q", msg.Header.Get("To"))
	}
	if msg.Header.Get("Subject") != "Hello" {
		t.Errorf("Subject: %q", msg.Header.Get("Subject"))
	}
	if msg.Header.Get("MIME-Version") != "1.0" {
		t.Errorf("MIME-Version: %q", msg.Header.Get("MIME-Version"))
	}

	mt, _ := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "text/plain" {
		t.Errorf("Content-Type: %q", mt)
	}
	if msg.Header.Get("Content-Transfer-Encoding") != "quoted-printable" {
		t.Errorf("CTE: %q", msg.Header.Get("Content-Transfer-Encoding"))
	}

	body := readQP(t, msg.Body)
	if body != "Hello world" {
		t.Errorf("body: %q", body)
	}
}

func TestBuild_PlainTextMultipleRecipients(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru", "c@c.ru").
		Subject("s").
		Text("t")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	to := msg.Header.Get("To")
	if !strings.Contains(to, "b@b.ru") || !strings.Contains(to, "c@c.ru") {
		t.Errorf("To header: %q", to)
	}
}

func TestBuild_SubjectCyrillic(t *testing.T) {
	m := NewMessage().From("a@a.ru").To("b@b.ru").Subject("Привет мир").Text("t")
	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	dec := new(mime.WordDecoder)
	subj, err := dec.DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if subj != "Привет мир" {
		t.Errorf("decoded subject: %q", subj)
	}
}

func TestBuild_SubjectASCII_NotEncoded(t *testing.T) {
	m := NewMessage().From("a@a.ru").To("b@b.ru").Subject("Hello").Text("t")
	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	if msg.Header.Get("Subject") != "Hello" {
		t.Errorf("ASCII subject should not be encoded, got: %q", msg.Header.Get("Subject"))
	}
}

func TestBuild_HTMLOnly(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Html("<p>test</p>")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	mt, params := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "multipart/alternative" {
		t.Fatalf("Content-Type: %q", mt)
	}

	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 1 {
		t.Fatalf("expected 1 part (html only), got %d", len(parts))
	}
	pmt, _ := mediaType(t, parts[0].Header.Get("Content-Type"))
	if pmt != "text/html" {
		t.Errorf("part Content-Type: %q", pmt)
	}
}

func TestBuild_TextAndHTML(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Text("plain").
		Html("<b>bold</b>")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	mt, params := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "multipart/alternative" {
		t.Fatalf("Content-Type: %q", mt)
	}

	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}

	pmt0, _ := mediaType(t, parts[0].Header.Get("Content-Type"))
	if pmt0 != "text/plain" {
		t.Errorf("part[0] Content-Type: %q", pmt0)
	}
	pmt1, _ := mediaType(t, parts[1].Header.Get("Content-Type"))
	if pmt1 != "text/html" {
		t.Errorf("part[1] Content-Type: %q", pmt1)
	}

	body0 := readQP(t, parts[0])
	if body0 != "plain" {
		t.Errorf("text part body: %q", body0)
	}
	body1 := readQP(t, parts[1])
	if body1 != "<b>bold</b>" {
		t.Errorf("html part body: %q", body1)
	}
}

func TestBuild_TextWithAttachment(t *testing.T) {
	fileData := []byte("file content")
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Text("see attachment").
		Attach("doc.txt", fileData, "text/plain")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	mt, params := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "multipart/mixed" {
		t.Fatalf("Content-Type: %q", mt)
	}

	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}

	pmt0, _ := mediaType(t, parts[0].Header.Get("Content-Type"))
	if pmt0 != "text/plain" {
		t.Errorf("part[0]: %q", pmt0)
	}

	if parts[1].Header.Get("Content-Transfer-Encoding") != "base64" {
		t.Errorf("attachment CTE: %q", parts[1].Header.Get("Content-Transfer-Encoding"))
	}
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := parts[1].Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(b.String(), "\r\n", ""))
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if string(decoded) != "file content" {
		t.Errorf("attachment data: %q", string(decoded))
	}
}

func TestBuild_HTMLWithAttachment(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Text("plain").
		Html("<b>html</b>").
		Attach("img.png", []byte{0x89, 0x50}, "image/png")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	mt, params := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "multipart/mixed" {
		t.Fatalf("Content-Type: %q", mt)
	}

	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (alternative block + attachment), got %d", len(parts))
	}

	pmt0, innerParams := mediaType(t, parts[0].Header.Get("Content-Type"))
	if pmt0 != "multipart/alternative" {
		t.Errorf("part[0] should be alternative, got %q", pmt0)
	}
	innerParts := collectParts(t, innerParams["boundary"], parts[0])
	if len(innerParts) != 2 {
		t.Fatalf("expected 2 inner parts, got %d", len(innerParts))
	}

	pmt1, _ := mediaType(t, parts[1].Header.Get("Content-Type"))
	if pmt1 != "image/png" {
		t.Errorf("attachment Content-Type: %q", pmt1)
	}
}

func TestBuild_MultipleAttachments(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Text("body").
		Attach("a.pdf", []byte("pdf"), "application/pdf").
		Attach("b.png", []byte("png"), "image/png")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	_, params := mediaType(t, msg.Header.Get("Content-Type"))
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts (text + 2 attachments), got %d", len(parts))
	}
}

func TestBuild_AttachmentDispositionFilename(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Text("t").
		Attach("report.pdf", []byte("x"), "application/pdf")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	msg := parseMessage(t, raw)
	_, params := mediaType(t, msg.Header.Get("Content-Type"))
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) < 2 {
		t.Fatal("no attachment part")
	}
	cd := parts[1].Header.Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition: %q", cd)
	}
	if !strings.Contains(cd, "report.pdf") {
		t.Errorf("filename missing in Content-Disposition: %q", cd)
	}
}

func TestBuild_AttachmentCyrillicFilename(t *testing.T) {
	m := NewMessage().
		From("a@a.ru").
		To("b@b.ru").
		Subject("s").
		Text("t").
		Attach("документ.pdf", []byte("x"), "application/pdf")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("empty output")
	}
}

func TestAttach_MimeTypeAutoDetect(t *testing.T) {
	cases := []struct {
		filename string
		want     string
	}{
		{"f.pdf", "application/pdf"},
		{"f.png", "image/png"},
		{"f.jpg", "image/jpeg"},
		{"f.jpeg", "image/jpeg"},
		{"f.gif", "image/gif"},
		{"f.webp", "image/webp"},
		{"f.svg", "image/svg+xml"},
		{"f.txt", "text/plain"},
		{"f.csv", "text/csv"},
		{"f.html", "text/html"},
		{"f.htm", "text/html"},
		{"f.json", "application/json"},
		{"f.xml", "application/xml"},
		{"f.zip", "application/zip"},
		{"f.tar", "application/x-tar"},
		{"f.gz", "application/gzip"},
		{"f.doc", "application/msword"},
		{"f.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"f.xls", "application/vnd.ms-excel"},
		{"f.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"f.ppt", "application/vnd.ms-powerpoint"},
		{"f.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"f.mp4", "video/mp4"},
		{"f.mov", "video/quicktime"},
		{"f.mp3", "audio/mpeg"},
		{"f.ogg", "audio/ogg"},
		{"f.unknown", "application/octet-stream"},
		{"f", "application/octet-stream"},
	}
	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			m := NewMessage().From("a@a.ru").To("b@b.ru").Subject("s").Text("t").
				Attach(tc.filename, []byte("x"), "")
			if got := m.attachments[0].MIMEType; got != tc.want {
				t.Errorf("Attach(%q): got %q want %q", tc.filename, got, tc.want)
			}
		})
	}
}

func TestAttach_ExplicitMimeTypeNotOverridden(t *testing.T) {
	m := NewMessage().Attach("f.pdf", []byte("x"), "application/octet-stream")
	if m.attachments[0].MIMEType != "application/octet-stream" {
		t.Errorf("explicit mime type overridden: %q", m.attachments[0].MIMEType)
	}
}

func TestEncodeRFC2047_ASCII(t *testing.T) {
	s := "Hello World"
	if got := encodeRFC2047(s); got != s {
		t.Errorf("ASCII string should not be encoded: %q", got)
	}
}

func TestEncodeRFC2047_Cyrillic(t *testing.T) {
	s := "Привет"
	enc := encodeRFC2047(s)
	if enc == s {
		t.Error("cyrillic string should be encoded")
	}
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != s {
		t.Errorf("roundtrip: got %q want %q", decoded, s)
	}
}

func TestDetectMimeByExtension_CaseInsensitive(t *testing.T) {
	if got := detectMimeByExtension(".PDF"); got != "application/pdf" {
		t.Errorf(".PDF: %q", got)
	}
	if got := detectMimeByExtension(".PNG"); got != "image/png" {
		t.Errorf(".PNG: %q", got)
	}
}

func TestDetectMimeByExtension_Unknown(t *testing.T) {
	if got := detectMimeByExtension(".xyz"); got != "application/octet-stream" {
		t.Errorf(".xyz: %q", got)
	}
}

func TestDetectMimeByExtension_Empty(t *testing.T) {
	if got := detectMimeByExtension(""); got != "application/octet-stream" {
		t.Errorf("empty ext: %q", got)
	}
}

func TestLineBreaker_BreaksAt76(t *testing.T) {
	var buf strings.Builder
	lb := &lineBreaker{w: &buf}
	data := strings.Repeat("A", 200)
	lb.Write([]byte(data))

	lines := strings.Split(buf.String(), "\r\n")
	for i, l := range lines {
		if i < len(lines)-1 && len(l) != 76 {
			t.Errorf("line %d length %d, want 76", i, len(l))
		}
		if i == len(lines)-1 && len(l) > 76 {
			t.Errorf("last line too long: %d", len(l))
		}
	}
}

func TestLineBreaker_SmallWrite(t *testing.T) {
	var buf strings.Builder
	lb := &lineBreaker{w: &buf}
	lb.Write([]byte("short"))
	if buf.String() != "short" {
		t.Errorf("small write: %q", buf.String())
	}
}

func TestBuild_EmptyBody(t *testing.T) {
	m := NewMessage().From("a@a.ru").To("b@b.ru").Subject("s")
	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestBuild_LargeAttachment(t *testing.T) {
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = byte(i % 256)
	}
	m := NewMessage().From("a@a.ru").To("b@b.ru").Subject("s").Text("t").
		Attach("big.bin", data, "application/octet-stream")

	raw, err := m.build()
	if err != nil {
		t.Fatalf("build 1MB attachment: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("empty output")
	}
}

func TestMessage_SendShorthand(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := NewMessage().Send(c)
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty via m.Send, got %v", err)
	}
}
