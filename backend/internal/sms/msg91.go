package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrSMSProviderNotConfigured indicates missing credentials or configuration.
	ErrSMSProviderNotConfigured = errors.New("SMS provider is not configured")
	// ErrSMSDeliveryTimeout indicates gateway network timeout.
	ErrSMSDeliveryTimeout = errors.New("SMS gateway delivery timed out")
	// ErrSMSAuthFailed indicates gateway rejected provider API credentials.
	ErrSMSAuthFailed = errors.New("SMS gateway authentication failed (invalid auth key)")
	// ErrSMSGatewayRejected indicates template, sender ID, or parameter rejection.
	ErrSMSGatewayRejected = errors.New("SMS gateway rejected request parameters or template")
	// ErrSMSDeliveryFailed indicates general gateway delivery failure.
	ErrSMSDeliveryFailed = errors.New("SMS delivery failed")
)

// MSG91Config holds credentials and runtime settings for MSG91 OTP API.
type MSG91Config struct {
	BaseURL    string
	AuthKey    string
	TemplateID string
	SenderID   string
	EntityID   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// MSG91Provider implements transactional OTP delivery via MSG91 OTP API v5.
type MSG91Provider struct {
	config MSG91Config
	client *http.Client
	logger *slog.Logger
}

// NewMSG91Provider initializes a new MSG91 SMS provider.
func NewMSG91Provider(cfg MSG91Config, logger *slog.Logger) *MSG91Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://control.msg91.com/api/v5/otp"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &MSG91Provider{
		config: cfg,
		client: httpClient,
		logger: logger,
	}
}

// SendOTP dispatches an OTP code via MSG91 OTP API.
func (p *MSG91Provider) SendOTP(ctx context.Context, phone string, otp string) error {
	if p == nil || strings.TrimSpace(p.config.AuthKey) == "" || strings.TrimSpace(p.config.TemplateID) == "" {
		return ErrSMSProviderNotConfigured
	}

	cleanPhone := strings.TrimSpace(phone)
	cleanPhone = strings.TrimPrefix(cleanPhone, "+")

	// Construct request URL with query parameters per MSG91 OTP v5 specification
	endpoint, err := url.Parse(p.config.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid MSG91 base URL: %w", err)
	}

	query := endpoint.Query()
	query.Set("template_id", p.config.TemplateID)
	query.Set("mobile", cleanPhone)
	query.Set("otp", otp)
	if p.config.SenderID != "" {
		query.Set("sender", p.config.SenderID)
	}
	endpoint.RawQuery = query.Encode()

	payloadBytes, _ := json.Marshal(map[string]string{
		"otp": otp,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create MSG91 HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authkey", p.config.AuthKey)

	// Safe masked phone for structured logging (Secret & PII redaction)
	maskedPhone := cleanPhone
	if len(cleanPhone) >= 10 {
		maskedPhone = cleanPhone[:4] + "****" + cleanPhone[len(cleanPhone)-2:]
	}

	p.logger.Info("Dispatching OTP via MSG91",
		slog.String("phone_masked", maskedPhone),
		slog.String("template_id", p.config.TemplateID),
	)

	resp, err := p.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			p.logger.Error("MSG91 request timed out",
				slog.String("phone_masked", maskedPhone),
				slog.Duration("timeout", p.config.Timeout),
			)
			return ErrSMSDeliveryTimeout
		}
		p.logger.Error("MSG91 network transport error",
			slog.String("phone_masked", maskedPhone),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%w: %v", ErrSMSDeliveryFailed, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var msg91Resp struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(bodyBytes, &msg91Resp)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		p.logger.Error("MSG91 authentication failure",
			slog.Int("http_status", resp.StatusCode),
			slog.String("message", msg91Resp.Message),
		)
		return ErrSMSAuthFailed
	}

	if resp.StatusCode == http.StatusBadRequest {
		p.logger.Error("MSG91 parameter or template rejection",
			slog.Int("http_status", resp.StatusCode),
			slog.String("message", msg91Resp.Message),
		)
		return fmt.Errorf("%w: %s", ErrSMSGatewayRejected, msg91Resp.Message)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logger.Error("MSG91 unexpected gateway error status",
			slog.Int("http_status", resp.StatusCode),
			slog.String("response", string(bodyBytes)),
		)
		return fmt.Errorf("%w: status %d", ErrSMSDeliveryFailed, resp.StatusCode)
	}

	if strings.ToLower(msg91Resp.Type) == "error" {
		p.logger.Error("MSG91 response returned error type",
			slog.String("message", msg91Resp.Message),
		)
		return fmt.Errorf("%w: %s", ErrSMSDeliveryFailed, msg91Resp.Message)
	}

	p.logger.Info("MSG91 OTP dispatched successfully",
		slog.String("phone_masked", maskedPhone),
		slog.String("gateway_message", msg91Resp.Message),
	)

	return nil
}
