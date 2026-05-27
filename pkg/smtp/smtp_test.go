package smtp

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
)

func parseMessage(t *testing.T, raw []byte) *mail.Message {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parseMessage: %v", err)
	}
	return msg
}

func mustBuild(t *testing.T, m *Message) []byte {
	t.Helper()
	raw, err := m.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return raw
}

func mediaType(t *testing.T, ct string) (string, map[string]string) {
	t.Helper()
	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("ParseMediaType(%q): %v", ct, err)
	}
	return mt, params
}

type bufferedPart struct {
	Header textproto.MIMEHeader
	Body   []byte
}

func collectParts(t *testing.T, boundary string, body io.Reader) []bufferedPart {
	t.Helper()
	mr := multipart.NewReader(body, boundary)
	var parts []bufferedPart
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		data, err := io.ReadAll(p)
		if err != nil {
			t.Fatalf("read part body: %v", err)
		}
		parts = append(parts, bufferedPart{Header: p.Header, Body: data})
	}
	return parts
}

func readQP(t *testing.T, data []byte) string {
	t.Helper()
	b, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(data)))
	if err != nil {
		t.Fatalf("readQP: %v", err)
	}
	return string(b)
}

func readMsgBodyQP(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(quotedprintable.NewReader(r))
	if err != nil {
		t.Fatalf("readMsgBodyQP: %v", err)
	}
	return string(b)
}

func readBase64(t *testing.T, data []byte) []byte {
	t.Helper()
	clean := strings.ReplaceAll(string(data), "\r\n", "")
	decoded, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	return decoded
}

func TestFrom_BothNameSurname(t *testing.T) {
	m := NewMessage().From("Иван", "Петров", "ivan@e-smail.ru")

	if m.fromAddr != "ivan@e-smail.ru" {
		t.Errorf("fromAddr: got %q, want %q", m.fromAddr, "ivan@e-smail.ru")
	}
	if !strings.Contains(m.fromDisplay, "ivan@e-smail.ru") {
		t.Errorf("fromDisplay missing addr: %q", m.fromDisplay)
	}
	if !strings.Contains(m.fromDisplay, "Иван") && !strings.Contains(m.fromDisplay, "=?") {
		t.Errorf("fromDisplay missing name (raw or encoded): %q", m.fromDisplay)
	}
}

func TestFrom_OnlyName(t *testing.T) {
	m := NewMessage().From("Иван", "", "ivan@e-smail.ru")
	if m.fromAddr != "ivan@e-smail.ru" {
		t.Errorf("fromAddr: %q", m.fromAddr)
	}
	if m.fromDisplay == "" {
		t.Error("fromDisplay is empty")
	}
}

func TestFrom_EmptyNames_NoDisplayName(t *testing.T) {
	m := NewMessage().From("", "", "ivan@e-smail.ru")
	if m.fromAddr != "ivan@e-smail.ru" {
		t.Errorf("fromAddr: %q", m.fromAddr)
	}
	if strings.Contains(m.fromDisplay, `"`) {
		t.Errorf("no display name expected, got: %q", m.fromDisplay)
	}
}

func TestFrom_SpaceTrimmedWhenOnePart(t *testing.T) {
	m := NewMessage().From("Иван", "", "a@b.ru")
	parsed, err := mail.ParseAddress(m.fromDisplay)
	if err != nil {
		t.Fatalf("parse fromDisplay: %v", err)
	}
	dec := new(mime.WordDecoder)
	name, _ := dec.DecodeHeader(parsed.Name)
	if strings.HasSuffix(name, " ") || strings.HasPrefix(name, " ") {
		t.Errorf("display name has extra spaces: %q", name)
	}
}

func TestFrom_EnvelopeAddr_IsPureEmail(t *testing.T) {
	m := NewMessage().From("Вася", "Пупкин", "vasya@e-smail.ru")
	if strings.ContainsAny(m.fromAddr, `"<>`) {
		t.Errorf("fromAddr contains display-name syntax: %q", m.fromAddr)
	}
	if m.fromAddr != "vasya@e-smail.ru" {
		t.Errorf("fromAddr: got %q, want plain email", m.fromAddr)
	}
}

