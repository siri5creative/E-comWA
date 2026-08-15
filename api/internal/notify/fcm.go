// Package notify sends push notifications to admin devices via Firebase
// Cloud Messaging (PRD section 6.6). Only Cloud Messaging is needed (per
// IMPLEMENTATION.md: "aktifkan Cloud Messaging saja"), so this talks to the
// FCM HTTP v1 REST API directly with golang.org/x/oauth2/google for
// service-account auth, instead of pulling in the full firebase-admin-go
// SDK (which also brings in Firestore/Auth/gRPC dependencies we don't
// need).
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

type FCMClient struct {
	httpClient *http.Client
	projectID  string
}

// NewFCMClient builds a client from a Firebase service account JSON
// credential. If serviceAccountJSON is empty, it returns (nil, nil) —
// push notifications are an enhancement, not a requirement to run the
// app, so callers should treat a nil client as "notifications disabled"
// rather than fail startup.
func NewFCMClient(ctx context.Context, serviceAccountJSON string) (*FCMClient, error) {
	if serviceAccountJSON == "" {
		return nil, nil
	}

	creds, err := google.CredentialsFromJSONWithType(ctx, []byte(serviceAccountJSON), google.ServiceAccount, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("parse firebase service account: %w", err)
	}
	if creds.ProjectID == "" {
		return nil, fmt.Errorf("firebase service account JSON is missing project_id")
	}

	return &FCMClient{
		httpClient: oauth2.NewClient(ctx, creds.TokenSource),
		projectID:  creds.ProjectID,
	}, nil
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

// Send delivers one push notification to a single device token.
func (c *FCMClient) Send(ctx context.Context, deviceToken, title, body string, data map[string]string) error {
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", c.projectID)

	payload, err := json.Marshal(fcmRequest{Message: fcmMessage{
		Token:        deviceToken,
		Notification: fcmNotification{Title: title, Body: body},
		Data:         data,
	}})
	if err != nil {
		return fmt.Errorf("marshal fcm payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build fcm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send fcm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm responded %s: %s", resp.Status, string(body))
	}
	return nil
}

// SendToAll delivers to multiple device tokens, best-effort — a failure on
// one token (e.g. it was revoked/uninstalled) is logged and does not stop
// delivery to the others.
func (c *FCMClient) SendToAll(ctx context.Context, deviceTokens []string, title, body string, data map[string]string) {
	for _, token := range deviceTokens {
		if err := c.Send(ctx, token, title, body, data); err != nil {
			slog.Warn("fcm send failed", "error", err)
		}
	}
}
