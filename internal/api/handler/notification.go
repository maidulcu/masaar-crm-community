package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type NotificationHandler struct {
	notifications *repo.NotificationRepo
}

func NewNotificationHandler(notifications *repo.NotificationRepo) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

func userIDFromCtx(c *fiber.Ctx) (uuid.UUID, bool) {
	claims := middleware.ClaimsFromCtx(c)
	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(sub)
	return id, err == nil
}

// List godoc
// @Summary      List notifications
// @Description  Returns the authenticated user's notifications, newest first.
// @Tags         Notifications
// @Produce      json
// @Param        page   query     int  false  "Page (default 1)"
// @Param        limit  query     int  false  "Page size (default 20)"
// @Success      200    {array}   object{id=string,user_id=string,type=string,title=string,body=string,read=bool,data=string,created_at=string}
// @Security     BearerAuth
// @Router       /notifications [get]
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	notifications, err := h.notifications.ListByUser(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(notifications)
}

// MarkRead godoc
// @Summary      Mark notification read
// @Description  Marks a notification as read. Validates ownership against the JWT subject.
// @Tags         Notifications
// @Param        id  path  string  true  "Notification UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /notifications/{id}/read [patch]
func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.notifications.MarkRead(c.Context(), id, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// MarkAllRead godoc
// @Summary      Mark all notifications read
// @Description  Marks all unread notifications as read for the authenticated user.
// @Tags         Notifications
// @Success      204
// @Security     BearerAuth
// @Router       /notifications/read-all [patch]
func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if err := h.notifications.MarkAllRead(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