func TestFrom_HeaderUsesDisplay(t *testing.T) {
	m := NewMessage().From("Вася", "Пупкин", "vasya@e-smail.ru").To("b@b.ru").Subject("s").Text("t")
	msg := parseMessage(t, mustBuild(t, m))

	fromHeader := msg.Header.Get("From")
	if !strings.Contains(fromHeader, "vasya@e-smail.ru") {
		t.Errorf("From header missing addr: %q", fromHeader)
	}
	if !strings.Contains(fromHeader, "=?") && !strings.Contains(fromHeader, "Вася") {
		t.Errorf("From header missing name: %q", fromHeader)
	}
}

func TestFrom_CyrillicName_DecodesCorrectly(t *testing.T) {
	m := NewMessage().From("Вася", "Пупкин", "vasya@e-smail.ru").To("b@b.ru").Subject("s").Text("t")
	msg := parseMessage(t, mustBuild(t, m))

	parsed, err := mail.ParseAddress(msg.Header.Get("From"))
	if err != nil {
		t.Fatalf("parse From header: %v", err)
	}
	dec := new(mime.WordDecoder)
	name, err := dec.DecodeHeader(parsed.Name)
	if err != nil {
		t.Fatalf("decode name: %v", err)
	}
	if !strings.Contains(name, "Вася") || !strings.Contains(name, "Пупкин") {
		t.Errorf("decoded name: %q", name)
	}
}

func TestFrom_ASCIIName_NotEncoded(t *testing.T) {
	m := NewMessage().From("Ivan", "Petrov", "ivan@e-smail.ru").To("b@b.ru").Subject("s").Text("t")
	msg := parseMessage(t, mustBuild(t, m))

	fromHeader := msg.Header.Get("From")
	if strings.Contains(fromHeader, "=?") {
		t.Errorf("ASCII name should not be RFC2047-encoded: %q", fromHeader)
	}
	if !strings.Contains(fromHeader, "Ivan") || !strings.Contains(fromHeader, "Petrov") {
		t.Errorf("From header: %q", fromHeader)
	}
}

func TestNewMessage_ZeroValue(t *testing.T) {
	m := NewMessage()
	if m.fromAddr != "" || m.fromDisplay != "" {
		t.Error("expected empty fromAddr/fromDisplay")
	}
	if m.subject != "" || m.text != "" || m.html != "" {
		t.Error("expected empty subject/text/html")
	}
	if len(m.to) != 0 || len(m.attachments) != 0 {
		t.Error("expected nil to/attachments")
	}
}

func TestNewSmtpClient(t *testing.T) {
	c := NewSmtpClient("localhost", "587", "ignored", "ignored")
	if c.Host != "localhost" {
		t.Errorf("Host: %q", c.Host)
	}
	if c.Port != "587" {
		t.Errorf("Port: %q", c.Port)
	}
}

