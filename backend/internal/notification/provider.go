package notification

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrProviderNotConfigured = errors.New("notification provider not configured")
	ErrInvalidRecipient      = errors.New("invalid notification recipient")
	ErrInvalidToken          = errors.New("unregistered or invalid device push token")
)

// EmailMessage defines the contract for an outbound transactional email.
type EmailMessage struct {
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	ReferenceID string
}

// EmailProvider sends transactional emails.
type EmailProvider interface {
	Send(ctx context.Context, msg EmailMessage) (providerMessageID string, err error)
}

// PushMessage defines the contract for an outbound push message.
type PushMessage struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

// PushResult encapsulates the outcome of an external push send.
type PushResult struct {
	ProviderMessageID string
	InvalidToken      bool
	Err               error
}

// PushProvider sends push notifications to registered client devices.
type PushProvider interface {
	Send(ctx context.Context, msg PushMessage) PushResult
}

// MockEmailProvider is an in-memory test double for EmailProvider.
type MockEmailProvider struct {
	mu           sync.Mutex
	SentMessages []EmailMessage
	SendFunc     func(ctx context.Context, msg EmailMessage) (string, error)
}

func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		SentMessages: make([]EmailMessage, 0),
	}
}

func (m *MockEmailProvider) Send(ctx context.Context, msg EmailMessage) (string, error) {
	m.mu.Lock()
	m.SentMessages = append(m.SentMessages, msg)
	sendFunc := m.SendFunc
	m.mu.Unlock()

	if sendFunc != nil {
		return sendFunc(ctx, msg)
	}
	return "mock_resend_msg_" + msg.ReferenceID, nil
}

// MockPushProvider is an in-memory test double for PushProvider.
type MockPushProvider struct {
	mu           sync.Mutex
	SentMessages []PushMessage
	SendFunc     func(ctx context.Context, msg PushMessage) PushResult
}

func NewMockPushProvider() *MockPushProvider {
	return &MockPushProvider{
		SentMessages: make([]PushMessage, 0),
	}
}

func (m *MockPushProvider) Send(ctx context.Context, msg PushMessage) PushResult {
	m.mu.Lock()
	m.SentMessages = append(m.SentMessages, msg)
	sendFunc := m.SendFunc
	m.mu.Unlock()

	if sendFunc != nil {
		return sendFunc(ctx, msg)
	}

	if msg.Token == "invalid_fcm_token" {
		return PushResult{
			InvalidToken: true,
			Err:          ErrInvalidToken,
		}
	}

	return PushResult{
		ProviderMessageID: "mock_fcm_msg_" + msg.Token,
		InvalidToken:      false,
		Err:               nil,
	}
}
