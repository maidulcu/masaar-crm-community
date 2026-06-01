package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SenderConfig struct {
	BaseURL       string // https://graph.facebook.com/v{version}
	PhoneNumberID string
	AccessToken   string
}

type Sender struct {
	config     *SenderConfig
	httpClient *http.Client
}

func NewSender(config *SenderConfig) *Sender {
	return &Sender{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Sender) IsConfigured() bool {
	return s.config != nil && s.config.PhoneNumberID != "" && s.config.AccessToken != ""
}

// SendMessage sends a text message via WhatsApp API
func (s *Sender) SendMessage(ctx context.Context, to, body string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("WhatsApp sender not configured")
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": body,
		},
	}

	return s.sendRequest(ctx, payload)
}

// SendTemplate sends a message using a WhatsApp template
func (s *Sender) SendTemplate(ctx context.Context, to, templateName, languageCode string, params []string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("WhatsApp sender not configured")
	}

	components := []map[string]interface{}{
		{
			"type": "body",
			"parameters": func() []map[string]string {
				var p []map[string]string
				for _, param := range params {
					p = append(p, map[string]string{"type": "text", "text": param})
				}
				return p
			}(),
		},
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name": templateName,
			"language": map[string]string{
				"code": languageCode,
			},
			"components": components,
		},
	}

	return s.sendRequest(ctx, payload)
}

// SendMedia sends a message with media (image, document, audio)
func (s *Sender) SendMedia(ctx context.Context, to, mediaType, mediaURL string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("WhatsApp sender not configured")
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              mediaType, // image, document, audio, video
		mediaType: map[string]string{
			"link": mediaURL,
		},
	}

	return s.sendRequest(ctx, payload)
}

// SendMarkAsRead marks a message as read
func (s *Sender) SendMarkAsRead(ctx context.Context, waMessageID string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("WhatsApp sender not configured")
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        waMessageID,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s/messages", s.config.BaseURL, s.config.PhoneNumberID),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.AccessToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Meta API error: %v", errResp)
	}

	return nil
}

// sendRequest sends a request to Meta's WhatsApp API
func (s *Sender) sendRequest(ctx context.Context, payload map[string]interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s/messages", s.config.BaseURL, s.config.PhoneNumberID),
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.AccessToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("Meta API error: %v", errResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract message ID from response
	if messages, ok := result["messages"].([]interface{}); ok && len(messages) > 0 {
		if msg, ok := messages[0].(map[string]interface{}); ok {
			if id, ok := msg["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("no message ID in response")
}