func TestTo_Accumulates(t *testing.T) {
	m := NewMessage().To("a@a.ru").To("b@b.ru")
	if len(m.to) != 2 {
		t.Errorf("want 2, got %d", len(m.to))
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
			m := NewMessage().Attach(tc.filename, []byte("x"), "")
			if got := m.attachments[0].MIMEType; got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAttach_ExplicitMimeNotOverridden(t *testing.T) {
	m := NewMessage().Attach("f.pdf", []byte("x"), "application/octet-stream")
	if m.attachments[0].MIMEType != "application/octet-stream" {
		t.Errorf("explicit mime type overridden: %q", m.attachments[0].MIMEType)
	}
}

func TestAttach_CaseInsensitiveExtension(t *testing.T) {
	m := NewMessage().Attach("f.PDF", []byte("x"), "")
	if m.attachments[0].MIMEType != "application/pdf" {
		t.Errorf(".PDF not recognized: %q", m.attachments[0].MIMEType)
	}
}

func TestBuild_PlainText_BodyPresent(t *testing.T) {
	m := NewMessage().
		From("Иван", "Петров", "ivan@e-smail.ru").
		To("recv@mail.ru").
		Subject("Hello").
		Text("Hello world")

	msg := parseMessage(t, mustBuild(t, m))
	body := readMsgBodyQP(t, msg.Body)
	if body != "Hello world" {
		t.Errorf("body: got %q, want %q", body, "Hello world")
	}
}

func TestBuild_PlainText_Headers(t *testing.T) {
	m := NewMessage().
		From("Ivan", "Petrov", "ivan@e-smail.ru").
		To("a@b.ru").Subject("Test").Text("body")

	msg := parseMessage(t, mustBuild(t, m))

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
}

func TestBuild_PlainText_MultipleRecipients(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru", "c@c.ru").Subject("s").Text("t")

	msg := parseMessage(t, mustBuild(t, m))
	to := msg.Header.Get("To")
	if !strings.Contains(to, "b@b.ru") || !strings.Contains(to, "c@c.ru") {
		t.Errorf("To header: %q", to)
	}
}

func TestBuild_PlainText_CyrillicBody(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("Привет, мир!")

	msg := parseMessage(t, mustBuild(t, m))
	body := readMsgBodyQP(t, msg.Body)
	if body != "Привет, мир!" {
		t.Errorf("body: %q", body)
	}
}

func TestBuild_EmptyTextBody(t *testing.T) {
	m := NewMessage().From("", "", "a@a.ru").To("b@b.ru").Subject("s")
	raw := mustBuild(t, m)
	if len(raw) == 0 {
		t.Fatal("expected non-empty raw")
	}
	parseMessage(t, raw)
}

func TestBuild_SubjectCyrillic_RoundTrip(t *testing.T) {
	m := NewMessage().From("", "", "a@a.ru").To("b@b.ru").Subject("Привет мир").Text("t")
	msg := parseMessage(t, mustBuild(t, m))

	dec := new(mime.WordDecoder)
	subj, err := dec.DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if subj != "Привет мир" {
		t.Errorf("subject: got %q, want %q", subj, "Привет мир")
	}
}

func TestBuild_SubjectASCII_NotEncoded(t *testing.T) {
	m := NewMessage().From("", "", "a@a.ru").To("b@b.ru").Subject("Hello World").Text("t")
	msg := parseMessage(t, mustBuild(t, m))

	if msg.Header.Get("Subject") != "Hello World" {
		t.Errorf("ASCII subject should not be encoded: %q", msg.Header.Get("Subject"))
	}
}

func TestBuild_HTMLOnly(t *testing.T) {
	m := NewMessage().From("", "", "a@a.ru").To("b@b.ru").Subject("s").Html("<p>hi</p>")
	msg := parseMessage(t, mustBuild(t, m))

	mt, params := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "multipart/alternative" {
		t.Fatalf("Content-Type: %q", mt)
	}
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	pmt, _ := mediaType(t, parts[0].Header.Get("Content-Type"))
	if pmt != "text/html" {
		t.Errorf("part Content-Type: %q", pmt)
	}
	body := readQP(t, parts[0].Body)
	if body != "<p>hi</p>" {
		t.Errorf("html body: %q", body)
	}
}

func TestBuild_TextAndHTML_PartOrder(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("plain text").Html("<b>bold</b>")

	msg := parseMessage(t, mustBuild(t, m))
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
		t.Errorf("part[0]: %q, want text/plain", pmt0)
	}
	pmt1, _ := mediaType(t, parts[1].Header.Get("Content-Type"))
	if pmt1 != "text/html" {
		t.Errorf("part[1]: %q, want text/html", pmt1)
	}
	if got := readQP(t, parts[0].Body); got != "plain text" {
		t.Errorf("text part: %q", got)
	}
	if got := readQP(t, parts[1].Body); got != "<b>bold</b>" {
		t.Errorf("html part: %q", got)
	}
}

func TestBuild_TextWithAttachment_Structure(t *testing.T) {
	fileData := []byte("file content here")
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("see attachment").
		Attach("doc.txt", fileData, "text/plain")

	msg := parseMessage(t, mustBuild(t, m))
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
	if got := readQP(t, parts[0].Body); got != "see attachment" {
		t.Errorf("text body: %q", got)
	}
	if parts[1].Header.Get("Content-Transfer-Encoding") != "base64" {
		t.Errorf("attachment CTE: %q", parts[1].Header.Get("Content-Transfer-Encoding"))
	}
	if got := string(readBase64(t, parts[1].Body)); got != "file content here" {
		t.Errorf("attachment data: %q", got)
	}
}

func TestBuild_TextWithAttachment_BodyActuallyPresent(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("non-empty body text").
		Attach("f.bin", []byte{0x01, 0x02}, "application/octet-stream")

	msg := parseMessage(t, mustBuild(t, m))
	_, params := mediaType(t, msg.Header.Get("Content-Type"))
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) < 1 {
		t.Fatal("no parts")
	}
	if got := readQP(t, parts[0].Body); got == "" {
		t.Error("text body is empty — regression: mw.Close() must be called before buf.Bytes()")
	}
}

