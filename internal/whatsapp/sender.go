package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SenderConfig struct {
	BaseURL       string // https://graph.facebook.com/v{version}
	PhoneNumberID string
	AccessToken   string

	// MediaHosts overrides the host suffixes media may be downloaded from (default: Meta's own
	// domains). AllowInsecureMedia permits plain http and internal addresses for the media
	// download; it exists for tests and must never be enabled in production.
	MediaHosts         []string
	AllowInsecureMedia bool
}

type Sender struct {
	config      *SenderConfig
	httpClient  *http.Client // talks to the Graph API (a fixed, trusted URL)
	mediaClient *http.Client // downloads media from URLs Meta returns (SSRF-guarded)
}

func NewSender(config *SenderConfig) *Sender {
	s := &Sender{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	s.mediaClient = newMediaClient(config)
	return s
}

func (s *Sender) IsConfigured() bool {
	return s != nil && s.config != nil && s.config.PhoneNumberID != "" && s.config.AccessToken != ""
}

// APIError is an error response from the WhatsApp Cloud API.
type APIError struct {
	HTTPStatus int
	Code       int    // Meta error code, e.g. 131047 "re-engagement message"
	Subcode    int    //
	Type       string //
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Meta API error %d (HTTP %d): %s", e.Code, e.HTTPStatus, e.Message)
}

// parseAPIError reads Meta's {"error":{...}} body; the body is consumed.
func parseAPIError(resp *http.Response) *APIError {
	var body struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
			Subcode int    `json:"error_subcode"`
		} `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body)
	msg := body.Error.Message
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return &APIError{HTTPStatus: resp.StatusCode, Code: body.Error.Code, Subcode: body.Error.Subcode, Type: body.Error.Type, Message: msg}
}

func asAPIError(err error, target **APIError) bool { return errors.As(err, target) }

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

	template := map[string]interface{}{
		"name":     templateName,
		"language": map[string]string{"code": languageCode},
	}
	// A template without variables must not carry a body component: Meta rejects one whose
	// "parameters" is null/empty.
	if len(params) > 0 {
		var p []map[string]string
		for _, param := range params {
			p = append(p, map[string]string{"type": "text", "text": param})
		}
		template["components"] = []map[string]interface{}{{"type": "body", "parameters": p}}
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template":          template,
	}

	return s.sendRequest(ctx, payload)
}

// SendMarkAsRead marks a message as read
func (s *Sender) SendMarkAsRead(ctx context.Context, waMessageID string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("WhatsApp sender not configured")
	}
	_, err := s.postJSON(ctx, map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        waMessageID,
	})
	return err
}

// sendRequest posts a message payload and returns the new message's WhatsApp id.
func (s *Sender) sendRequest(ctx context.Context, payload map[string]interface{}) (string, error) {
	result, err := s.postJSON(ctx, payload)
	if err != nil {
		return "", err
	}
	if messages, ok := result["messages"].([]interface{}); ok && len(messages) > 0 {
		if msg, ok := messages[0].(map[string]interface{}); ok {
			if id, ok := msg["id"].(string); ok {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("no message ID in response")
}

func (s *Sender) postJSON(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/messages", s.config.BaseURL, s.config.PhoneNumberID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}
