package sms

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Provider defines the interface for transactional OTP SMS delivery.
type Provider interface {
	SendOTP(ctx context.Context, phone string, otp string) error
}

// MockProvider is used in development and automated testing environments.
// It logs OTP delivery without calling external networks and retains the last sent code for testing.
type MockProvider struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	lastOTPs map[string]string
}

// NewMockProvider creates an instance of MockProvider.
func NewMockProvider(logger *slog.Logger) *MockProvider {
	return &MockProvider{
		logger:   logger,
		lastOTPs: make(map[string]string),
	}
}

// SendOTP records the OTP in memory and logs it to structured output.
func (m *MockProvider) SendOTP(ctx context.Context, phone string, otp string) error {
	m.mu.Lock()
	m.lastOTPs[phone] = otp
	m.mu.Unlock()

	m.logger.Info("Mock SMS OTP dispatched",
		slog.String("phone", phone),
		slog.String("otp", otp),
	)
	return nil
}

// GetLastOTP retrieves the most recently dispatched OTP for a phone number (used in testing).
func (m *MockProvider) GetLastOTP(phone string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	otp, exists := m.lastOTPs[phone]
	return otp, exists
}

// ProductionProviderConfig holds credentials for DLT-compliant Indian SMS gateways.
type ProductionProviderConfig struct {
	GatewayURL string
	AuthKey    string
	SenderID   string
	EntityID   string
	TemplateID string
}

// ProductionProvider is a placeholder struct ready for integration with MSG91 or Fast2SMS.
type ProductionProvider struct {
	config ProductionProviderConfig
	logger *slog.Logger
}

// NewProductionProvider creates a production SMS provider instance.
func NewProductionProvider(cfg ProductionProviderConfig, logger *slog.Logger) *ProductionProvider {
	return &ProductionProvider{
		config: cfg,
		logger: logger,
	}
}

// SendOTP dispatches an SMS via the configured production gateway.
func (p *ProductionProvider) SendOTP(ctx context.Context, phone string, otp string) error {
	// Provider selection is pluggable per ADR 008.
	// When production vendor credentials are supplied, HTTP dispatch occurs here.
	p.logger.Info("Production SMS gateway invoked", slog.String("phone", phone))
	return fmt.Errorf("production SMS gateway not configured yet")
}