func TestBuild_AttachmentDispositionAndFilename(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("t").
		Attach("report.pdf", []byte("x"), "application/pdf")

	msg := parseMessage(t, mustBuild(t, m))
	_, params := mediaType(t, msg.Header.Get("Content-Type"))
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) < 2 {
		t.Fatal("no attachment part")
	}
	cd := parts[1].Header.Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition missing 'attachment': %q", cd)
	}
	if !strings.Contains(cd, "report.pdf") {
		t.Errorf("filename missing in Content-Disposition: %q", cd)
	}
}

func TestBuild_AttachmentCyrillicFilename(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("t").
		Attach("документ.pdf", []byte("x"), "application/pdf")

	raw := mustBuild(t, m)
	if len(raw) == 0 {
		t.Fatal("empty output")
	}
	parseMessage(t, raw)
}

func TestBuild_MultipleAttachments(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("body").
		Attach("a.pdf", []byte("pdf data"), "application/pdf").
		Attach("b.png", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png")

	msg := parseMessage(t, mustBuild(t, m))
	_, params := mediaType(t, msg.Header.Get("Content-Type"))
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts (text + 2 attachments), got %d", len(parts))
	}
	if got := string(readBase64(t, parts[1].Body)); got != "pdf data" {
		t.Errorf("first attachment data: %q", got)
	}
	want2 := []byte{0x89, 0x50, 0x4E, 0x47}
	if got := readBase64(t, parts[2].Body); string(got) != string(want2) {
		t.Errorf("second attachment data mismatch: %v", got)
	}
}

func TestBuild_HTMLWithAttachment_NestedStructure(t *testing.T) {
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("plain").Html("<b>html</b>").
		Attach("img.png", []byte{0x89, 0x50}, "image/png")

	msg := parseMessage(t, mustBuild(t, m))
	mt, params := mediaType(t, msg.Header.Get("Content-Type"))
	if mt != "multipart/mixed" {
		t.Fatalf("outer Content-Type: %q", mt)
	}
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) != 2 {
		t.Fatalf("expected 2 outer parts, got %d", len(parts))
	}

	innerMT, innerParams := mediaType(t, parts[0].Header.Get("Content-Type"))
	if innerMT != "multipart/alternative" {
		t.Errorf("inner Content-Type: %q", innerMT)
	}
	innerParts := collectParts(t, innerParams["boundary"], bytes.NewReader(parts[0].Body))
	if len(innerParts) != 2 {
		t.Fatalf("expected 2 inner parts, got %d", len(innerParts))
	}
	imt0, _ := mediaType(t, innerParts[0].Header.Get("Content-Type"))
	if imt0 != "text/plain" {
		t.Errorf("inner part[0]: %q", imt0)
	}
	imt1, _ := mediaType(t, innerParts[1].Header.Get("Content-Type"))
	if imt1 != "text/html" {
		t.Errorf("inner part[1]: %q", imt1)
	}

	pmt1, _ := mediaType(t, parts[1].Header.Get("Content-Type"))
	if pmt1 != "image/png" {
		t.Errorf("attachment Content-Type: %q", pmt1)
	}
}

