package main

import (
	"esp32-rgb/ui"
	"fmt"
	"github.com/a-h/templ"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"strings"
)

func renderControlPanel(c echo.Context, currentColor, currentBrightness, currentAnimation string, isOn bool) error {
	var brightnessComponent templ.Component
	var animationSelector templ.Component
	switch currentAnimation {
	case "solid":
		brightnessComponent = ui.BrightnessSolid(currentColor, currentBrightness)
		animationSelector = ui.AnimationSelectorSolid()
	case "rainbow":
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorRainbow()
	case "fade":
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorFade()
	case "chase":
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorChase()
	case "twinkle":
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorTwinkle()
	default:
		brightnessComponent = ui.BrightnessSolid(currentColor, currentBrightness)
		animationSelector = ui.AnimationSelectorSolid()
	}
	return Render(c, ui.Main(ui.ControlPanel(currentColor, currentBrightness, currentAnimation, brightnessComponent, animationSelector, isOn)))
}

func HomeHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	currentColor, ok := sess.Values["color"].(string)
	if !ok {
		currentColor = "#ffffff"
		sess.Values["color"] = currentColor
	}
	currentBrightness, ok := sess.Values["brightness"].(string)
	if !ok {
		currentBrightness = "120"
		sess.Values["brightness"] = currentBrightness
	}
	currentAnimation, ok := sess.Values["animation"].(string)
	if !ok {
		currentAnimation = "solid"
		sess.Values["animation"] = currentAnimation
	}
	isOn, ok := sess.Values["isOn"].(bool)
	if !ok {
		isOn = true
		sess.Values["isOn"] = isOn
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Printf("HomeHandler: color=%s, brightness=%s, animation=%s, isOn=%v", currentColor, currentBrightness, currentAnimation, isOn)
	return renderControlPanel(c, currentColor, currentBrightness, currentAnimation, isOn)
}

func SetColorHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	color := c.FormValue("color")
	brightness := c.FormValue("brightness")
	if color == "" || brightness == "" {
		log.Println("Invalid color or brightness")
		return c.String(http.StatusBadRequest, "Invalid input")
	}
	sess.Values["color"] = color
	sess.Values["brightness"] = brightness
	sess.Values["animation"] = "solid"
	// Safely handle isOn
	isOn, ok := sess.Values["isOn"].(bool)
	if !ok {
		isOn = true // Default to true if unset
		sess.Values["isOn"] = isOn
	}
	if isOn {
		r, g, b := hexToRGB(color)
		message := fmt.Sprintf("color:%d,%d,%d,%s", r, g, b, brightness)
		if err := SendWSMessage(message); err != nil {
			log.Println("WebSocket error:", err)
			return c.String(http.StatusInternalServerError, "Failed to update color")
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Printf("SetColor: color=%s, brightness=%s", color, brightness)
	return renderControlPanel(c, color, brightness, "solid", isOn)
}

func SetBrightnessHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	brightness := c.FormValue("brightness")
	if brightness == "" {
		log.Println("Invalid brightness")
		return c.String(http.StatusBadRequest, "Invalid brightness")
	}
	sess.Values["brightness"] = brightness
	isOn, ok := sess.Values["isOn"].(bool)
	if !ok {
		isOn = true
		sess.Values["isOn"] = isOn
	}
	if isOn {
		message := fmt.Sprintf("brightness:%s", brightness)
		if err := SendWSMessage(message); err != nil {
			log.Println("WebSocket error:", err)
			return c.String(http.StatusInternalServerError, "Failed to update brightness")
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Println("SetBrightness: brightness=", brightness)
	return renderControlPanel(c, sess.Values["color"].(string), brightness, sess.Values["animation"].(string), isOn)
}

func SetAnimationHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	animation := c.FormValue("animation")
	if animation == "" {
		log.Println("Invalid animation")
		return c.String(http.StatusBadRequest, "Invalid animation")
	}
	sess.Values["animation"] = animation
	isOn, ok := sess.Values["isOn"].(bool)
	if !ok {
		isOn = true
		sess.Values["isOn"] = isOn
	}
	if isOn {
		message := fmt.Sprintf("animation:%s", animation)
		if err := SendWSMessage(message); err != nil {
			log.Println("WebSocket error:", err)
			return c.String(http.StatusInternalServerError, "Failed to update animation")
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Println("SetAnimation: animation=", animation)
	return renderControlPanel(c, sess.Values["color"].(string), sess.Values["brightness"].(string), animation, isOn)
}

func SetPowerHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	power := c.FormValue("power")
	if power != "on" && power != "off" {
		log.Println("Invalid power value:", power)
		return c.String(http.StatusBadRequest, "Invalid power value")
	}
	isOn := power == "on"
	sess.Values["isOn"] = isOn
	message := fmt.Sprintf("power:%s", power)
	if err := SendWSMessage(message); err != nil {
		log.Println("WebSocket error:", err)
		return c.String(http.StatusInternalServerError, "Failed to update power state")
	}
	if isOn {
		currentAnimation := sess.Values["animation"].(string)
		currentBrightness := sess.Values["brightness"].(string)
		if currentAnimation == "solid" {
			r, g, b := hexToRGB(sess.Values["color"].(string))
			message := fmt.Sprintf("color:%d,%d,%d,%s", r, g, b, currentBrightness)
			if err := SendWSMessage(message); err != nil {
				log.Println("WebSocket error:", err)
				return c.String(http.StatusInternalServerError, "Failed to restore color")
			}
		} else {
			message := fmt.Sprintf("animation:%s", currentAnimation)
			if err := SendWSMessage(message); err != nil {
				log.Println("WebSocket error:", err)
				return c.String(http.StatusInternalServerError, "Failed to restore animation")
			}
			message = fmt.Sprintf("brightness:%s", currentBrightness)
			if err := SendWSMessage(message); err != nil {
				log.Println("WebSocket error:", err)
				return c.String(http.StatusInternalServerError, "Failed to restore brightness")
			}
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Printf("SetPower: isOn=%v, message=%s", isOn, message)
	return renderControlPanel(c, sess.Values["color"].(string), sess.Values["brightness"].(string), sess.Values["animation"].(string), isOn)
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
