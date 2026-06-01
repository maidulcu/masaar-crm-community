package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	basicauth "github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberws "github.com/gofiber/websocket/v2"
	_ "github.com/maidulcu/masaar-crm/docs" // swagger generated docs
	"github.com/maidulcu/masaar-crm/internal/api/handler"
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	fiberswagger "github.com/swaggo/fiber-swagger"
)

type Handlers struct {
	Auth              *handler.AuthHandler
	User              *handler.UserHandler
	Stats             *handler.StatsHandler
	Contact           *handler.ContactHandler
	Lead              *handler.LeadHandler
	WhatsApp          *handler.WhatsAppHandler
	WhatsAppOutbound  *handler.WhatsAppOutboundHandler
	AI                *handler.AIHandler
	Message           *handler.MessageHandler
	Notification      *handler.NotificationHandler
	Deal              *handler.DealHandler
	Invoice           *handler.InvoiceHandler
	Property          *handler.PropertyHandler
	Settings          *handler.SettingsHandler
	Email             *handler.EmailHandler
	RentalProperty    *handler.RentalPropertyHandler
	Tenant            *handler.TenantHandler
	LeaseTemplate     *handler.LeaseTemplateHandler
	Lease             *handler.LeaseHandler
	Payment           *handler.PaymentHandler
	BankIntegration   *handler.BankIntegrationHandler
	BankStatement     *handler.BankStatementHandler
	PaymentConfirmation *handler.PaymentConfirmationHandler
	Analytics         *handler.AnalyticsHandler
	Expense           *handler.ExpenseHandler
	Inspection        *handler.InspectionHandler
	Maintenance       *handler.MaintenanceTaskHandler
	LeaseRenewal      *handler.LeaseRenewalHandler
	Document          *handler.DocumentHandler
	ApiKey            *handler.ApiKeyHandler
	PublicLead        *handler.PublicLeadHandler
	WebhookSub        *handler.WebhookSubHandler
	Billing           *handler.BillingHandler
	MessageTemplate   *handler.MessageTemplateHandler
	AuditLog          *handler.AuditHandler
	Listing           *handler.ListingHandler
	BOS24Integration  *handler.BOS24IntegrationHandler
	Offer             *handler.OfferHandler
	LeadRotation      *handler.LeadRotationHandler
	Commission        *handler.CommissionHandler
	Performance       *handler.PerformanceHandler
	Viewing           *handler.ViewingHandler
	Marketing         *handler.MarketingHandler
	ImportExport      *handler.ImportExportHandler
	PipelineStage     *handler.PipelineStageHandler
	Approval          *handler.ApprovalHandler
	DocusignWebhook   *handler.DocusignWebhookHandler
}

// webhookLimiter allows Meta's burst delivery (300 req/min per IP) while
// blocking abuse. Meta retries on 429 so legitimate messages are never lost.
var webhookLimiter = limiter.New(limiter.Config{
	Max:        300,
	Expiration: 1 * time.Minute,
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
	},
})

// loginLimiter prevents brute-force on the auth endpoint.
var loginLimiter = limiter.New(limiter.Config{
	Max:        10,
	Expiration: 1 * time.Minute,
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many login attempts"})
	},
})

// magicLinkLimiter caps magic link requests to prevent abuse (3 requests/minute per IP).
var magicLinkLimiter = limiter.New(limiter.Config{
	Max:        3,
	Expiration: 1 * time.Minute,
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many requests, please try again later"})
	},
})

// smsOTPLimiter caps SMS OTP requests to 3 per minute per IP.
var smsOTPLimiter = limiter.New(limiter.Config{
	Max:        3,
	Expiration: 1 * time.Minute,
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many requests, please try again later"})
	},
})

// registrationLimiter caps new company signups to 3 per minute per IP.
// Tighter than loginLimiter because registration creates DB rows and sends emails.
var registrationLimiter = limiter.New(limiter.Config{
	Max:        3,
	Expiration: 1 * time.Minute,
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many registration attempts, please try again later"})
	},
})

// apiLimiter caps general authenticated API usage to 100 requests/min per IP.
var apiLimiter = limiter.New(limiter.Config{
	Max:        100,
	Expiration: 1 * time.Minute,
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
	},
})

// apiKeyLimiter caps each API key to 300 requests/min using Redis sliding window.
func makeAPIKeyLimiter(rdb *redis.Client) fiber.Handler {
	return middleware.APIKeyRateLimit(rdb, 300, time.Minute)
}

