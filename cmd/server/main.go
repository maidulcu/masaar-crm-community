// Command server is the Masaar CRM API entry point.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver used by goose
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"

	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/api"
	"github.com/maidulcu/masaar-crm/internal/api/handler"
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/billing"
	"github.com/maidulcu/masaar-crm/internal/bos24"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/docusign"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/sms"
	"github.com/maidulcu/masaar-crm/internal/webhook"
	"github.com/maidulcu/masaar-crm/internal/whatsapp"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("refusing to start: %v", err)
	}
	if !cfg.IsProduction() && cfg.JWTSecret == "change-me-in-production" {
		log.Println("WARNING: running with the default JWT_SECRET (development only)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runMigrations(cfg.DatabaseURL, envOr("MIGRATIONS_DIR", "migrations")); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	pool, err := repo.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis: %v", err)
	}

	// ── Repositories ─────────────────────────────────────────────────────────
	userRepo := repo.NewUserRepo(pool)
	companyRepo := repo.NewCompanyRepo(pool)
	auditRepo := repo.NewAuditLogRepo(pool)
	contactRepo := repo.NewContactRepo(pool)
	leadRepo := repo.NewLeadRepo(pool)
	leadTagRepo := repo.NewLeadTagRepo(pool)
	commHistRepo := repo.NewCommunicationHistoryRepo(pool)
	waRepo := repo.NewWhatsAppRepo(pool)
	waOutboundRepo := repo.NewWhatsAppOutboundRepo(pool)
	notificationRepo := repo.NewNotificationRepo(pool)
	dealRepo := repo.NewDealRepo(pool)
	invoiceRepo := repo.NewInvoiceRepo(pool)
	statsRepo := repo.NewStatsRepo(pool)
	apiSettingsRepo := repo.NewSettingsRepo(pool)
	companySettingsRepo := repo.NewCompanySettingsRepo(pool)
	emailRepo := repo.NewEmailRepository(pool)
	rentalPropertyRepo := repo.NewRentalPropertyRepo(pool)
	tenantRepo := repo.NewTenantRepo(pool)
	leaseTemplateRepo := repo.NewLeaseTemplateRepo(pool)
	leaseRepo := repo.NewLeaseRepo(pool)
	paymentRepo := repo.NewPaymentRepo(pool)
	bankIntegrationRepo := repo.NewBankIntegrationRepo(pool)
	bankStatementRepo := repo.NewBankStatementRepo(pool)
	paymentConfirmationRepo := repo.NewPaymentConfirmationRepo(pool)
	analyticsRepo := repo.NewAnalyticsRepository(pool, rdb)
	expenseRepo := repo.NewExpenseRepository(pool)
	inspectionTemplateRepo := repo.NewInspectionTemplateRepo(pool)
	inspectionRepo := repo.NewInspectionRepo(pool)
	maintenanceRepo := repo.NewMaintenanceTaskRepo(pool)
	leaseRenewalRepo := repo.NewLeaseRenewalRepo(pool)
	renewalTemplateRepo := repo.NewRenewalTemplateRepo(pool)
	renewalLogRepo := repo.NewRenewalCommunicationLogRepo(pool)
	documentRepo := repo.NewDocumentRepo(pool)
	apiKeyRepo := repo.NewApiKeyRepo(pool)
	webhookRepo := repo.NewWebhookRepo(pool)
	billingRepo := repo.NewBillingRepo(pool)
	messageTemplateRepo := repo.NewMessageTemplateRepo(pool)
	listingRepo := repo.NewListingRepo(pool)
	bos24Repo := repo.NewBOS24IntegrationRepo(pool)
	offerRepo := repo.NewOfferRepo(pool)
	leadRotationRepo := repo.NewLeadRotationRepo(pool)
	commissionRepo := repo.NewCommissionRepo(pool)
	performanceRepo := repo.NewPerformanceRepo(pool)
	viewingRepo := repo.NewViewingRepo(pool)
	pipelineStageRepo := repo.NewPipelineStageRepo(pool)
	approvalRepo := repo.NewApprovalRepo(pool)
	paymentReminderRepo := repo.NewPaymentReminderRepo(pool)

	// ── Services / clients ───────────────────────────────────────────────────
	hub := ws.NewHub()
	dispatcher := webhook.NewDispatcher(webhookRepo)

	emailSvc := email.NewService(&email.Config{
		SMTPHost:         cfg.SMTPHost,
		SMTPPort:         cfg.SMTPPort,
		SMTPUser:         cfg.SMTPUser,
		SMTPPassword:     cfg.SMTPPassword,
		FromEmail:        cfg.SMTPFromEmail,
		FromName:         cfg.SMTPFromName,
		AzureEndpoint:    cfg.AzureCommEndpoint,
		AzureKey:         cfg.AzureCommKey,
		AzureFromAddress: cfg.AzureCommFromAddress,
	})

	var smsClient *sms.Client
	if cfg.SMSCountryEnabled {
		smsClient = sms.NewClient(cfg.SMSCountryAuthKey, cfg.SMSCountryAuthToken, cfg.SMSCountrySenderID)
	}

	aiSensitive := ai.NewClient(cfg.OllamaBaseURL, cfg.OllamaModel)
	aiCloud := ai.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	scoringSvc := ai.NewScoringService(leadRepo, commHistRepo, leadTagRepo)
	taggingSvc := ai.NewTaggingService(aiSensitive, leadRepo, waRepo, leadTagRepo, contactRepo)
	confirmationSvc := ai.NewPaymentConfirmationService(paymentRepo, paymentConfirmationRepo, leaseRepo, tenantRepo, rentalPropertyRepo, companySettingsRepo, emailSvc)

	waSender := whatsapp.NewSender(&whatsapp.SenderConfig{
		BaseURL:       whatsAppBaseURL(cfg.WABaseURL, cfg.WAAPIVersion),
		PhoneNumberID: cfg.WAPhoneNumberID,
		AccessToken:   cfg.WAAccessToken,
	})

	bos24Client := bos24.NewClient(cfg.BOS24Token, cfg.BOS24BaseURL, rdb)
	bos24Sync := bos24.NewSyncService(bos24Repo, contactRepo, hub)

	var dsClient *docusign.Client
	if cfg.DocusignIntegrationKey != "" {
		dsClient = docusign.NewClient(docusign.Config{
			IntegrationKey: cfg.DocusignIntegrationKey,
			PrivateKeyPEM:  cfg.DocusignPrivateKey,
			UserID:         cfg.DocusignUserID,
			AccountID:      cfg.DocusignAccountID,
			BaseURL:        cfg.DocusignBaseURL,
			WebhookSecret:  cfg.DocusignWebhookSecret,
		})
	}

	stripeCfg := &billing.StripeConfig{
		SecretKey:       cfg.StripeSecretKey,
		WebhookSecret:   cfg.StripeWebhookSecret,
		PriceIDStarter:  cfg.StripePriceIDStarter,
		PriceIDPro:      cfg.StripePriceIDPro,
		PriceIDBusiness: cfg.StripePriceIDBusiness,
		AppURL:          cfg.AppURL,
	}

	// ── Handlers ─────────────────────────────────────────────────────────────
	h := &api.Handlers{
		Auth:                handler.NewAuthHandler(userRepo, companyRepo, rdb, cfg, auditRepo, emailSvc, smsClient),
		User:                handler.NewUserHandler(userRepo, auditRepo, emailSvc, cfg),
		Stats:               handler.NewStatsHandler(statsRepo),
		Contact:             handler.NewContactHandler(contactRepo, auditRepo),
		Lead:                handler.NewLeadHandler(leadRepo, contactRepo, commHistRepo, scoringSvc, leadTagRepo, hub, auditRepo, dispatcher, pipelineStageRepo),
		WhatsApp:            handler.NewWhatsAppHandler(waRepo, contactRepo, taggingSvc, hub, cfg),
		WhatsAppOutbound:    handler.NewWhatsAppOutboundHandler(waSender, waOutboundRepo, waRepo),
		AI:                  handler.NewAIHandler(aiSensitive, aiCloud, contactRepo, leadRepo, waRepo),
		Message:             handler.NewMessageHandler(aiSensitive, waRepo, contactRepo, leadRepo, commHistRepo, leadTagRepo, scoringSvc, hub),
		Notification:        handler.NewNotificationHandler(notificationRepo),
		Deal:                handler.NewDealHandler(dealRepo, invoiceRepo, auditRepo),
		Invoice:             handler.NewInvoiceHandler(invoiceRepo, dealRepo, companySettingsRepo),
		Property:            handler.NewPropertyHandler(bos24Client, companySettingsRepo),
		Settings:            handler.NewSettingsHandler(apiSettingsRepo, companySettingsRepo),
		Email:               handler.NewEmailHandler(emailSvc, emailRepo),
		RentalProperty:      handler.NewRentalPropertyHandler(rentalPropertyRepo),
		Tenant:              handler.NewTenantHandler(tenantRepo),
		LeaseTemplate:       handler.NewLeaseTemplateHandler(leaseTemplateRepo),
		Lease:               handler.NewLeaseHandler(leaseRepo),
		Payment:             handler.NewPaymentHandler(paymentRepo),
		BankIntegration:     handler.NewBankIntegrationHandler(bankIntegrationRepo),
		BankStatement:       handler.NewBankStatementHandler(bankStatementRepo),
		PaymentConfirmation: handler.NewPaymentConfirmationHandler(paymentConfirmationRepo, confirmationSvc),
		Analytics:           handler.NewAnalyticsHandler(analyticsRepo),
		Expense:             handler.NewExpenseHandler(expenseRepo),
		Inspection:          handler.NewInspectionHandler(inspectionTemplateRepo, inspectionRepo),
		Maintenance:         handler.NewMaintenanceTaskHandler(maintenanceRepo),
		LeaseRenewal:        handler.NewLeaseRenewalHandler(leaseRenewalRepo, renewalTemplateRepo, renewalLogRepo),
		Document:            handler.NewDocumentHandler(documentRepo, auditRepo, dsClient),
		ApiKey:              handler.NewApiKeyHandler(apiKeyRepo),
		PublicLead:          handler.NewPublicLeadHandler(contactRepo, leadRepo, dispatcher),
		WebhookSub:          handler.NewWebhookSubHandler(webhookRepo, dispatcher),
		Billing:             handler.NewBillingHandler(billingRepo, companySettingsRepo, stripeCfg),
		MessageTemplate:     handler.NewMessageTemplateHandler(messageTemplateRepo),
		AuditLog:            handler.NewAuditHandler(auditRepo),
		Listing:             handler.NewListingHandler(listingRepo, approvalRepo),
		BOS24Integration:    handler.NewBOS24IntegrationHandler(bos24Repo, bos24Sync, cfg),
		Offer:               handler.NewOfferHandler(offerRepo, contactRepo, leadRepo, dealRepo, hub),
		LeadRotation:        handler.NewLeadRotationHandler(leadRotationRepo, leadRepo, userRepo, hub),
		Commission:          handler.NewCommissionHandler(commissionRepo),
		Performance:         handler.NewPerformanceHandler(performanceRepo),
		Viewing:             handler.NewViewingHandler(viewingRepo, notificationRepo, hub),
		Marketing:           handler.NewMarketingHandler(listingRepo, contactRepo, companySettingsRepo, userRepo, emailSvc),
		ImportExport:        handler.NewImportExportHandler(contactRepo, leadRepo, listingRepo),
		PipelineStage:       handler.NewPipelineStageHandler(pipelineStageRepo),
		Approval:            handler.NewApprovalHandler(approvalRepo),
	}
	if cfg.DocusignWebhookSecret != "" {
		h.DocusignWebhook = handler.NewDocusignWebhookHandler(documentRepo, cfg.DocusignWebhookSecret)
	}
	_ = paymentReminderRepo // reserved for the scheduled reminder service (not part of Community Edition)

	// ── HTTP server ──────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:                 "Masaar CRM",
		DisableStartupMessage:   cfg.IsProduction(),
		BodyLimit:               10 * 1024 * 1024,
		ReadTimeout:             30 * time.Second,
		WriteTimeout:            30 * time.Second,
		IdleTimeout:             120 * time.Second,
		ErrorHandler:            errorHandler,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          splitList(cfg.TrustedProxies),
	})
	app.Use(recover.New())
	app.Use(middleware.PIISafeLogger())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-API-Key",
		MaxAge:       600,
	}))

	api.RegisterRoutes(app, h, hub, cfg, rdb, pool, apiKeyRepo, billingRepo, companyRepo)

	errCh := make(chan error, 1)
	go func() { errCh <- app.Listen(":" + cfg.Port) }()
	log.Printf("Masaar CRM listening on :%s (env=%s)", cfg.Port, cfg.AppEnv)

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("server: %v", err)
		}
	case <-ctx.Done():
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}

// errorHandler never leaks internal error text to clients.
func errorHandler(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	log.Printf("unhandled error: %s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

func runMigrations(databaseURL, dir string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, dir)
}

// whatsAppBaseURL returns the Graph API base including the version segment, since the
// sender builds URLs as {base}/{phone_number_id}/messages. WA_BASE_URL may be given with
// or without the version (".env.example" documents it without).
func whatsAppBaseURL(base, version string) string {
	base = strings.TrimRight(base, "/")
	u, err := url.Parse(base)
	if err != nil || (u.Path != "" && u.Path != "/") {
		return base
	}
	return fmt.Sprintf("%s/%s", base, version)
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
