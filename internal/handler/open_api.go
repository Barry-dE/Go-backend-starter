package handler

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/labstack/echo/v4"
)

type OpenAPIHandler struct {
	Handler
}

func NewOpenAPIHandler(s *server.Server) *OpenAPIHandler {
	return &OpenAPIHandler{
		Handler: NewHandler(s),
	}

}

func (o *OpenAPIHandler) OpenAPIUI(c echo.Context) error {
	templateByte, err := os.ReadFile("static/openapi.html")

	c.Response().Header().Set("Cache-Control", "no-cache")

	if err != nil {
		return fmt.Errorf("failed to read OpenAPI template: %w ", err)
	}
	templateString := string(templateByte)

	if err := c.HTML(http.StatusOK, templateString); err != nil {
		return fmt.Errorf("failed to write HTML response")
	}

	return nil
}
