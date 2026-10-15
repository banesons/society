package notification

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Worker processes pending outbox events and delivers notifications via configured providers.
type Worker struct {
	repo          *Repository
	emailProvider EmailProvider
	pushProvider  PushProvider
	logger        *slog.Logger
	workerID      string
	stopChan      chan struct{}
}

// NewWorker initializes a notification worker instance.
func NewWorker(repo *Repository, emailProv EmailProvider, pushProv PushProvider, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	workerID := fmt.Sprintf("worker-%s-%d", uuid.New().String()[:8], time.Now().UnixNano())
	return &Worker{
		repo:          repo,
		emailProvider: emailProv,
		pushProvider:  pushProv,
		logger:        logger,
		workerID:      workerID,
		stopChan:      make(chan struct{}),
	}
}

// Start begins processing pending outbox events in a loop.
func (w *Worker) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.logger.Info("notification worker started", "worker_id", w.workerID, "interval", interval.String())

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("notification worker stopping via context", "worker_id", w.workerID)
			return
		case <-w.stopChan:
			w.logger.Info("notification worker stopped via signal", "worker_id", w.workerID)
			return
		case <-ticker.C:
			w.ProcessBatch(ctx, 10)
		}
	}
}

// Stop signals the worker loop to terminate.
func (w *Worker) Stop() {
	close(w.stopChan)
}

// ProcessBatch claims and processes up to limit outbox events. Returns the count processed.
func (w *Worker) ProcessBatch(ctx context.Context, limit int) int {
	if w.repo == nil {
		return 0
	}

	events, err := w.repo.ClaimOutboxBatch(ctx, w.workerID, limit)
	if err != nil {
		w.logger.Error("failed to claim outbox batch", "error", err)
		return 0
	}

	if len(events) == 0 {
		return 0
	}

	for _, event := range events {
		if err := w.processEvent(ctx, event); err != nil {
			w.logger.Warn("event processing failed", "event_id", event.ID, "type", event.EventType, "error", err)
			backoff := calculateBackoff(event.AttemptCount)
			_ = w.repo.FailOutboxEvent(ctx, event.ID, err.Error(), backoff)
		} else {
			_ = w.repo.CompleteOutboxEvent(ctx, event.ID)
		}
	}

	return len(events)
}

func calculateBackoff(attempt int) time.Duration {
	base := 2.0
	multiplier := 1 << uint(attempt)
	secs := base * float64(multiplier)
	jitter := float64(rand.Intn(500)) / 1000.0 // 0 to 0.5s jitter
	return time.Duration((secs + jitter) * float64(time.Second))
}

