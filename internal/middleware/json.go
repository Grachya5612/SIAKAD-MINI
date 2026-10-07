package middleware

import (
	"strings"

	"siakad-mini/internal/response"

	"github.com/gofiber/fiber/v2"
)

var bodyMethods = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// RequireJSON menolak request berbody yang Content-Type-nya bukan JSON.
// Status yang tepat adalah 415, bukan 400. Dipasang per grup, bukan global.
func RequireJSON(c *fiber.Ctx) error {
	if bodyMethods[c.Method()] {
		ct := c.Get(fiber.HeaderContentType)
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return response.Fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
		}
	}
	return c.Next()
}
