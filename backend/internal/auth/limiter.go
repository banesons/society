package auth

import (
	"errors"
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
)

const (
	MaxPhoneRequestsPerHour = 5
	MaxIPRequestsPerHour    = 10
	PhoneCooldownSeconds    = 60
)

type phoneRecord struct {
	lastSentAt   time.Time
	requestTimes []time.Time
}

type ipRecord struct {
	requestTimes []time.Time
}

// RateLimiter tracks OTP request frequency across phone numbers and source IPs.
type RateLimiter struct {
	mu     sync.Mutex
	phones map[string]*phoneRecord
	ips    map[string]*ipRecord
}

// NewRateLimiter creates an in-memory rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		phones: make(map[string]*phoneRecord),
		ips:    make(map[string]*ipRecord),
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
