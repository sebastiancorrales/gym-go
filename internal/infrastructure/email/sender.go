package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// Config holds SMTP configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Attachment represents a file to be attached to an email.
type Attachment struct {
	Filename    string // e.g. "cierre_2024-01-15.xlsx"
	ContentType string // e.g. "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	Data        []byte
}

// Sender sends emails via SMTP
type Sender struct {
	config Config
}

// NewSender creates a new email sender
func NewSender(config Config) *Sender {
	return &Sender{config: config}
}

// IsConfigured returns true if SMTP is configured
func (s *Sender) IsConfigured() bool {
	return s.config.Host != "" && s.config.Username != "" && s.config.Password != ""
}

// Send sends an HTML email without attachments. Delegates to SendWithAttachments.
func (s *Sender) Send(to []string, subject, htmlBody string) error {
	return s.SendWithAttachments(to, subject, htmlBody, nil)
}

// SendWithAttachments sends an HTML email with optional file attachments using
// MIME multipart/mixed encoding. When attachments is nil or empty it falls back
// to a simple text/html message (no multipart overhead).
func (s *Sender) SendWithAttachments(to []string, subject, htmlBody string, attachments []Attachment) error {
	if !s.IsConfigured() {
		return fmt.Errorf("SMTP not configured")
	}

	msg, err := s.buildMessage(to, subject, htmlBody, attachments)
	if err != nil {
		return fmt.Errorf("building email message: %w", err)
	}

	return s.send(to, msg)
}

// Timeouts for the SMTP conversation. smtp.SendMail has none at all: a server
// that accepts the TCP connection and then stops answering used to block the
// caller forever, and the daily-close job sends inside a loop that also writes to
// the database between messages.
const (
	smtpDialTimeout   = 10 * time.Second
	smtpTotalDeadline = 90 * time.Second
)

// send performs the SMTP exchange with explicit deadlines. It mirrors what
// smtp.SendMail does (STARTTLS when offered, then AUTH) but over a connection we
// control, so a hung server fails instead of hanging.
func (s *Sender) send(to []string, msg []byte) error {
	// JoinHostPort, not Sprintf: an IPv6 host needs the brackets.
	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))

	conn, err := net.DialTimeout("tcp", addr, smtpDialTimeout)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	// Covers the whole conversation, attachments included.
	if err := conn.SetDeadline(time.Now().Add(smtpTotalDeadline)); err != nil {
		conn.Close()
		return fmt.Errorf("smtp set deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.config.Host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp rcpt to %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}

	return client.Quit()
}

func (s *Sender) buildMessage(to []string, subject, htmlBody string, attachments []Attachment) ([]byte, error) {
	var buf bytes.Buffer

	if len(attachments) == 0 {
		// Simple single-part HTML email
		headers := []string{
			fmt.Sprintf("From: %s", s.config.From),
			fmt.Sprintf("To: %s", strings.Join(to, ", ")),
			fmt.Sprintf("Subject: %s", subject),
			"MIME-Version: 1.0",
			"Content-Type: text/html; charset=UTF-8",
		}
		buf.WriteString(strings.Join(headers, "\r\n"))
		buf.WriteString("\r\n\r\n")
		buf.WriteString(htmlBody)
		return buf.Bytes(), nil
	}

	// Multipart/mixed for attachments
	mw := multipart.NewWriter(&buf)

	headers := []string{
		fmt.Sprintf("From: %s", s.config.From),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		fmt.Sprintf(`Content-Type: multipart/mixed; boundary="%s"`, mw.Boundary()),
	}
	buf.WriteString(strings.Join(headers, "\r\n"))
	buf.WriteString("\r\n\r\n")

	// HTML body part (quoted-printable so special chars survive transit)
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := mw.CreatePart(htmlHeader)
	if err != nil {
		return nil, fmt.Errorf("creating HTML part: %w", err)
	}
	qp := quotedprintable.NewWriter(htmlPart)
	if _, err := qp.Write([]byte(htmlBody)); err != nil {
		return nil, fmt.Errorf("writing HTML body: %w", err)
	}
	qp.Close()

	// Attachment parts (base64)
	for _, att := range attachments {
		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", att.ContentType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Filename))
		attPart, err := mw.CreatePart(attHeader)
		if err != nil {
			return nil, fmt.Errorf("creating attachment part %q: %w", att.Filename, err)
		}
		encoded := base64.StdEncoding.EncodeToString(att.Data)
		// RFC 2045 §6.8: lines must be <= 76 chars
		for len(encoded) > 76 {
			if _, err := fmt.Fprintf(attPart, "%s\r\n", encoded[:76]); err != nil {
				return nil, err
			}
			encoded = encoded[76:]
		}
		if len(encoded) > 0 {
			fmt.Fprintf(attPart, "%s\r\n", encoded)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("closing MIME writer: %w", err)
	}

	return buf.Bytes(), nil
}
