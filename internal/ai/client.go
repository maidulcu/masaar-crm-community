package ai

import (
	"context"
	"errors"
)

// ErrCommunityEdition is returned by all AI methods in the community build.
var ErrCommunityEdition = errors.New("AI features require a Pro plan — visit https://masaar.io/pricing")

// Client is a no-op stub for the community edition.
type Client struct{}

func NewClient(baseURL, model string) *Client        { return &Client{} }
func NewGeminiClient(apiKey, model string) *Client   { return &Client{} }

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) ScoreLead(ctx context.Context, contactName, summary, source string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) DraftReply(ctx context.Context, contactName, language, threadSummary string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) SummarizeThread(ctx context.Context, messages []string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) ParseIntent(ctx context.Context, message string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) EnrichLead(ctx context.Context, contactName, conversation string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) DescribePropertyListing(ctx context.Context, area, propertyType, bedrooms string, sizeSqft int, amenities []string, lang string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) ExtractBuyerProfile(ctx context.Context, messages []string) (string, error) {
	return "", ErrCommunityEdition
}

func (c *Client) SuggestAction(ctx context.Context, message, threadSummary string) (string, error) {
	return "", ErrCommunityEdition
}
