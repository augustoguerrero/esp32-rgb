package main

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"net/http"
)

// Render renders a Templ component with Echo
func Render(c echo.Context, component templ.Component) error {
	w := c.Response().Writer
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return component.Render(c.Request().Context(), w)
}
