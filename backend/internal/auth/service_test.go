package auth

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"chs/backend/internal/sms"
)

func TestServiceSendOTP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	smsProvider := sms.NewMockProvider(logger)
	limiter := NewRateLimiter()

	cfg := ServiceConfig{
		DevTestPhones: []string{"919967089101", "918422862001"},
		DevTestOTP:    "123456",
	}

	svc := NewService(nil, tm, smsProvider, limiter, cfg, logger)
	ctx := context.Background()

	t.Run("sends dev test OTP for configured number", func(t *testing.T) {
		resendSec, otpCode, err := svc.SendOTP(ctx, "919967089101", "127.0.0.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resendSec != 60 {
			t.Errorf("expected 60s cooldown, got %d", resendSec)
		}
		if otpCode != "123456" {
			t.Errorf("expected otpCode 123456, got %s", otpCode)
		}

		otp, ok := smsProvider.GetLastOTP("919967089101")
		if !ok || otp != "123456" {
			t.Errorf("expected test OTP 123456 dispatched, got %s (exists=%v)", otp, ok)
		}
	})

	t.Run("immediate resend triggers cooldown error", func(t *testing.T) {
		_, _, err := svc.SendOTP(ctx, "919967089101", "127.0.0.1")
		if err == nil {
			t.Fatalf("expected cooldown error on immediate resend")
		}
	})

	t.Run("invalid phone rejected before SMS dispatch", func(t *testing.T) {
		_, _, err := svc.SendOTP(ctx, "invalid-phone", "127.0.0.2")
		if err == nil {
			t.Fatalf("expected error for invalid phone")
		}
	})

	t.Run("random 6-digit OTP generated for non-test numbers", func(t *testing.T) {
		nonTestPhone := "919820123456"
		resendSec, otpCode, err := svc.SendOTP(ctx, nonTestPhone, "127.0.0.3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resendSec != 60 {
			t.Errorf("expected 60s cooldown, got %d", resendSec)
		}
		if len(otpCode) != 6 {
			t.Errorf("expected 6 digit otpCode, got %s", otpCode)
		}

		otp, ok := smsProvider.GetLastOTP(nonTestPhone)
		if !ok {
			t.Fatalf("expected OTP dispatched for %s", nonTestPhone)
		}
		if len(otp) != 6 {
			t.Errorf("expected 6-digit OTP, got %s", otp)
		}
	})
}
