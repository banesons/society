package auth

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	// ErrOTPCooldown is returned when an OTP is requested before the 60-second cooldown expires.
	ErrOTPCooldown = errors.New("please wait before requesting another OTP")

	// ErrPhoneRateLimited is returned when a phone exceeds 5 requests per hour.
	ErrPhoneRateLimited = errors.New("maximum OTP requests exceeded for this phone, please try again later")

	// ErrIPRateLimited is returned when an IP exceeds 10 requests per hour.
	ErrIPRateLimited = errors.New("too many requests from this network, please try again later")

	// ErrVerifyIPRateLimited is returned when an IP exceeds failed verification attempts.
	ErrVerifyIPRateLimited = errors.New("too many failed verification attempts from this network, please try again later")
)

const (
	MaxPhoneRequestsPerHour = 5
	MaxIPRequestsPerHour    = 10
	PhoneCooldownSeconds    = 60
	MaxIPVerifyFailures     = 20
	VerifyFailureWindow     = 10 * time.Minute
)

type phoneRecord struct {
	lastSentAt   time.Time
	requestTimes []time.Time
}

type ipRecord struct {
	requestTimes []time.Time
}

type verifyRecord struct {
	failureTimes []time.Time
}

// RateLimiter tracks OTP request frequency across phone numbers and source IPs.
type RateLimiter struct {
	mu             sync.Mutex
	phones         map[string]*phoneRecord
	ips            map[string]*ipRecord
	verifyFailures map[string]*verifyRecord
}

// NewRateLimiter creates an in-memory rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		phones:         make(map[string]*phoneRecord),
		ips:            make(map[string]*ipRecord),
		verifyFailures: make(map[string]*verifyRecord),
	}
}

// CheckAndRecordPhone evaluates whether an OTP can be requested for the given phone.
// Returns resendAfterSeconds (remaining cooldown if in cooldown, or 60 upon successful dispatch).
func (rl *RateLimiter) CheckAndRecordPhone(phone string) (int, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, exists := rl.phones[phone]

	if !exists {
		rec = &phoneRecord{
			requestTimes: []time.Time{},
		}
		rl.phones[phone] = rec
	}

	// 1. Check cooldown (60 seconds)
	if !rec.lastSentAt.IsZero() {
		elapsed := now.Sub(rec.lastSentAt)
		if elapsed < PhoneCooldownSeconds*time.Second {
			remaining := int((PhoneCooldownSeconds*time.Second - elapsed).Seconds()) + 1
			return remaining, ErrOTPCooldown
		}
	}

	// 2. Prune timestamps older than 1 hour
	oneHourAgo := now.Add(-1 * time.Hour)
	validTimes := rec.requestTimes[:0]
	for _, t := range rec.requestTimes {
		if t.After(oneHourAgo) {
			validTimes = append(validTimes, t)
		}
	}
	rec.requestTimes = validTimes

	// 3. Check 5 requests per hour limit
	if len(rec.requestTimes) >= MaxPhoneRequestsPerHour {
		return PhoneCooldownSeconds, ErrPhoneRateLimited
	}

	// Record new dispatch
	rec.lastSentAt = now
	rec.requestTimes = append(rec.requestTimes, now)

	return PhoneCooldownSeconds, nil
}

// CheckAndRecordIP evaluates whether the source IP has exceeded 10 requests per hour.
func (rl *RateLimiter) CheckAndRecordIP(ip string) error {
	if ip == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, exists := rl.ips[ip]

	if !exists {
		rec = &ipRecord{
			requestTimes: []time.Time{},
		}
		rl.ips[ip] = rec
	}

	// Prune timestamps older than 1 hour
	oneHourAgo := now.Add(-1 * time.Hour)
	validTimes := rec.requestTimes[:0]
	for _, t := range rec.requestTimes {
		if t.After(oneHourAgo) {
			validTimes = append(validTimes, t)
		}
	}
	rec.requestTimes = validTimes

	if len(rec.requestTimes) >= MaxIPRequestsPerHour {
		return ErrIPRateLimited
	}

	rec.requestTimes = append(rec.requestTimes, now)
	return nil
}

// CheckVerifyIP checks whether the client IP has exceeded the allowed failed verification attempts.
func (rl *RateLimiter) CheckVerifyIP(ip string) error {
	if ip == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rec, exists := rl.verifyFailures[ip]
	if !exists {
		return nil
	}

	now := time.Now()
	cutoff := now.Add(-VerifyFailureWindow)
	validTimes := rec.failureTimes[:0]
	for _, t := range rec.failureTimes {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}
	rec.failureTimes = validTimes

	if len(rec.failureTimes) >= MaxIPVerifyFailures {
		return ErrVerifyIPRateLimited
	}
	return nil
}

// RecordVerifyFailureIP records a failed verification attempt for the given client IP.
func (rl *RateLimiter) RecordVerifyFailureIP(ip string) {
	if ip == "" {
		return
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rec, exists := rl.verifyFailures[ip]
	if !exists {
		rec = &verifyRecord{
			failureTimes: []time.Time{},
		}
		rl.verifyFailures[ip] = rec
	}

	now := time.Now()
	cutoff := now.Add(-VerifyFailureWindow)
	validTimes := rec.failureTimes[:0]
	for _, t := range rec.failureTimes {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}
	rec.failureTimes = append(validTimes, now)
}

// ResetVerifyIP resets failure records for a client IP upon successful verification.
func (rl *RateLimiter) ResetVerifyIP(ip string) {
	if ip == "" {
		return
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.verifyFailures, ip)
}
