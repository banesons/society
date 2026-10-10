package auth

import (
	"testing"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	phone := "+919967089101"

	t.Run("first request succeeds", func(t *testing.T) {
		resendAfter, err := rl.CheckAndRecordPhone(phone)
		if err != nil {
			t.Fatalf("unexpected error on first request: %v", err)
		}
		if resendAfter != PhoneCooldownSeconds {
			t.Errorf("expected %d, got %d", PhoneCooldownSeconds, resendAfter)
		}
	})

	t.Run("immediate second request triggers cooldown", func(t *testing.T) {
		_, err := rl.CheckAndRecordPhone(phone)
		if err != ErrOTPCooldown {
			t.Fatalf("expected ErrOTPCooldown, got %v", err)
		}
	})

	t.Run("IP rate limit triggers after 10 requests", func(t *testing.T) {
		ip := "192.168.1.100"
		for i := 0; i < 10; i++ {
			if err := rl.CheckAndRecordIP(ip); err != nil {
				t.Fatalf("unexpected error on request %d: %v", i+1, err)
			}
		}

		err := rl.CheckAndRecordIP(ip)
		if err != ErrIPRateLimited {
			t.Fatalf("expected ErrIPRateLimited after 10 requests, got %v", err)
		}
	})
}
