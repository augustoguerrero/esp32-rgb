package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, cfg Config) {
	// Page
	e.GET("/", HomeHandler)

	// Health
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"status":    "ok",
			"esp32":     ESP32Connected(),
			"num_leds":  cfg.NumLEDs,
		})
	})

	// Browser WebSocket (live preview + control)
	e.GET("/ws", BrowserWSHandler)

	// Strip control
	e.POST("/api/power", PowerHandler)
	e.POST("/api/brightness", BrightnessHandler)
	e.POST("/api/stop", StopHandler)
	e.POST("/api/play/:id", PlayHandler)
	e.POST("/api/frame", SetFrameHandler)

	// Animation CRUD
	e.GET("/api/animations", ListAnimationsHandler)
	e.POST("/api/animations", CreateAnimationHandler)
	e.GET("/api/animations/:id", GetAnimationHandler)
	e.PUT("/api/animations/:id", UpdateAnimationHandler)
	e.DELETE("/api/animations/:id", DeleteAnimationHandler)
}
