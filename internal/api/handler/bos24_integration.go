package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type BOS24IntegrationHandler struct{}

func NewBOS24IntegrationHandler(
	bos24Repo *repo.BOS24IntegrationRepo,
	syncService interface{},
	cfg *config.Config,
) *BOS24IntegrationHandler {
	return &BOS24IntegrationHandler{}
}

func (h *BOS24IntegrationHandler) ReceiveWebhook(c *fiber.Ctx) error  { return proOnly(c) }
func (h *BOS24IntegrationHandler) GetSettings(c *fiber.Ctx) error     { return proOnly(c) }
func (h *BOS24IntegrationHandler) UpdateSettings(c *fiber.Ctx) error  { return proOnly(c) }
func (h *BOS24IntegrationHandler) RegisterWebhook(c *fiber.Ctx) error { return proOnly(c) }
func (h *BOS24IntegrationHandler) SyncNow(c *fiber.Ctx) error         { return proOnly(c) }
