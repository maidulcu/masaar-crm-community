package docusign

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	IntegrationKey string
	PrivateKeyPEM  string
	UserID         string // the API account user ID (sub claim)
	AccountID      string
	BaseURL        string // e.g. https://demo.docusign.net/restapi
	WebhookSecret  string
}

type Client struct {
	cfg       Config
	httpClient *http.Client
	mu        sync.Mutex
	acctBase  string // cached {base_uri}/restapi/v2.1/accounts/{account_id}
	token     string
	tokenExp  time.Time
}

func NewClient(cfg Config) *Client {
	base := strings.TrimRight(cfg.BaseURL, "/")
	acctBase := fmt.Sprintf("%s/v2.1/accounts/%s", base, cfg.AccountID)
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		acctBase:   acctBase,
	}
}

// Enabled returns true when all required credentials are present.
func (c *Client) Enabled() bool {
	return c.cfg.IntegrationKey != "" && c.cfg.PrivateKeyPEM != "" &&
		c.cfg.UserID != "" && c.cfg.AccountID != ""
}

// ── OAuth (JWT Grant) ──────────────────────────────────────────────────────

func (c *Client) tokenExpiry() time.Duration {
	d := time.Until(c.tokenExp)
	if d < 60*time.Second {
		return 0
	}
	return d
}

func (c *Client) ensureToken(ctx context.Context) error {
	if c.tokenExpiry() > 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenExpiry() > 0 {
		return nil
	}

	privKey, err := parsePrivateKey(c.cfg.PrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("docusign parse key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.cfg.IntegrationKey,
		"sub": c.cfg.UserID,
		"aud": "account-d.docusign.com",
		"iat": now.Unix(),
		"exp": now.Add(59 * time.Minute).Unix(),
		"scope": "signature impersonation",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	assertion, err := tok.SignedString(privKey)
	if err != nil {
		return fmt.Errorf("docusign sign jwt: %w", err)
	}

	body := strings.NewReader("grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=" + assertion)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://account-d.docusign.com/oauth/token", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("docusign oauth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docusign oauth status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("docusign oauth decode: %w", err)
	}

	c.token = result.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return nil
}

// ── Envelope ──────────────────────────────────────────────────────────────

type Signer struct {
	Name  string
	Email string
}

type EnvelopeResult struct {
	EnvelopeID string `json:"envelopeId"`
	Status     string `json:"status"`
	URI        string `json:"uri"`
}

// SendEnvelope creates and sends a DocuSign envelope for the given document.
// docContent is the raw PDF or document content as a base64 string.
// docName is the filename shown to the signer.
func (c *Client) SendEnvelope(ctx context.Context, docContent []byte, docName string, signer Signer) (*EnvelopeResult, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	b64Content := base64.StdEncoding.EncodeToString(docContent)
	body := map[string]interface{}{
		"emailSubject": fmt.Sprintf("Please sign: %s", docName),
		"documents": []map[string]interface{}{
			{
				"documentBase64": b64Content,
				"name":           docName,
				"fileExtension":  "pdf",
				"documentId":     "1",
			},
		},
		"recipients": map[string]interface{}{
			"signers": []map[string]interface{}{
				{
					"email":          signer.Email,
					"name":           signer.Name,
					"recipientId":    "1",
					"routingOrder":   "1",
					"tabs": map[string]interface{}{
						"signHereTabs": []map[string]interface{}{
							{
								"documentId":  "1",
								"pageNumber":  "1",
								"xPosition":   "200",
								"yPosition":   "700",
							},
						},
					},
				},
			},
		},
		"status": "sent",
	}

	payload, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/envelopes", c.acctBase)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docusign envelope: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docusign envelope status %d: %s", resp.StatusCode, string(b))
	}

	var result EnvelopeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("docusign envelope decode: %w", err)
	}
	return &result, nil
}

// GetEnvelopeStatus returns the current status of an envelope.
func (c *Client) GetEnvelopeStatus(ctx context.Context, envelopeID string) (*EnvelopeResult, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/envelopes/%s", c.acctBase, envelopeID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docusign status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docusign status %d: %s", resp.StatusCode, string(b))
	}

	var result EnvelopeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("docusign status decode: %w", err)
	}
	return &result, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────

func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		decoded, err := base64.StdEncoding.DecodeString(pemStr)
		if err == nil {
			block, _ = pem.Decode(decoded)
		}
	}
	if block == nil {
		return nil, errors.New("no PEM block found in private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: pkcs8: %w, pkcs1: %w", err, err2)
		}
		return key2, nil
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}
