package middleware

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ValidateURLFields rejects write requests whose JSON body contains a URL field with an
// unsafe scheme. Any key named "url" or ending in "_url"/"_urls" (case-insensitive) must be
// empty or an absolute http(s) URL, or a same-site path ("/uploads/..."). This stops values
// such as `javascript:...` or `data:text/html,...` from being stored and later rendered as a
// link or image by the web app (stored XSS), or fetched server-side.
func ValidateURLFields() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch:
		default:
			return c.Next()
		}
		if !strings.HasPrefix(strings.ToLower(c.Get(fiber.HeaderContentType)), fiber.MIMEApplicationJSON) {
			return c.Next()
		}
		body := c.Body()
		if len(bytes.TrimSpace(body)) == 0 {
			return c.Next()
		}
		var doc any
		if err := json.Unmarshal(body, &doc); err != nil {
			return c.Next() // not valid JSON: the handler's own parser reports it
		}
		if field := findUnsafeURL(doc, ""); field != "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": "field " + field + " must be an http(s) URL",
			})
		}
		return c.Next()
	}
}

func isURLKey(key string) bool {
	k := strings.ToLower(key)
	return k == "url" || k == "urls" || strings.HasSuffix(k, "_url") || strings.HasSuffix(k, "_urls")
}

// findUnsafeURL walks the decoded JSON and returns the name of the first offending field.
func findUnsafeURL(v any, key string) string {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if f := findUnsafeURL(child, k); f != "" {
				return f
			}
		}
	case []any:
		for _, child := range t {
			if f := findUnsafeURL(child, key); f != "" { // array elements inherit the key (image_urls)
				return f
			}
		}
	case string:
		if key != "" && isURLKey(key) && !SafeURL(t) {
			return key
		}
	}
	return ""
}

// SafeURL reports whether s is empty, an absolute http(s) URL, or a same-site absolute path.
func SafeURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	// Control characters can be used to smuggle a scheme past naive checks ("java\tscript:").
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	if strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//") && !strings.HasPrefix(s, `/\`) {
		return true
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	}
	return false
}