func TestBuild_LargeAttachment_Roundtrip(t *testing.T) {
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = byte(i % 256)
	}
	m := NewMessage().
		From("", "", "a@a.ru").To("b@b.ru").Subject("s").
		Text("large").
		Attach("big.bin", data, "application/octet-stream")

	msg := parseMessage(t, mustBuild(t, m))
	_, params := mediaType(t, msg.Header.Get("Content-Type"))
	parts := collectParts(t, params["boundary"], msg.Body)
	if len(parts) < 2 {
		t.Fatal("no attachment part")
	}
	decoded := readBase64(t, parts[1].Body)
	if len(decoded) != len(data) {
		t.Fatalf("decoded len %d, want %d", len(decoded), len(data))
	}
	for i := range data {
		if decoded[i] != data[i] {
			t.Fatalf("data mismatch at byte %d: got %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestSend_EmptyRecipients(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := c.Send(NewMessage().From("", "", "a@a.ru").Text("hi"))
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty, got %v", err)
	}
}

func TestSend_EmptyRecipients_ViaSendEmail(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := c.SendEmail("Ivan", "Petrov", "a@a.ru", nil, "subj", "body", nil)
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty, got %v", err)
	}
}

func TestSend_EmptyRecipients_ViaSendPlainText(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := c.SendPlainText(nil, "subj", "body")
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty, got %v", err)
	}
}

func TestMessage_SendShorthand(t *testing.T) {
	c := &SmtpClient{Host: "localhost", Port: "587"}
	err := NewMessage().Send(c)
	if err != ErrRecipientListIsEmpty {
		t.Errorf("expected ErrRecipientListIsEmpty via m.Send, got %v", err)
	}
}

func TestLineBreaker_BreaksAt76(t *testing.T) {
	var buf strings.Builder
	lb := &lineBreaker{w: &buf}
	if _, err := lb.Write([]byte(strings.Repeat("A", 200))); err != nil {
		t.Fatalf("write: %v", err)
	}
	lines := strings.Split(buf.String(), "\r\n")
	for i, l := range lines {
		if i < len(lines)-1 && len(l) != 76 {
			t.Errorf("line %d: length %d, want 76", i, len(l))
		}
		if i == len(lines)-1 && len(l) > 76 {
			t.Errorf("last line too long: %d", len(l))
		}
	}
}

func TestLineBreaker_SmallWrite_NoBreak(t *testing.T) {
	var buf strings.Builder
	lb := &lineBreaker{w: &buf}
	lb.Write([]byte("short"))
	if buf.String() != "short" {
		t.Errorf("small write: %q", buf.String())
	}
}

func TestLineBreaker_ExactBoundary_152bytes(t *testing.T) {
	var buf strings.Builder
	lb := &lineBreaker{w: &buf}
	lb.Write([]byte(strings.Repeat("X", 76)))
	lb.Write([]byte(strings.Repeat("Y", 76)))

	lines := strings.Split(buf.String(), "\r\n")
	if len(lines) < 2 {
		t.Fatalf("expected ≥2 lines, got %d", len(lines))
	}
	if len(lines[0]) != 76 {
		t.Errorf("first line length: %d, want 76", len(lines[0]))
	}
}

func TestEncodeRFC2047_ASCII_Unchanged(t *testing.T) {
	s := "Hello World 123"
	if got := encodeRFC2047(s); got != s {
		t.Errorf("ASCII should not be encoded: %q", got)
	}
}

func TestEncodeRFC2047_Cyrillic_RoundTrip(t *testing.T) {
	s := "Привет мир"
	encoded := encodeRFC2047(s)
	if encoded == s {
		t.Fatal("cyrillic string should be encoded")
	}
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != s {
		t.Errorf("roundtrip: got %q, want %q", decoded, s)
	}
}

func TestEncodeRFC2047_EmptyString(t *testing.T) {
	if got := encodeRFC2047(""); got != "" {
		t.Errorf("empty string: %q", got)
	}
}

func TestDetectMime_CaseInsensitive(t *testing.T) {
	for _, ext := range []string{".PDF", ".PNG", ".JPEG", ".MP4"} {
		t.Run(ext, func(t *testing.T) {
			got := detectMimeByExtension(ext)
			if got == "application/octet-stream" {
				t.Errorf("%q not recognized", ext)
			}
		})
	}
}

func TestDetectMime_Unknown(t *testing.T) {
	if got := detectMimeByExtension(".xyz42"); got != "application/octet-stream" {
		t.Errorf(".xyz42: %q", got)
	}
}

func TestDetectMime_NoExtension(t *testing.T) {
	if got := detectMimeByExtension(""); got != "application/octet-stream" {
		t.Errorf("empty ext: %q", got)
	}
}
