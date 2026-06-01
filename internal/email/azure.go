package email

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type AzureService struct {
	endpoint    string
	key         []byte
	fromAddress string
	client      *http.Client
}

func NewAzureService(endpoint, key, fromAddress string) *AzureService {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		decoded = []byte(key)
	}
	return &AzureService{
		endpoint:    strings.TrimRight(endpoint, "/"),
		key:         decoded,
		fromAddress: fromAddress,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

type azureEmailContent struct {
	Subject   string `json:"subject"`
	PlainText string `json:"plainText,omitempty"`
	HTML      string `json:"html,omitempty"`
}

type azureEmailAddress struct {
	Address     string `json:"address"`
	DisplayName string `json:"displayName,omitempty"`
}

type azureRecipients struct {
	To []azureEmailAddress `json:"to"`
}

type azureEmailPayload struct {
	SenderAddress string            `json:"senderAddress"`
	Content       azureEmailContent `json:"content"`
	Recipients    azureRecipients   `json:"recipients"`
}

func (s *AzureService) IsConfigured() bool {
	return s.endpoint != "" && len(s.key) > 0 && s.fromAddress != ""
}

func (s *AzureService) Send(email *domain.EmailHistory) error {
	if !s.IsConfigured() {
		return fmt.Errorf("azure email service not configured")
	}

	payload := azureEmailPayload{
		SenderAddress: s.fromAddress,
		Content: azureEmailContent{
			Subject:   email.Subject,
			PlainText: email.Body,
			HTML:      email.HTMLBody,
		},
		Recipients: azureRecipients{
			To: []azureEmailAddress{{
				Address: email.ToEmail,
			}},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("azure marshal payload: %w", err)
	}

	reqURL := s.endpoint + "/emails:send?api-version=2023-03-31"
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("azure create request: %w", err)
	}

	// Azure requires GMT not UTC in the date string
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	contentHashB64 := sha256Base64(body)
	host := req.URL.Host
	// Use the actual parsed path+query so the signature matches exactly what's sent
	pathAndQuery := req.URL.RequestURI()

	sig := s.sign("POST", pathAndQuery, date, host, contentHashB64)
	auth := fmt.Sprintf("HMAC-SHA256 SignedHeaders=x-ms-date;host;x-ms-content-sha256&Signature=%s", sig)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ms-date", date)
	req.Header.Set("x-ms-content-sha256", contentHashB64)
	req.Header.Set("Authorization", auth)

	log.Printf("[azure-email] sending to=%s subject=%q endpoint=%s", email.ToEmail, email.Subject, s.endpoint)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("azure send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[azure-email] accepted status=%d", resp.StatusCode)
		return nil
	}

	return fmt.Errorf("azure email API error: %s - %s", resp.Status, string(respBody))
}

func (s *AzureService) sign(verb, pathAndQuery, date, host, contentHashBase64 string) string {
	stringToSign := fmt.Sprintf("%s\n%s\n%s;%s;%s", verb, pathAndQuery, date, host, contentHashBase64)
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func sha256Base64(data []byte) string {
	h := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(h[:])
}
