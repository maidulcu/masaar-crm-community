package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://restapi.smscountry.com"

type Client struct {
	authKey    string
	authToken  string
	senderID   string
	httpClient *http.Client
}

func NewClient(authKey, authToken, senderID string) *Client {
	return &Client{
		authKey:   authKey,
		authToken: authToken,
		senderID:  senderID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
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

	url := fmt.Sprintf("%s/v0.1/Accounts/%s/SMSes/", baseURL, c.authKey)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("smscountry: error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
