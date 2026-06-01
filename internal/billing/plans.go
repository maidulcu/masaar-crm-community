package billing

// Plan identifiers
const (
	PlanCommunity = "community"
	PlanStarter   = "starter"
	PlanPro       = "pro"
	PlanBusiness  = "business"
)

// Quotas defines monthly limits for a plan. -1 means unlimited, 0 means blocked.
type Quotas struct {
	BOS24Monthly int // BuyOrSell24 API calls per month
	AIMonthly    int // AI (Gemini/Ollama) requests per month
	PDFMonthly   int // PDF report exports per month
	AIUserDaily  int // per-user daily AI cap (prevents one agent draining quota)
}

// Plan describes a subscription tier.
type Plan struct {
	ID            string
	Name          string
	PriceUSDMonth int    // display price in USD
	StripePriceID string // populated from config at runtime
	Quotas        Quotas
	Features      []string
}

// Plans is the canonical list of all tiers.
var Plans = []Plan{
	{
		ID:            PlanCommunity,
		Name:          "Community",
		PriceUSDMonth: 0,
		Quotas:        Quotas{BOS24Monthly: 0, AIMonthly: 0, PDFMonthly: 0, AIUserDaily: 0},
		Features: []string{
			"Unlimited leads & contacts",
			"WhatsApp inbox",
			"Pipeline kanban",
			"Invoices & deals",
			"Rental management",
			"Self-hosted — your data stays local",
		},
	},
	{
		ID:            PlanStarter,
		Name:          "Starter",
		PriceUSDMonth: 29,
		Quotas:        Quotas{BOS24Monthly: 200, AIMonthly: 100, PDFMonthly: 10, AIUserDaily: 20},
		Features: []string{
			"Everything in Community",
			"200 BOS24 market data calls/month",
			"100 AI requests/month",
			"10 property report PDFs/month",
			"Area price heatmap",
			"Comparable transactions",
		},
	},
	{
		ID:            PlanPro,
		Name:          "Pro",
		PriceUSDMonth: 79,
		Quotas:        Quotas{BOS24Monthly: 1000, AIMonthly: 500, PDFMonthly: -1, AIUserDaily: 100},
		Features: []string{
			"Everything in Starter",
			"1,000 BOS24 calls/month",
			"500 AI requests/month",
			"Unlimited PDF reports",
			"Rental yield analysis",
			"AI property description writer",
			"Pause POS integration",
		},
	},
	{
		ID:            PlanBusiness,
		Name:          "Business",
		PriceUSDMonth: 199,
		Quotas:        Quotas{BOS24Monthly: -1, AIMonthly: -1, PDFMonthly: -1, AIUserDaily: -1},
		Features: []string{
			"Everything in Pro",
			"Unlimited BOS24 calls",
			"Unlimited AI requests",
			"Push listings to BOS24 (1-click publish)",
			"AI Listing Health Score",
			"Pause POS inventory sync",
			"Priority support",
		},
	},
}

// Get returns a plan by ID, defaulting to Community if not found.
func Get(planID string) Plan {
	for _, p := range Plans {
		if p.ID == planID {
			return p
		}
	}
	return Plans[0] // community
}

// CheckQuota returns whether the given usage count is within plan limits.
// resource: "bos24", "ai", or "pdf"
func CheckQuota(plan Plan, resource string, currentCount int) (allowed bool, limit int) {
	switch resource {
	case "bos24":
		limit = plan.Quotas.BOS24Monthly
	case "ai":
		limit = plan.Quotas.AIMonthly
	case "pdf":
		limit = plan.Quotas.PDFMonthly
	default:
		return true, -1
	}
	if limit == 0 {
		return false, 0 // plan doesn't include this feature
	}
	if limit == -1 {
		return true, -1 // unlimited
	}
	return currentCount < limit, limit
}
