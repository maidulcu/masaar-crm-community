package middleware

import (
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
)

// PIISafeLogger returns a Fiber access logger that omits query parameters from
// log lines. This prevents phone numbers submitted in query strings (e.g.
// ?phone=+971501234567) from appearing in access logs in plain text.
// Request bodies are never logged by the access logger regardless.
func PIISafeLogger() fiber.Handler {
	return fiberlog.New(fiberlog.Config{
		// ${path} is path-only (no query string). ${url} would include query params.
		Format: "[${time}] ${status} ${method} ${path} ${latency} ${bytesSent}b\n",
	})
}
