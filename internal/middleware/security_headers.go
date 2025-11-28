package middleware

import (
	"auction/pkg/httperror"
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func NewSecurityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := strings.TrimSpace(c.Get("User-ID"))
		userEmail := strings.TrimSpace(c.Get("User-Email"))
		authorization := strings.TrimSpace(c.Get("Authorization"))

		if userID == "" || userEmail == "" || authorization == "" {
			return unauthorized(c)
		}

		userCtx := c.UserContext()
		if userCtx == nil {
			userCtx = context.Background()
		}

		userCtx = context.WithValue(userCtx, "UserID", userID)
		userCtx = context.WithValue(userCtx, "UserEmail", userEmail)
		userCtx = context.WithValue(userCtx, "Jwt", authorization)

		c.SetUserContext(userCtx)
		return c.Next()
	}
}

func unauthorized(c *fiber.Ctx) error {
	err := httperror.Unauthorized(
		"auction.security_headers.unauthorized",
		"Security headers mismatch",
		nil,
	)

	return c.Status(err.Status).JSON(fiber.Map{
		"code":    err.Code,
		"message": err.Message,
	})
}