func (w *Worker) processEvent(ctx context.Context, event OutboxEvent) error {
	p := event.Payload
	if p == nil {
		return nil
	}

	var recipientIDStr string
	if r, ok := p["recipient_user_id"].(string); ok {
		recipientIDStr = r
	} else if r, ok := p["resident_user_id"].(string); ok {
		recipientIDStr = r
	}
	if recipientIDStr == "" {
		return fmt.Errorf("missing recipient user ID in event payload")
	}

	recipientUserID, err := uuid.Parse(recipientIDStr)
	if err != nil {
		return fmt.Errorf("invalid recipient UUID: %w", err)
	}

	var societyID *uuid.UUID
	if socStr, ok := p["society_id"].(string); ok && socStr != "" {
		if parsed, err := uuid.Parse(socStr); err == nil {
			societyID = &parsed
		}
	}

	var title, body, actionURL, refType string
	var refID *uuid.UUID

	switch event.EventType {
	case EventVisitorApprovalRequested:
		gateName := getString(p, "gate_name", "the gate")
		visitorName := getString(p, "visitor_name", "A visitor")
		title = "Visitor Approval Requested"
		body = fmt.Sprintf("%s is at %s requesting access to your unit.", visitorName, gateName)
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#visitors", societyID.String())
		}
		refType = "society_visits"
		if vIDStr := getString(p, "visit_id", ""); vIDStr != "" {
			if parsed, err := uuid.Parse(vIDStr); err == nil {
				refID = &parsed
			}
		}

	case EventVisitApproved:
		unitNumber := getString(p, "unit_number", "")
		title = "Visit Approved"
		if unitNumber != "" {
			body = fmt.Sprintf("Your visit request for Unit %s has been approved.", unitNumber)
		} else {
			body = "Your visit request has been approved."
		}
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#visitors", societyID.String())
		}
		refType = "society_visits"
		if vIDStr := getString(p, "visit_id", ""); vIDStr != "" {
			if parsed, err := uuid.Parse(vIDStr); err == nil {
				refID = &parsed
			}
		}

	case EventVisitRejected:
		unitNumber := getString(p, "unit_number", "")
		title = "Visit Declined"
		if unitNumber != "" {
			body = fmt.Sprintf("The visit request for Unit %s was declined.", unitNumber)
		} else {
			body = "The visit request was declined by the resident."
		}
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#visitors", societyID.String())
		}
		refType = "society_visits"
		if vIDStr := getString(p, "visit_id", ""); vIDStr != "" {
			if parsed, err := uuid.Parse(vIDStr); err == nil {
				refID = &parsed
			}
		}

	case EventAnnouncementPublished:
		annTitle := getString(p, "title", "Announcement")
		title = "New Announcement: " + annTitle
		body = "A new society announcement has been published."
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#announcements", societyID.String())
		}
		refType = "society_announcements"
		if aIDStr := getString(p, "announcement_id", ""); aIDStr != "" {
			if parsed, err := uuid.Parse(aIDStr); err == nil {
				refID = &parsed
			}
		}

	case EventBillIssued:
		billNo := getString(p, "bill_number", "")
		period := getString(p, "period_name", "")
		dueDate := getString(p, "due_date", "")
		title = fmt.Sprintf("Maintenance Bill %s Issued", billNo)
		if dueDate != "" {
			body = fmt.Sprintf("Maintenance bill for %s has been generated. Due date: %s.", period, dueDate)
		} else {
			body = fmt.Sprintf("Maintenance bill for %s has been generated.", period)
		}
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#maintenance", societyID.String())
		}
		refType = "maintenance_bills"
		if bIDStr := getString(p, "bill_id", ""); bIDStr != "" {
			if parsed, err := uuid.Parse(bIDStr); err == nil {
				refID = &parsed
			}
		}

	case EventPaymentRecorded:
		receiptNo := getString(p, "receipt_number", "")
		amount := getString(p, "amount", "")
		title = fmt.Sprintf("Payment Received: Receipt #%s", receiptNo)
		if amount != "" {
			body = fmt.Sprintf("Payment of ₹%s received successfully.", amount)
		} else {
			body = "Your maintenance payment has been received and confirmed."
		}
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#maintenance", societyID.String())
		}
		refType = "maintenance_payments"
		if pIDStr := getString(p, "payment_id", ""); pIDStr != "" {
			if parsed, err := uuid.Parse(pIDStr); err == nil {
				refID = &parsed
			}
		}

	case EventPaymentReversed:
		receiptNo := getString(p, "receipt_number", "")
		reason := getString(p, "reason", "Administrative adjustment")
		title = fmt.Sprintf("Payment Reversal: Receipt #%s", receiptNo)
		body = fmt.Sprintf("Receipt #%s was reversed. Reason: %s", receiptNo, reason)
		if societyID != nil {
			actionURL = fmt.Sprintf("/society/%s#maintenance", societyID.String())
		}
		refType = "maintenance_payments"
		if pIDStr := getString(p, "payment_id", ""); pIDStr != "" {
			if parsed, err := uuid.Parse(pIDStr); err == nil {
				refID = &parsed
			}
		}

	default:
		title = "Society Notice"
		body = getString(p, "message", "You have a new society notification.")
	}

	// 1. Resolve registered devices for FCM push
	devices, err := w.repo.ListUserDevices(ctx, recipientUserID)
	if err != nil {
		w.logger.Warn("could not list user devices for notification", "user_id", recipientUserID, "error", err)
	}

	// 2. Prepare outbound deliveries
	var deliveries []NotificationDelivery
	for _, dev := range devices {
		if strings.TrimSpace(dev.FCMToken) != "" {
			deliveries = append(deliveries, NotificationDelivery{
				Channel:  ChannelFCM,
				Target:   dev.FCMToken,
				DeviceID: &dev.ID,
				Status:   DeliveryStatusPending,
			})
		}
	}

	// Add email delivery if email is present and event type warrants email
	recipientEmail := getString(p, "recipient_email", "")
	isEmailEvent := event.EventType == EventBillIssued ||
		event.EventType == EventPaymentRecorded ||
		event.EventType == EventPaymentReversed

	if isEmailEvent && recipientEmail != "" && strings.Contains(recipientEmail, "@") {
		deliveries = append(deliveries, NotificationDelivery{
			Channel: ChannelEmail,
			Target:  recipientEmail,
			Status:  DeliveryStatusPending,
		})
	}

	if p == nil {
		p = make(map[string]interface{})
	}
	p["outbox_event_id"] = event.ID.String()

	// 3. Persist notification and delivery records inside single short DB transaction
	nRecord := Notification{
		RecipientUserID: recipientUserID,
		SocietyID:       societyID,
		Type:            event.EventType,
		Title:           title,
		Body:            body,
		ActionURL:       actionURL,
		Metadata:        p,
		ReferenceType:   refType,
		ReferenceID:     refID,
	}

	createdN, createdDeliveries, err := w.repo.CreateNotificationAndDeliveries(ctx, nRecord, deliveries)
	if err != nil {
		return fmt.Errorf("failed to persist notification and deliveries: %w", err)
	}

	// 4. Perform external provider calls OUTSIDE any database transaction
	for _, delivery := range createdDeliveries {
		w.deliverChannel(ctx, createdN, delivery)
	}

	return nil
}

