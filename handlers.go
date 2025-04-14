package main

import (
	"esp32-rgb/ui"
	"fmt"
	"github.com/a-h/templ"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

// renderControlPanel is a helper to render the ControlPanel with the correct brightness and animation selector components
func renderControlPanel(c echo.Context, currentColor, currentBrightness, currentAnimation string) error {
	var brightnessComponent templ.Component
	var animationSelector templ.Component
	if currentAnimation == "solid" {
		brightnessComponent = ui.BrightnessSolid(currentColor, currentBrightness)
		animationSelector = ui.AnimationSelectorSolid()
	} else if currentAnimation == "rainbow" {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorRainbow()
	} else {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorFade()
	}
	return Render(c, ui.Main(ui.ControlPanel(currentColor, currentBrightness, currentAnimation, brightnessComponent, animationSelector)))
}

// HomeHandler serves the main page
func HomeHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	// Initialize default values if not set
	currentColor, ok := sess.Values["color"].(string)
	if !ok {
		currentColor = "#ffffff"
		sess.Values["color"] = currentColor
	}
	currentBrightness, ok := sess.Values["brightness"].(string)
	if !ok {
		currentBrightness = "100"
		sess.Values["brightness"] = currentBrightness
	}
	currentAnimation, ok := sess.Values["animation"].(string)
	if !ok {
		currentAnimation = "solid"
		sess.Values["animation"] = currentAnimation
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	return renderControlPanel(c, currentColor, currentBrightness, currentAnimation)
}

// SetColorHandler handles color change requests
func SetColorHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	color := c.FormValue("color")
	brightness := c.FormValue("brightness")
	if color != "" && brightness != "" {
		sess.Values["color"] = color
		sess.Values["brightness"] = brightness
		sess.Values["animation"] = "solid"
		r, g, b := hexToRGB(color)
		message := fmt.Sprintf("color:%d,%d,%d,%s", r, g, b, brightness)
		if err := SendWSMessage(message); err != nil {
			return c.String(http.StatusInternalServerError, "Failed to update color")
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	return renderControlPanel(c, sess.Values["color"].(string), sess.Values["brightness"].(string), sess.Values["animation"].(string))
}

// SetBrightnessHandler handles brightness-only changes
func SetBrightnessHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	brightness := c.FormValue("brightness")
	if brightness != "" {
		sess.Values["brightness"] = brightness
		message := fmt.Sprintf("brightness:%s", brightness)
		if err := SendWSMessage(message); err != nil {
			return c.String(http.StatusInternalServerError, "Failed to update brightness")
		}
		if sess.Values["animation"].(string) == "solid" {
			r, g, b := hexToRGB(sess.Values["color"].(string))
			message := fmt.Sprintf("color:%d,%d,%d,%s", r, g, b, brightness)
			if err := SendWSMessage(message); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to update color with new brightness")
			}
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	return renderControlPanel(c, sess.Values["color"].(string), sess.Values["brightness"].(string), sess.Values["animation"].(string))
}

// SetAnimationHandler handles animation change requests
func SetAnimationHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	animation := c.FormValue("animation")
	if animation != "" {
		sess.Values["animation"] = animation
		message := fmt.Sprintf("animation:%s", animation)
		if err := SendWSMessage(message); err != nil {
			return c.String(http.StatusInternalServerError, "Failed to update animation")
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	return renderControlPanel(c, sess.Values["color"].(string), sess.Values["brightness"].(string), sess.Values["animation"].(string))
}

// hexToRGB converts hex color (#FF0000) to RGB values
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
