package handler

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
)

type OpenAPIHandler struct {
	Handler
}

func NewOpenAPIHandler(h Handler) *OpenAPIHandler {
	return &OpenAPIHandler{
		Handler: h,
	}
}

func (h *OpenAPIHandler) ServeOpenAPIUI(c *fiber.Ctx) error {
	templateBytes, err := os.ReadFile("static/openapi.html")
	c.Set("Cache-Control", "no-cache")
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI UI template: %w", err)
	}

	templateString := string(templateBytes)

	if err := c.Type("html").Status(http.StatusOK).SendString(templateString); err != nil {
		return fmt.Errorf("failed to write HTML response: %w", err)
	}

	return nil
}
