package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds SMTP configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
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

// Send sends an email
func (s *Sender) Send(to []string, subject, htmlBody string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("SMTP not configured")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	headers := []string{
		fmt.Sprintf("From: %s", s.config.From),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}

	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody)

	return smtp.SendMail(addr, auth, s.config.From, to, msg)
}
