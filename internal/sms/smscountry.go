package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/maidulcu/masaar-crm/internal/safehttp"
)

const baseURL = "https://restapi.smscountry.com"

type Client struct {
	baseURL    string
	authKey    string
	authToken  string
	senderID   string
	httpClient *http.Client
}

func NewClient(authKey, authToken, senderID string) *Client {
	return &Client{
		baseURL:    baseURL,
		authKey:    authKey,
		authToken:  authToken,
		senderID:   senderID,
		// Enforce dial-time SSRF protection to block access to internal infrastructure/metadata.
		httpClient: safehttp.NewClient(10 * time.Second),
	}
}

func IsEnabled(authKey, authToken string) bool {
	return authKey != "" && authToken != ""
}

// SendOTP sends a 6-digit OTP to the given phone number.
func (c *Client) SendOTP(ctx context.Context, phone, otp string) error {
	body := map[string]string{
		"Text":     fmt.Sprintf("Your Masaar login code is: %s. Valid for 10 minutes.", otp),
		"Number":   phone,
		"SenderId": c.senderID,
		"Tool":     "API",
	}
	b, _ := json.Marshal(body)

	targetBase := c.baseURL
	if targetBase == "" {
		targetBase = baseURL
	}
	url := fmt.Sprintf("%s/v0.1/Accounts/%s/SMSes/", targetBase, c.authKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.authKey, c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("smscountry: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		// Limit response body read to 4KB to mitigate DoS / memory exhaustion from large error payloads.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("smscountry: error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