func RegisterRoutes(app *fiber.App, h *Handlers, hub *ws.Hub, cfg *config.Config, rdb *redis.Client, pool *pgxpool.Pool, apiKeyRepo *repo.ApiKeyRepo, billingRepo *repo.BillingRepo, companyRepo *repo.CompanyRepo) {
	// ── Public routes ────────────────────────────────────────────────────────
	app.Post("/api/v1/auth/login", loginLimiter, h.Auth.Login)
	app.Post("/api/v1/auth/register", registrationLimiter, h.Auth.Register)
	app.Post("/api/v1/auth/magic-link/request", magicLinkLimiter, h.Auth.RequestMagicLink)
	app.Post("/api/v1/auth/magic-link/verify", h.Auth.VerifyMagicLink)
	app.Post("/api/v1/auth/sms/request", smsOTPLimiter, h.Auth.RequestSMSOTP)
	app.Post("/api/v1/auth/sms/verify", h.Auth.VerifySMSOTP)
	app.Post("/api/v1/auth/refresh", h.Auth.Refresh)
	app.Post("/api/v1/auth/forgot-password", loginLimiter, h.Auth.ForgotPassword)
	app.Post("/api/v1/auth/reset-password", h.Auth.ResetPassword)

	// WhatsApp webhook — Meta calls this publicly
	app.Get("/webhooks/whatsapp", h.WhatsApp.Verify)
	app.Post("/webhooks/whatsapp", webhookLimiter, h.WhatsApp.Receive)

	// Stripe webhook — must be public (raw body, no JWT)
	app.Post("/webhooks/stripe", h.Billing.StripeWebhook)

	// DocuSign Connect webhook — receives signature completion events
	if h.DocusignWebhook != nil {
		app.Post("/webhooks/docusign", h.DocusignWebhook.Handle)
	}

	// BOS24 inbound webhook — company identified by ?token=<secret>, HMAC-verified
	app.Post("/webhooks/bos24", webhookLimiter, h.BOS24Integration.ReceiveWebhook)

	// Public listing page — no auth required (for shareable links)
	app.Get("/api/public/listings/:id", h.Marketing.PublicListing)

	// Public document signing — signature UUID is the access token
	app.Get("/api/public/sign/:id", h.Document.PublicGetSignature)
	app.Post("/api/public/sign/:id", h.Document.PublicSign)

	// Public lead intake — API key auth (scope: lead:create)
	apiKeyLimiter := makeAPIKeyLimiter(rdb)
	app.Post("/webhooks/leads",
		webhookLimiter,
		middleware.ValidateAPIKey(apiKeyRepo),
		apiKeyLimiter,
		middleware.RequireAPIKeyScope("lead:create"),
		h.PublicLead.SubmitLead,
	)

	// ── WebSocket — authenticated upgrade ────────────────────────────────────
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Personal notifications
	trialCheck := middleware.TrialCheck(companyRepo)
	app.Get("/ws/notifications",
		middleware.JWT(cfg.JWTSecret),
		middleware.CheckBlacklist(rdb),
		trialCheck,
		middleware.ExtractClaims(),
		fiberws.New(hub.Handler()),
	)

	// ── Authenticated API ────────────────────────────────────────────────────
	v1 := app.Group("/api/v1",
		apiLimiter,
		middleware.JWT(cfg.JWTSecret),
		middleware.CheckBlacklist(rdb),
		trialCheck,
		middleware.ExtractClaims(),
		middleware.DemoGuard(), // blocks writes on demo accounts; reads is_demo from JWT
	)

	v1.Delete("/auth/logout", h.Auth.Logout)

	// Dashboard stats — all authenticated users
	v1.Get("/stats", h.Stats.Overview)

	// User management — list/create: admin only; personal settings: any auth user
	v1.Get("/users", middleware.RequireRole(domain.RoleAdmin), h.User.ListUsers)
	v1.Post("/users", middleware.RequireRole(domain.RoleAdmin), h.User.CreateUser)
	v1.Post("/users/invite", middleware.RequireRole(domain.RoleAdmin), h.User.InviteUser)
	v1.Get("/users/me", h.User.GetMe)
	v1.Patch("/users/me/password", h.User.ChangePassword)
	v1.Patch("/users/me/lang", h.User.UpdateLang)
	v1.Patch("/users/:id", middleware.RequireRole(domain.RoleAdmin), h.User.UpdateUser)
	v1.Patch("/users/:id/active", middleware.RequireRole(domain.RoleAdmin), h.User.SetActive)
	v1.Delete("/users/:id", middleware.RequireRole(domain.RoleAdmin), h.User.DeleteUser)

	// API Settings — admin only (integrations, API keys)
	v1.Get("/settings/bos24",
		middleware.RequireRole(domain.RoleAdmin),
		h.Settings.GetBOS24Settings,
	)
	v1.Patch("/settings/bos24",
		middleware.RequireRole(domain.RoleAdmin),
		h.Settings.UpdateBOS24Settings,
	)

	// Viewings / Calendar
	v1.Get("/viewings", h.Viewing.List)
	v1.Get("/viewings/:id", h.Viewing.Get)
	v1.Post("/viewings",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Viewing.Create,
	)
	v1.Patch("/viewings/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Viewing.Update,
	)
	v1.Patch("/viewings/:id/status",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Viewing.UpdateStatus,
	)
	v1.Delete("/viewings/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Viewing.Delete,
	)

	// Import / Export
	v1.Get("/import/template/:entity", h.ImportExport.Template)
	v1.Post("/import/contacts",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.ImportExport.ImportContacts,
	)
	v1.Post("/import/leads",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.ImportExport.ImportLeads,
	)
	v1.Get("/export/contacts", h.ImportExport.ExportContacts)
	v1.Get("/export/leads", h.ImportExport.ExportLeads)
	v1.Get("/export/listings",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.ImportExport.ExportListings,
	)

	// Agent Performance & Gamification
	v1.Get("/performance/leaderboard", h.Performance.Leaderboard)
	v1.Get("/performance/agent/:id", h.Performance.AgentKPIs)
	v1.Get("/performance/agent/:id/trends", h.Performance.AgentTrends)
	v1.Get("/performance/agent/:id/targets", h.Performance.AgentTargets)
	v1.Post("/performance/targets",
		middleware.RequireRole(domain.RoleAdmin),
		h.Performance.UpsertTarget,
	)
	v1.Delete("/performance/targets/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Performance.DeleteTarget,
	)

	// Commission Structures
	v1.Get("/commissions/structures",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.ListStructures,
	)
	v1.Post("/commissions/structures",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.CreateStructure,
	)
	v1.Patch("/commissions/structures/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.UpdateStructure,
	)
	v1.Delete("/commissions/structures/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.DeleteStructure,
	)

	// Agent Commissions
	v1.Get("/commissions", h.Commission.List)
	v1.Post("/commissions",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.Create,
	)
	v1.Post("/commissions/calculate",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Commission.Calculate,
	)
	v1.Patch("/commissions/:id/status",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.UpdateStatus,
	)
	v1.Patch("/commissions/:id/amount",
		middleware.RequireRole(domain.RoleAdmin),
		h.Commission.UpdateAmount,
	)

	// Lead Rotation — admin: configure; agents: trigger auto-assign
	v1.Get("/settings/lead-rotation",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeadRotation.GetSettings,
	)
	v1.Patch("/settings/lead-rotation",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeadRotation.UpdateSettings,
	)
	v1.Post("/leads/:id/auto-assign",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.LeadRotation.AutoAssign,
	)

	// BOS24 Integration — admin only (marketplace listing + inquiry sync)
	v1.Get("/settings/bos24-integration",
		middleware.RequireRole(domain.RoleAdmin),
		h.BOS24Integration.GetSettings,
	)
	v1.Patch("/settings/bos24-integration",
		middleware.RequireRole(domain.RoleAdmin),
		h.BOS24Integration.UpdateSettings,
	)
	v1.Post("/settings/bos24-integration/register",
		middleware.RequireRole(domain.RoleAdmin),
		h.BOS24Integration.RegisterWebhook,
	)
	v1.Post("/settings/bos24-integration/sync",
		middleware.RequireRole(domain.RoleAdmin),
		h.BOS24Integration.SyncNow,
	)

	// Approval Workflows — admin only (config + review)
	v1.Get("/approval-config",
		middleware.RequireRole(domain.RoleAdmin),
		h.Approval.GetConfig,
	)
	v1.Patch("/approval-config",
		middleware.RequireRole(domain.RoleAdmin),
		h.Approval.SaveConfig,
	)
	v1.Get("/approval-requests",
		middleware.RequireRole(domain.RoleAdmin),
		h.Approval.ListRequests,
	)
	v1.Post("/approval-requests/:id/review",
		middleware.RequireRole(domain.RoleAdmin),
		h.Approval.ReviewRequest,
	)

	// Company Settings — admin only (invoice details, VAT number, bank info)
	v1.Get("/settings/company",
		middleware.RequireRole(domain.RoleAdmin),
		h.Settings.GetCompanySettings,
	)
	v1.Patch("/settings/company",
		middleware.RequireRole(domain.RoleAdmin),
		h.Settings.UpdateCompanySettings,
	)

	// Pipeline Stages — admin only; define custom sales pipeline stages
	v1.Get("/pipeline-stages",
		middleware.RequireRole(domain.RoleAdmin),
		h.PipelineStage.List,
	)
	v1.Post("/pipeline-stages",
		middleware.RequireRole(domain.RoleAdmin),
		h.PipelineStage.Create,
	)
	v1.Patch("/pipeline-stages/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.PipelineStage.Update,
	)
	v1.Delete("/pipeline-stages/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.PipelineStage.Delete,
	)
	v1.Post("/pipeline-stages/reorder",
		middleware.RequireRole(domain.RoleAdmin),
		h.PipelineStage.Reorder,
	)
	v1.Post("/pipeline-stages/reset-default",
		middleware.RequireRole(domain.RoleAdmin),
		h.PipelineStage.ResetDefault,
	)

	// API Keys — admin only (for external integrations)
	v1.Get("/settings/api-keys",
		middleware.RequireRole(domain.RoleAdmin),
		h.ApiKey.ListApiKeys,
	)
	v1.Post("/settings/api-keys",
		middleware.RequireRole(domain.RoleAdmin),
		h.ApiKey.CreateApiKey,
	)
	v1.Delete("/settings/api-keys/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.ApiKey.RevokeApiKey,
	)

	// Outbound Webhooks — admin only
	v1.Get("/settings/webhooks",
		middleware.RequireRole(domain.RoleAdmin),
		h.WebhookSub.List,
	)
	v1.Post("/settings/webhooks",
		middleware.RequireRole(domain.RoleAdmin),
		h.WebhookSub.Create,
	)
	v1.Delete("/settings/webhooks/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.WebhookSub.Delete,
	)
	v1.Post("/settings/webhooks/:id/test",
		middleware.RequireRole(domain.RoleAdmin),
		h.WebhookSub.Test,
	)

	// Billing & plans — all authenticated users view; admin manages
	v1.Get("/billing", h.Billing.GetBilling)
	v1.Get("/billing/usage", h.Billing.GetUsage)
	v1.Post("/billing/checkout",
		middleware.RequireRole(domain.RoleAdmin),
		h.Billing.CreateCheckout,
	)
	v1.Post("/billing/portal",
		middleware.RequireRole(domain.RoleAdmin),
		h.Billing.CreatePortal,
	)

	// Contacts — viewers: read-only; agents: create+update; admin: delete
	v1.Get("/contacts", h.Contact.List)
	v1.Get("/contacts/:id", h.Contact.Get)
	v1.Post("/contacts",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Contact.Create,
	)
	v1.Patch("/contacts/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Contact.Update,
	)
	v1.Delete("/contacts/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Contact.Delete,
	)

	// Leads / Pipeline — viewers: read-only; agents: create+move; admin: all
	v1.Get("/leads", h.Lead.KanbanBoard)
	v1.Get("/leads/search", h.Lead.List)
	v1.Get("/leads/:id", h.Lead.Get)
	v1.Post("/leads",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lead.Create,
	)
	v1.Patch("/leads/:id/stage",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lead.UpdateStage,
	)
	v1.Patch("/leads/:id/notes",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lead.UpdateNotes,
	)
	v1.Patch("/leads/:id/assign",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lead.Assign,
	)
	v1.Delete("/leads/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Lead.Delete,
	)
	v1.Get("/leads/:id/communications", h.Lead.GetCommunications)
	v1.Get("/leads/:id/tags", h.Lead.GetTags)
	v1.Post("/leads/:id/tags",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lead.AddTag,
	)
	v1.Delete("/leads/:id/tags/:tag",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lead.RemoveTag,
	)

	// WhatsApp inbox — all authenticated users read; agents+ can close/send
	v1.Get("/threads", h.WhatsApp.ListThreads)
	v1.Get("/threads/:id", h.WhatsApp.GetThread)
	v1.Get("/threads/:id/messages", h.WhatsApp.GetMessages)
	v1.Post("/threads/:id/close",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.WhatsApp.CloseThread,
	)
	v1.Post("/threads/:id/reopen",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.WhatsApp.ReopenThread,
	)

	// WhatsApp outbound — agents+ send messages
	v1.Post("/threads/:id/send-message",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.WhatsAppOutbound.SendMessage,
	)
	v1.Post("/threads/:id/send-template",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.WhatsAppOutbound.SendTemplate,
	)
	v1.Get("/threads/:id/outbound-messages",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.WhatsAppOutbound.GetOutboundMessages,
	)
	v1.Post("/threads/:id/send-media",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.WhatsAppOutbound.SendMedia,
	)

	// AI (manual) — agents and admin only, quota enforced
	aiQuota := middleware.CheckQuota(billingRepo, rdb, "ai")
	aiUserQuota := middleware.CheckUserAIQuota(billingRepo, rdb)

	v1.Post("/ai/summarize/:thread_id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.AI.SummarizeThread,
	)
	// AI buyer profile extraction
	v1.Post("/ai/extract-buyer-profile/:thread_id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.AI.ExtractBuyerProfile,
	)

	// AI scoring and drafting
	v1.Post("/ai/score-lead/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.AI.ScoreLead,
	)
	v1.Post("/ai/score-contact/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.AI.ScoreContact,
	)
	v1.Post("/ai/draft-reply/:thread_id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.AI.DraftReply,
	)

	// Non-PII: uses Gemini if configured, falls back to Ollama
	v1.Post("/ai/describe-listing",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota,
		h.AI.DescribePropertyListing,
	)

	// Message Analysis — agents and admin only (intent parsing, enrichment, auto-lead)
	v1.Post("/messages/analyze",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.Message.AnalyzeMessage,
	)
	v1.Post("/messages/suggest-action",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.Message.SuggestNextAction,
	)
	v1.Post("/messages/auto-create-lead",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		aiQuota, aiUserQuota,
		h.Message.AutoCreateLead,
	)

	// Real Estate Market Data (BuyOrSell24) — agents and admin only, quota enforced
	bos24Quota := middleware.CheckQuota(billingRepo, rdb, "bos24")
	prop := v1.Group("/properties",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		bos24Quota,
	)

	// AI search
	prop.Post("/search", h.Property.SearchProperties)
	prop.Post("/report/pdf", h.Property.GeneratePropertyReport)
	prop.Post("/projects/search", h.Property.SearchProjects)
	prop.Post("/ai/describe", h.Property.DescribeProperty)

	// Transactions
	prop.Get("/transactions", h.Property.GetTransactions)
	prop.Get("/transactions/areas", h.Property.GetTransactionAreas)

	// Buildings
	prop.Get("/buildings", h.Property.GetBuildings)
	prop.Get("/buildings/:id", h.Property.GetBuildingByID)

	// Areas
	prop.Get("/areas", h.Property.GetAreas)
	prop.Get("/areas/:slug/summary", h.Property.GetAreaSummary)
	prop.Get("/areas/:slug/buildings", h.Property.GetAreaBuildings)

	// Map data
	prop.Get("/map/areas", h.Property.GetMapAreas)
	prop.Get("/pois", h.Property.GetNearbyPOIs)
	prop.Get("/schools/nearby", h.Property.GetNearbySchools)

	// Rentals & yield
	prop.Get("/rentals", h.Property.GetRentals)
	prop.Get("/rentals/ejari", h.Property.GetEjariRentals)
	prop.Get("/rentals/ejari/yield", h.Property.GetEjariYield)

	// Developers & projects
	prop.Get("/developers", h.Property.GetDevelopers)
	prop.Get("/projects", h.Property.GetProjects)

	// Units & valuations
	prop.Get("/units", h.Property.GetUnits)
	prop.Get("/valuations", h.Property.GetValuations)

	// Analytics
	prop.Get("/yield-analysis", h.Property.GetYieldAnalysis)
	prop.Get("/comparables", h.Property.GetComparables)
	prop.Get("/market-trends", h.Property.GetMarketTrends)

	// Insights
	prop.Get("/insights/market-overview", h.Property.GetMarketOverview)
	prop.Get("/insights/area-comparison", h.Property.GetAreaComparison)
	prop.Get("/insights/price-trends", h.Property.GetPriceTrends)
	prop.Get("/insights/top-areas", h.Property.GetTopAreas)

	// Brokers
	prop.Get("/brokers", h.Property.GetBrokers)

	// Transaction detail routes (must come before /transactions/:id to avoid ambiguity)
	prop.Get("/transactions/by-project/:name", h.Property.GetTransactionsByProject)
	prop.Get("/transactions/area/:name/summary", h.Property.GetAreaTransactionSummary)
	prop.Get("/transactions/:id", h.Property.GetTransaction)
	prop.Get("/transactions/:id/enriched", h.Property.GetEnrichedTransaction)

	// Rental sub-routes
	prop.Get("/rentals/stats", h.Property.GetRentalStats)
	prop.Get("/rentals/areas", h.Property.GetRentalAreas)
	prop.Get("/rentals/project/:name", h.Property.GetRentalsByProject)
	prop.Get("/rentals/building/:name", h.Property.GetRentalsByBuilding)

	// Lands
	prop.Get("/lands", h.Property.GetLands)
	prop.Get("/lands/:id", h.Property.GetLand)

	// Map extras
	prop.Get("/map/config", h.Property.GetMapConfig)
	prop.Get("/map/bounds", h.Property.GetMapBounds)
	prop.Get("/map/poi-categories", h.Property.GetPOICategories)
	prop.Get("/map/heatmap", h.Property.GetPropertyHeatmap)
	prop.Get("/map/area/:name", h.Property.GetAreaLocation)

	// Single entity lookups
	prop.Get("/areas/:id", h.Property.GetAreaByID)
	prop.Get("/developers/:id", h.Property.GetDeveloper)
	prop.Get("/projects/:id", h.Property.GetProject)
	prop.Get("/units/:id", h.Property.GetUnit)
	prop.Get("/valuations/:id", h.Property.GetValuationByID)

	// Notifications — personal; no role restriction beyond auth
	v1.Get("/notifications", h.Notification.List)
	v1.Patch("/notifications/read-all", h.Notification.MarkAllRead)
	v1.Patch("/notifications/:id/read", h.Notification.MarkRead)

	// Deals — viewers: read-only; agents: create+stage; admin: all
	v1.Get("/deals", h.Deal.List)
	v1.Get("/deals/:id", h.Deal.Get)
	v1.Get("/deals/:id/invoices", h.Deal.ListInvoices)
	v1.Post("/deals",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Deal.Create,
	)
	v1.Patch("/deals/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Deal.Update,
	)
	v1.Patch("/deals/:id/stage",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Deal.UpdateStage,
	)
	v1.Delete("/deals/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Deal.Delete,
	)

	// Invoices — agents: create+view; admin: send+update status
	v1.Get("/invoices", h.Invoice.List)
	v1.Get("/invoices/:id", h.Invoice.Get)
	v1.Get("/invoices/:id/pdf", h.Invoice.DownloadPDF)
	v1.Post("/invoices",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Invoice.Create,
	)
	v1.Post("/invoices/:id/send",
		middleware.RequireRole(domain.RoleAdmin),
		h.Invoice.Send,
	)
	v1.Patch("/invoices/:id/status",
		middleware.RequireRole(domain.RoleAdmin),
		h.Invoice.UpdateStatus,
	)

	// Email — agents and admin only
	v1.Get("/emails",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Email.ListAllEmailHistory,
	)
	v1.Post("/emails/send",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Email.SendEmail,
	)
	v1.Get("/emails/history",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Email.GetEmailHistory,
	)

	// Rental Properties — agents: create+view+update; admin: all; viewers: read-only
	v1.Get("/rental-properties", h.RentalProperty.List)
	v1.Get("/rental-properties/:id", h.RentalProperty.Get)
	v1.Post("/rental-properties",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.RentalProperty.Create,
	)
	v1.Patch("/rental-properties/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.RentalProperty.Update,
	)
	v1.Delete("/rental-properties/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.RentalProperty.Delete,
	)

	// Listings — agents: create+view+update; admin: all; viewers: read-only
	v1.Get("/listings", h.Listing.List)
	v1.Get("/listings/:id", h.Listing.Get)
	v1.Post("/listings",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Listing.Create,
	)
	v1.Patch("/listings/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Listing.Update,
	)
	v1.Patch("/listings/:id/status",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Listing.UpdateStatus,
	)
	v1.Delete("/listings/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Listing.Delete,
	)

	// Listing marketing tools
	v1.Get("/listings/:id/brochure", h.Marketing.DownloadBrochure)
	v1.Get("/listings/:id/qr", h.Marketing.GenerateQR)
	v1.Post("/listings/:id/email-campaign",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Marketing.EmailCampaign,
	)

	// Offers — buyer offers on listings, negotiation, auto-deal on accept
	v1.Get("/offers", h.Offer.List)
	v1.Get("/offers/:id", h.Offer.Get)
	v1.Post("/offers",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Offer.Create,
	)
	v1.Patch("/offers/:id/status",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Offer.UpdateStatus,
	)
	v1.Post("/offers/:id/counter",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Offer.Counter,
	)
	v1.Post("/offers/:id/accept",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Offer.Accept,
	)
	v1.Delete("/offers/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Offer.Delete,
	)

	// Tenants — agents: create+view+update; admin: all; viewers: read-only
	v1.Get("/tenants", h.Tenant.List)
	v1.Get("/tenants/:id", h.Tenant.Get)
	v1.Post("/tenants",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Tenant.Create,
	)
	v1.Patch("/tenants/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Tenant.Update,
	)
	v1.Delete("/tenants/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Tenant.Delete,
	)
	v1.Post("/tenants/:id/verify",
		middleware.RequireRole(domain.RoleAdmin),
		h.Tenant.Verify,
	)

	// Lease Templates — agents: view; admin: all
	v1.Get("/lease-templates", h.LeaseTemplate.List)
	v1.Get("/lease-templates/:id", h.LeaseTemplate.Get)
	v1.Post("/lease-templates",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseTemplate.Create,
	)
	v1.Patch("/lease-templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseTemplate.Update,
	)
	v1.Delete("/lease-templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseTemplate.Delete,
	)

	// Message Templates — agents+viewers: view; admin: all
	v1.Get("/message-templates", h.MessageTemplate.List)
	v1.Get("/message-templates/:id", h.MessageTemplate.Get)
	v1.Post("/message-templates",
		middleware.RequireRole(domain.RoleAdmin),
		h.MessageTemplate.Create,
	)
	v1.Patch("/message-templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.MessageTemplate.Update,
	)
	v1.Delete("/message-templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.MessageTemplate.Delete,
	)

	// Leases — agents: create+view+update; admin: all; viewers: read-only
	v1.Get("/leases", h.Lease.List)
	v1.Get("/leases/:id", h.Lease.Get)
	v1.Post("/leases",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lease.Create,
	)
	v1.Patch("/leases/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Lease.Update,
	)
	v1.Delete("/leases/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Lease.Delete,
	)

	// Payments — agents: create+view+update; admin: all; viewers: read-only
	v1.Get("/payments", h.Payment.List)
	v1.Get("/payments/:id", h.Payment.Get)
	v1.Post("/payments",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Payment.Create,
	)
	v1.Patch("/payments/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Payment.Update,
	)
	v1.Delete("/payments/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Payment.Delete,
	)

	// Bank Integrations — admin only
	v1.Get("/bank-integrations", h.BankIntegration.List)
	v1.Get("/bank-integrations/:id", h.BankIntegration.Get)
	v1.Post("/bank-integrations",
		middleware.RequireRole(domain.RoleAdmin),
		h.BankIntegration.Create,
	)
	v1.Patch("/bank-integrations/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.BankIntegration.Update,
	)
	v1.Delete("/bank-integrations/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.BankIntegration.Delete,
	)

	// Bank Statements — agents: upload+view; admin: all
	v1.Get("/bank-statements",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.BankStatement.List,
	)
	v1.Get("/bank-statements/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.BankStatement.Get,
	)
	v1.Post("/bank-statements/upload",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.BankStatement.Upload,
	)
	v1.Delete("/bank-statements/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.BankStatement.Delete,
	)

	// Payment Confirmations — agents: view+send; admin: all
	v1.Get("/payments/:payment_id/confirmation",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.PaymentConfirmation.GetByPayment,
	)
	v1.Post("/payments/:payment_id/send-confirmation",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.PaymentConfirmation.Send,
	)

	// Analytics — read-only; all authenticated users
	v1.Get("/analytics/tenant-overview", h.Analytics.GetTenantOverview)
	v1.Get("/analytics/properties", h.Analytics.ListPropertiesAnalytics)
	v1.Get("/analytics/properties/:propertyID", h.Analytics.GetPropertyAnalytics)
	v1.Get("/analytics/tenants", h.Analytics.ListTenantsPerformance)
	v1.Get("/analytics/tenants/:tenantID", h.Analytics.GetTenantPerformance)
	v1.Get("/analytics/financial", h.Analytics.GetFinancialAnalytics)
	v1.Get("/analytics/maintenance", h.Analytics.GetMaintenanceAnalytics)

	// Expenses — agents: create+view+update; admin: all
	v1.Get("/expense-categories",
		h.Expense.ListCategories,
	)
	v1.Post("/expense-categories",
		middleware.RequireRole(domain.RoleAdmin),
		h.Expense.CreateCategory,
	)
	v1.Get("/expenses", h.Expense.ListExpenses)
	v1.Get("/expenses/:id", h.Expense.GetExpense)
	v1.Post("/expenses",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Expense.CreateExpense,
	)
	v1.Patch("/expenses/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Expense.UpdateExpense,
	)
	v1.Delete("/expenses/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Expense.DeleteExpense,
	)
	v1.Post("/expenses/:id/approve",
		middleware.RequireRole(domain.RoleAdmin),
		h.Expense.ApproveExpense,
	)

	// Inspection Templates — admin: create/update; all: list
	v1.Get("/inspection-templates", h.Inspection.ListTemplates)
	v1.Post("/inspection-templates",
		middleware.RequireRole(domain.RoleAdmin),
		h.Inspection.CreateTemplate,
	)

	// Inspections — agents: create+view+update; admin: all
	v1.Get("/inspections", h.Inspection.ListInspections)
	v1.Get("/inspections/:id", h.Inspection.GetInspection)
	v1.Post("/inspections",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Inspection.CreateInspection,
	)
	v1.Patch("/inspections/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Inspection.UpdateInspection,
	)
	v1.Post("/inspections/:id/complete",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Inspection.CompleteInspection,
	)

	// Maintenance Tasks — agents: create+view+update; admin: all
	v1.Get("/maintenance-tasks", h.Maintenance.List)
	v1.Get("/maintenance-tasks/:id", h.Maintenance.Get)
	v1.Post("/maintenance-tasks",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Maintenance.Create,
	)
	v1.Patch("/maintenance-tasks/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Maintenance.Update,
	)
	v1.Post("/maintenance-tasks/:id/complete",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Maintenance.Complete,
	)
	v1.Post("/maintenance-tasks/:id/photos",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Maintenance.AddPhoto,
	)
	v1.Get("/maintenance-tasks/:id/photos", h.Maintenance.GetPhotos)
	v1.Delete("/maintenance-tasks/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Maintenance.Delete,
	)

	// Lease Renewals
	v1.Get("/lease-renewals", h.LeaseRenewal.List)
	v1.Get("/lease-renewals/:id", h.LeaseRenewal.Get)
	v1.Post("/lease-renewals/:lease_id/initiate",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseRenewal.Initiate,
	)
	v1.Put("/lease-renewals/:id/propose",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.LeaseRenewal.Propose,
	)
	v1.Post("/lease-renewals/:id/send-offer",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.LeaseRenewal.SendOffer,
	)
	v1.Put("/lease-renewals/:id/accept",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.LeaseRenewal.Accept,
	)
	v1.Put("/lease-renewals/:id/reject",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.LeaseRenewal.Reject,
	)
	v1.Post("/lease-renewals/:id/counter-offer",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.LeaseRenewal.CounterOffer,
	)

	// Renewal Templates
	v1.Get("/renewal-templates", h.LeaseRenewal.ListTemplates)
	v1.Post("/renewal-templates",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseRenewal.CreateTemplate,
	)
	v1.Patch("/renewal-templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseRenewal.UpdateTemplate,
	)
	v1.Delete("/renewal-templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.LeaseRenewal.DeleteTemplate,
	)

	// Document Templates
	v1.Get("/documents/templates", h.Document.ListTemplates)
	v1.Get("/documents/templates/:id", h.Document.GetTemplate)
	v1.Post("/documents/templates",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Document.CreateTemplate,
	)
	v1.Patch("/documents/templates/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Document.UpdateTemplate,
	)
	v1.Delete("/documents/templates/:id",
		middleware.RequireRole(domain.RoleAdmin),
		h.Document.DeleteTemplate,
	)

	// Documents
	v1.Get("/documents", h.Document.ListDocuments)
	v1.Get("/documents/:id", h.Document.GetDocument)
	v1.Post("/documents",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Document.CreateDocument,
	)
	v1.Post("/documents/:id/request-signature",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Document.SendForSignature,
	)
	v1.Patch("/documents/signatures/:id/mark-signed",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Document.MarkSigned,
	)
	v1.Delete("/documents/:id",
		middleware.RequireRole(domain.RoleAdmin, domain.RoleAgent),
		h.Document.DeleteDocument,
	)

	// Audit Log — admin only
	v1.Get("/audit-logs",
		middleware.RequireRole(domain.RoleAdmin),
		h.AuditLog.List,
	)

	// Health — checks DB and Redis connectivity
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		errs := fiber.Map{}

		if err := pool.Ping(ctx); err != nil {
			errs["database"] = "unreachable"
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			errs["redis"] = "unreachable"
		}

		if len(errs) > 0 {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "degraded",
				"checks": errs,
			})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Swagger UI — gated behind BasicAuth in production
	if cfg.AppEnv == "production" {
		app.Get("/docs/*",
			basicauth.New(basicauth.Config{
				Users: map[string]string{
					"admin": cfg.JWTSecret[:16], // use first 16 chars of JWT secret as password
				},
			}),
			fiberswagger.WrapHandler,
		)
	} else {
		app.Get("/docs/*", fiberswagger.WrapHandler)
	}
}
