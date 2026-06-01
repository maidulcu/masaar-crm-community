package billing

import "errors"

var errProOnly = errors.New("billing features require a Pro plan — visit https://masaar.io/pricing")

type StripeConfig struct {
	SecretKey       string
	WebhookSecret   string
	PriceIDStarter  string
	PriceIDPro      string
	PriceIDBusiness string
	AppURL          string
}

func (c *StripeConfig) IsEnabled() bool { return false }
func SetupStripe(cfg *StripeConfig)     {}

func CreateCheckoutSession(cfg *StripeConfig, plan Plan, companyID, customerID string) (string, error) {
	return "", errProOnly
}

func CreatePortalSession(cfg *StripeConfig, customerID string) (string, error) {
	return "", errProOnly
}

func GetOrCreateCustomer(companyName, email, existingID string) (string, error) {
	return "", errProOnly
}
