package auth

import (
	"testing"
)

func TestGenerateNumericOTP(t *testing.T) {
	for i := 0; i < 20; i++ {
		otp, err := GenerateNumericOTP()
		if err != nil {
			t.Fatalf("unexpected error generating OTP: %v", err)
		}
		if len(otp) != 6 {
			t.Errorf("expected 6 digits, got %d (%s)", len(otp), otp)
		}
		for _, ch := range otp {
			if ch < '0' || ch > '9' {
				t.Errorf("non-numeric character in OTP: %c", ch)
			}
		}
	}
}

func TestVerifyOTPHash(t *testing.T) {
	phone := "+919967089101"
	otp := "123456"

	hash := HashOTP(phone, otp)

	if !VerifyOTPHash(phone, otp, hash) {
		t.Errorf("expected VerifyOTPHash to return true for matching OTP")
	}

	if VerifyOTPHash(phone, "654321", hash) {
		t.Errorf("expected VerifyOTPHash to return false for incorrect OTP")
	}

	if VerifyOTPHash("+918422862001", otp, hash) {
		t.Errorf("expected VerifyOTPHash to return false for different phone salt")
	}
}

func TestOpaqueTokens(t *testing.T) {
	token1, err := GenerateRandomOpaqueToken(32)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	token2, err := GenerateRandomOpaqueToken(32)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token1 == token2 {
		t.Errorf("collision in random opaque tokens: %s", token1)
	}

	hash1 := HashToken(token1)
	hash2 := HashToken(token1)
	if hash1 != hash2 {
		t.Errorf("HashToken is not deterministic")
	}
}
