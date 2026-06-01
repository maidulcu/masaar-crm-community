package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type AIHandler struct{}

func NewAIHandler(sensitive, cloud *ai.Client, contacts *repo.ContactRepo, leads *repo.LeadRepo, wa *repo.WhatsAppRepo) *AIHandler {
	return &AIHandler{}
}

func (h *AIHandler) ScoreLead(c *fiber.Ctx) error        { return proOnly(c) }
func (h *AIHandler) ScoreContact(c *fiber.Ctx) error     { return proOnly(c) }
func (h *AIHandler) DraftReply(c *fiber.Ctx) error       { return proOnly(c) }
func (h *AIHandler) ExtractBuyerProfile(c *fiber.Ctx) error { return proOnly(c) }
func (h *AIHandler) SummarizeThread(c *fiber.Ctx) error  { return proOnly(c) }
func (h *AIHandler) DescribePropertyListing(c *fiber.Ctx) error { return proOnly(c) }

func proOnly(c *fiber.Ctx) error {
	return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
		"error":       "This feature requires a Pro plan",
		"upgrade_url": "https://masaar.io/pricing",
	})
}
