package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMSG91Provider(t *testing.T) {
	testPhone := "919967089101"
	testOTP := "654321"

	t.Run("missing provider configuration returns ErrSMSProviderNotConfigured", func(t *testing.T) {
		p := NewMSG91Provider(MSG91Config{
			AuthKey:    "",
			TemplateID: "",
		}, nil)

		err := p.SendOTP(context.Background(), testPhone, testOTP)
		if !errors.Is(err, ErrSMSProviderNotConfigured) {
			t.Fatalf("expected ErrSMSProviderNotConfigured, got: %v", err)
		}
	})

	t.Run("successful OTP delivery dispatches request and logs masked phone without authkey", func(t *testing.T) {
		var capturedAuthKey string
		var capturedTemplateID string
		var capturedMobile string
		var capturedOTP string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedAuthKey = r.Header.Get("authkey")
			capturedTemplateID = r.URL.Query().Get("template_id")
			capturedMobile = r.URL.Query().Get("mobile")
			capturedOTP = r.URL.Query().Get("otp")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"type":    "success",
				"message": "OTP sent successfully",
			})
		}))
		defer server.Close()

		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, nil))

		p := NewMSG91Provider(MSG91Config{
			BaseURL:    server.URL,
			AuthKey:    "test-secret-auth-key-12345",
			TemplateID: "DLT_TEMPLATE_001",
			SenderID:   "CHSNEU",
			Timeout:    2 * time.Second,
		}, logger)

		err := p.SendOTP(context.Background(), testPhone, testOTP)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedAuthKey != "test-secret-auth-key-12345" {
			t.Errorf("expected authkey header to match, got: %s", capturedAuthKey)
		}
		if capturedTemplateID != "DLT_TEMPLATE_001" {
			t.Errorf("expected template_id DLT_TEMPLATE_001, got: %s", capturedTemplateID)
		}
		if capturedMobile != testPhone {
			t.Errorf("expected mobile %s, got: %s", testPhone, capturedMobile)
		}
		if capturedOTP != testOTP {
			t.Errorf("expected otp %s, got: %s", testOTP, capturedOTP)
		}

		// Verify Secret Redaction: Auth key must NEVER be printed in logs
		logOutput := logBuf.String()
		if strings.Contains(logOutput, "test-secret-auth-key-12345") {
			t.Errorf("SECURITY RISK: AuthKey leaked in log output: %s", logOutput)
		}
		// Verify phone is masked in logs
		if strings.Contains(logOutput, "919967089101") {
			t.Errorf("PII RISK: Unmasked phone found in log output: %s", logOutput)
		}
		if !strings.Contains(logOutput, "9199****01") {
			t.Errorf("expected masked phone 9199****01 in log output, got: %s", logOutput)
		}
	})

	t.Run("gateway 401 returns ErrSMSAuthFailed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"type":    "error",
				"message": "Invalid Auth Key",
			})
		}))
		defer server.Close()

		p := NewMSG91Provider(MSG91Config{
			BaseURL:    server.URL,
			AuthKey:    "invalid-key",
			TemplateID: "DLT_001",
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))

		err := p.SendOTP(context.Background(), testPhone, testOTP)
		if !errors.Is(err, ErrSMSAuthFailed) {
			t.Fatalf("expected ErrSMSAuthFailed, got: %v", err)
		}
	})

	t.Run("gateway 400 parameter rejection returns ErrSMSGatewayRejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"type":    "error",
				"message": "Template ID not registered on DLT",
			})
		}))
		defer server.Close()

		p := NewMSG91Provider(MSG91Config{
			BaseURL:    server.URL,
			AuthKey:    "valid-key",
			TemplateID: "UNREGISTERED_DLT",
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))

		err := p.SendOTP(context.Background(), testPhone, testOTP)
		if !errors.Is(err, ErrSMSGatewayRejected) {
			t.Fatalf("expected ErrSMSGatewayRejected, got: %v", err)
		}
	})

	t.Run("timeout returns ErrSMSDeliveryTimeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		p := NewMSG91Provider(MSG91Config{
			BaseURL:    server.URL,
			AuthKey:    "valid-key",
			TemplateID: "DLT_001",
			Timeout:    20 * time.Millisecond,
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err := p.SendOTP(ctx, testPhone, testOTP)
		if !errors.Is(err, ErrSMSDeliveryTimeout) {
			t.Fatalf("expected ErrSMSDeliveryTimeout, got: %v", err)
		}
	})

	t.Run("gateway 500 internal server error returns ErrSMSDeliveryFailed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		p := NewMSG91Provider(MSG91Config{
			BaseURL:    server.URL,
			AuthKey:    "valid-key",
			TemplateID: "DLT_001",
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))

		err := p.SendOTP(context.Background(), testPhone, testOTP)
		if !errors.Is(err, ErrSMSDeliveryFailed) {
			t.Fatalf("expected ErrSMSDeliveryFailed on 500, got: %v", err)
		}
	})

	t.Run("gateway returns type error in json body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"type":    "error",
				"message": "insufficient wallet balance",
			})
		}))
		defer server.Close()

		p := NewMSG91Provider(MSG91Config{
			BaseURL:    server.URL,
			AuthKey:    "valid-key",
			TemplateID: "DLT_001",
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))

		err := p.SendOTP(context.Background(), testPhone, testOTP)
		if !errors.Is(err, ErrSMSDeliveryFailed) {
			t.Fatalf("expected ErrSMSDeliveryFailed on json type error, got: %v", err)
		}
	})
}
