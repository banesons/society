package notification

import (
	"context"
	"fmt"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMPushProvider dispatches web/mobile push notifications through Firebase Cloud Messaging HTTP v1.
type FCMPushProvider struct {
	client *messaging.Client
}

// NewFCMPushProvider creates a new FCM push provider using credentials file or Application Default Credentials.
func NewFCMPushProvider(ctx context.Context, projectID, credentialsFile string) (*FCMPushProvider, error) {
	var opts []option.ClientOption
	cleanedPath := strings.TrimSpace(credentialsFile)
	if cleanedPath != "" {
		if _, err := os.Stat(cleanedPath); err == nil {
			opts = append(opts, option.WithCredentialsFile(cleanedPath))
		}
	}

	fbConfig := &firebase.Config{
		ProjectID: strings.TrimSpace(projectID),
	}

	app, err := firebase.NewApp(ctx, fbConfig, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fcm messaging client: %w", err)
	}

	return &FCMPushProvider{
		client: msgClient,
	}, nil
}

// Send dispatches a push notification to a target device registration token.
func (p *FCMPushProvider) Send(ctx context.Context, msg PushMessage) PushResult {
	if p == nil || p.client == nil {
		return PushResult{
			Err: ErrProviderNotConfigured,
		}
	}

	token := strings.TrimSpace(msg.Token)
	if token == "" {
		return PushResult{
			InvalidToken: true,
			Err:          ErrInvalidToken,
		}
	}

	fcmMsg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Data: msg.Data,
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: msg.Title,
				Body:  msg.Body,
				Icon:  "/favicon.svg",
			},
		},
	}

	msgID, err := p.client.Send(ctx, fcmMsg)
	if err != nil {
		isInvalid := messaging.IsUnregistered(err) ||
			messaging.IsInvalidArgument(err) ||
			strings.Contains(strings.ToLower(err.Error()), "registration-token-not-registered") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid-registration-token")

		return PushResult{
			InvalidToken: isInvalid,
			Err:          err,
		}
	}

	return PushResult{
		ProviderMessageID: msgID,
		InvalidToken:      false,
		Err:               nil,
	}
}
