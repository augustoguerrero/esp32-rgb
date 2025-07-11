package main

import (
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up the Echo middleware and routes
func RegisterRoutes(e *echo.Echo) {
	// Add session middleware
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret-key"))))

	// Define routes
	e.GET("/", HomeHandler)
	e.POST("/set-hsv", SetHSVHandler)
	e.POST("/set-hex", SetHexHandler)
	e.POST("/set-brightness", SetBrightnessHandler)
	e.POST("/set-animation", SetAnimationHandler)
	e.POST("/set-power", SetPowerHandler)
}
