package smtp

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// EmailSender implements port.EmailSender via stdlib net/smtp.
type EmailSender struct {
	cfg pkgconfig.SMTPConfig
}

var _ port.EmailSender = (*EmailSender)(nil)

// NewEmailSender creates a new SMTP-backed EmailSender.
func NewEmailSender(cfg pkgconfig.SMTPConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

// Send delivers msg via the configured SMTP relay.
// MailHog accepts unauthenticated connections — no SMTP auth is used when
// cfg.Username is empty.
func (s *EmailSender) Send(_ context.Context, msg port.EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		s.cfg.From, msg.To, msg.Subject,
	)
	body := headers + msg.HTMLBody

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, []byte(body)); err != nil {
		return apperrors.NewInternal("smtp: failed to send email", err)
	}
	return nil
}