func (w *Worker) deliverChannel(ctx context.Context, n *Notification, d NotificationDelivery) {
	if d.Status == DeliveryStatusDelivered || d.Status == DeliveryStatusPermanentlyFailed || d.Status == DeliveryStatusSkipped {
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch d.Channel {
	case ChannelFCM:
		if w.pushProvider == nil {
			_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusSkipped, nil, strPtr("PROVIDER_NOT_CONFIGURED"), strPtr("FCM push provider not configured"))
			return
		}

		pushMsg := PushMessage{
			Token: d.Target,
			Title: n.Title,
			Body:  n.Body,
			Data: map[string]string{
				"notification_id": n.ID.String(),
				"type":            n.Type,
				"action_url":      n.ActionURL,
			},
		}

		res := w.pushProvider.Send(sendCtx, pushMsg)
		if res.Err != nil {
			if res.InvalidToken {
				_ = w.repo.DeactivateToken(ctx, d.Target)
				_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusPermanentlyFailed, nil, strPtr("INVALID_TOKEN"), strPtr(res.Err.Error()))
			} else {
				_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusFailed, nil, strPtr("PUSH_ERROR"), strPtr(res.Err.Error()))
			}
			return
		}

		_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusDelivered, &res.ProviderMessageID, nil, nil)

	case ChannelEmail:
		if w.emailProvider == nil {
			_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusSkipped, nil, strPtr("PROVIDER_NOT_CONFIGURED"), strPtr("Resend email provider not configured"))
			return
		}

		htmlContent := fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e5e5; border-radius: 8px;">
				<h2 style="color: #111;">%s</h2>
				<p style="color: #444; line-height: 1.5;">%s</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
				<p style="font-size: 12px; color: #888;">This is an automated administrative notification from Cooperative Housing Society (CHS).</p>
			</div>
		`, html.EscapeString(n.Title), html.EscapeString(n.Body))

		emailMsg := EmailMessage{
			To:          d.Target,
			Subject:     n.Title,
			HTMLBody:    htmlContent,
			TextBody:    n.Body,
			ReferenceID: n.ID.String(),
		}

		msgID, err := w.emailProvider.Send(sendCtx, emailMsg)
		if err != nil {
			_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusFailed, nil, strPtr("EMAIL_ERROR"), strPtr(err.Error()))
			return
		}

		_ = w.repo.UpdateDeliveryOutcome(ctx, d.ID, DeliveryStatusDelivered, &msgID, nil, nil)
	}
}

func getString(m map[string]interface{}, key, fallback string) string {
	if val, ok := m[key]; ok && val != nil {
		if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return fallback
}

func strPtr(s string) *string {
	return &s
}
