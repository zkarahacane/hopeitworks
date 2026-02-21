package port

import "context"

// EmailMessage represents a single outbound email.
type EmailMessage struct {
	To      string
	Subject string
	// HTMLBody is the HTML email body.
	HTMLBody string
}

// EmailSender delivers transactional emails.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
