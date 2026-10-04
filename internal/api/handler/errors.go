package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// Raw error text (SQL, driver and upstream-service messages) must never reach API clients:
// it leaks schema and infrastructure details. These helpers log the real error server-side
// and return a stable, generic message with an accurate status code.

// apiError maps an internal error to an HTTP status and a client-safe message.
func apiError(err error) (int, string) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, tenant.ErrMissing):
		return fiber.StatusUnauthorized, "unauthorized"
	case errors.Is(err, repo.ErrInvalidPhone):
		return fiber.StatusUnprocessableEntity, "invalid phone number"
	case errors.Is(err, pgx.ErrNoRows),
		errors.Is(err, repo.ErrForeignReference),
		errors.Is(err, repo.ErrUserNotFound):
		return fiber.StatusNotFound, "not found"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505": // unique_violation
			return fiber.StatusConflict, "already exists"
		case "23502", "23503", "23514", "22P02", "22007", "22003": // not-null, FK, check, bad text/date/number
			return fiber.StatusUnprocessableEntity, "invalid data"
		}
	}
	return fiber.StatusInternalServerError, "internal server error"
}

// serverError logs err and writes a generic response (see apiError for the mapping).
func serverError(c *fiber.Ctx, err error) error {
	status, msg := apiError(err)
	if status == fiber.StatusInternalServerError {
		log.Printf("handler error: %s %s: %v", c.Method(), c.Path(), err)
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// badRequest is for malformed request bodies; the parser's message is not echoed.
func badRequest(c *fiber.Ctx, err error) error {
	if err != nil {
		log.Printf("bad request: %s %s: %v", c.Method(), c.Path(), err)
	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
}

// upstreamError is for failures of an external dependency (AI, email, WhatsApp, ...).
func upstreamError(c *fiber.Ctx, err error) error {
	log.Printf("upstream error: %s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "upstream service unavailable"})
}

// safeMsg logs err and returns only the fixed message, for handlers that compose their own
// JSON body instead of using serverError.
func safeMsg(msg string, err error) string {
	log.Printf("%s: %v", msg, err)
	return msg
}

// rowError returns a client-safe description of why one imported row failed.
func rowError(err error) string {
	_, msg := apiError(err)
	return msg
}

// pageParams reads and bounds ?page and ?limit so a client cannot request an unbounded result
// set (memory/CPU exhaustion) or a negative offset.
func pageParams(c *fiber.Ctx, defaultLimit, maxLimit int) (page, limit int) {
	page = c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit = c.QueryInt("limit", defaultLimit)
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}
