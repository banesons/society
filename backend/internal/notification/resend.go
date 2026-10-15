package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/resend/resend-go/v4"
)

// ResendEmailProvider delivers transactional emails via the Resend API.
type ResendEmailProvider struct {
	client    *resend.Client
	fromEmail string
}

// NewResendEmailProvider initializes a new Resend client with the provided API key and sender email.
func NewResendEmailProvider(apiKey, fromEmail string) *ResendEmailProvider {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	sender := strings.TrimSpace(fromEmail)
	if sender == "" {
		sender = "onboarding@resend.dev"
	}
	return &ResendEmailProvider{
		client:    resend.NewClient(apiKey),
		fromEmail: sender,
	}
}

// Send dispatches an outbound email through Resend.
func (p *ResendEmailProvider) Send(ctx context.Context, msg EmailMessage) (string, error) {
	if p == nil || p.client == nil {
		return "", ErrProviderNotConfigured
	}

	to := strings.TrimSpace(msg.To)
	if to == "" || !strings.Contains(to, "@") {
		return "", ErrInvalidRecipient
	}

	params := &resend.SendEmailRequest{
		From:    p.fromEmail,
		To:      []string{to},
		Subject: msg.Subject,
		Html:    msg.HTMLBody,
		Text:    msg.TextBody,
	}

	sent, err := p.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return "", fmt.Errorf("resend send failed: %w", err)
	}

	return sent.Id, nil
}
