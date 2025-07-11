package main

import (
	"esp32-rgb/ui"
	"fmt"
	"github.com/a-h/templ"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func renderControlPanel(c echo.Context, currentBrightness, currentAnimation string, isOn bool) error {
	var brightnessComponent templ.Component
	var animationSelector templ.Component
	var showColorPicker bool
	hueStr, satStr, valStr, currentHex, satLeft, satRight, valRight := "", "", "", "", "", "", ""
	sess, _ := session.Get("session", c) // Assume no error for simplicity

	currentHue, ok := sess.Values["hue"].(int)
	if !ok {
		currentHue = 0
	}
	currentSat, ok := sess.Values["sat"].(int)
	if !ok {
		currentSat = 0
	}
	currentVal, ok := sess.Values["val"].(int)
	if !ok {
		currentVal = 100
	}

	if currentAnimation == "solid" {
		showColorPicker = true
		hueStr = strconv.Itoa(currentHue)
		satStr = strconv.Itoa(currentSat)
		valStr = strconv.Itoa(currentVal)
		r, g, b := hsvToRGB(float64(currentHue), float64(currentSat), float64(currentVal))
		currentHex = rgbToHex(r, g, b)

		// Sat left: s=0
		gr, gg, gb := hsvToRGB(float64(currentHue), 0, float64(currentVal))
		satLeft = rgbToHex(gr, gg, gb)
		// Sat right: s=100
		sr, sg, sb := hsvToRGB(float64(currentHue), 100, float64(currentVal))
		satRight = rgbToHex(sr, sg, sb)
		// Val right: v=100
		vr, vg, vb := hsvToRGB(float64(currentHue), float64(currentSat), 100)
		valRight = rgbToHex(vr, vg, vb)

		brightnessComponent = ui.BrightnessSolid(currentBrightness)
		animationSelector = ui.AnimationSelectorSolid()
	} else if currentAnimation == "rainbow" {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorRainbow()
	} else if currentAnimation == "fade" {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorFade()
	} else if currentAnimation == "chase" {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorChase()
	} else if currentAnimation == "twinkle" {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorTwinkle()
	} else if currentAnimation == "space" {
		brightnessComponent = ui.BrightnessAnim(currentBrightness)
		animationSelector = ui.AnimationSelectorSpace()
	}
	return Render(c, ui.Main(ui.ControlPanel(currentBrightness, currentAnimation, brightnessComponent, animationSelector, isOn, showColorPicker, hueStr, satStr, valStr, currentHex, satLeft, satRight, valRight)))
}

func HomeHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	_, ok := sess.Values["hue"]
	if !ok {
		sess.Values["hue"] = 0
		sess.Values["sat"] = 0
		sess.Values["val"] = 100
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
	log.Printf("HomeHandler: hue=%d, sat=%d, val=%d, brightness=%s, animation=%s, isOn=%v", sess.Values["hue"], sess.Values["sat"], sess.Values["val"], currentBrightness, currentAnimation, isOn)
	return renderControlPanel(c, currentBrightness, currentAnimation, isOn)
}

func SetHSVHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	hueStr := c.FormValue("hue")
	satStr := c.FormValue("sat")
	valStr := c.FormValue("val")
	if hueStr != "" {
		hue, _ := strconv.Atoi(hueStr)
		sess.Values["hue"] = hue
	}
	if satStr != "" {
		sat, _ := strconv.Atoi(satStr)
		sess.Values["sat"] = sat
	}
	if valStr != "" {
		val, _ := strconv.Atoi(valStr)
		sess.Values["val"] = val
	}
	currentHue := sess.Values["hue"].(int)
	currentSat := sess.Values["sat"].(int)
	currentVal := sess.Values["val"].(int)
	currentAnimation := sess.Values["animation"].(string)
	isOn, _ := sess.Values["isOn"].(bool)
	if isOn && currentAnimation == "solid" {
		r, g, b := hsvToRGB(float64(currentHue), float64(currentSat), float64(currentVal))
		brightness := sess.Values["brightness"].(string)
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
	return renderControlPanel(c, sess.Values["brightness"].(string), currentAnimation, isOn)
}

func SetHexHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Session error:", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	hex := c.FormValue("hex")
	if hex == "" || len(hex) != 7 || !strings.HasPrefix(hex, "#") {
		return c.String(http.StatusBadRequest, "Invalid hex code")
	}
	r, g, b := hexToRGB(hex)
	h, s, v := rgbToHSV(r, g, b)
	sess.Values["hue"] = int(h)
	sess.Values["sat"] = int(s)
	sess.Values["val"] = int(v)
	currentAnimation := sess.Values["animation"].(string)
	isOn, _ := sess.Values["isOn"].(bool)
	if isOn && currentAnimation == "solid" {
		brightness := sess.Values["brightness"].(string)
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
	return renderControlPanel(c, sess.Values["brightness"].(string), currentAnimation, isOn)
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
	currentAnimation, _ := sess.Values["animation"].(string)
	if isOn {
		if currentAnimation == "solid" {
			currentHue := sess.Values["hue"].(int)
			currentSat := sess.Values["sat"].(int)
			currentVal := sess.Values["val"].(int)
			r, g, b := hsvToRGB(float64(currentHue), float64(currentSat), float64(currentVal))
			message := fmt.Sprintf("color:%d,%d,%d,%s", r, g, b, brightness)
			if err := SendWSMessage(message); err != nil {
				log.Println("WebSocket error:", err)
				return c.String(http.StatusInternalServerError, "Failed to update brightness")
			}
		} else {
			message := fmt.Sprintf("brightness:%s", brightness)
			if err := SendWSMessage(message); err != nil {
				log.Println("WebSocket error:", err)
				return c.String(http.StatusInternalServerError, "Failed to update brightness")
			}
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Println("SetBrightness: brightness=", brightness)
	return renderControlPanel(c, brightness, currentAnimation, isOn)
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
		if animation == "solid" {
			currentHue := sess.Values["hue"].(int)
			currentSat := sess.Values["sat"].(int)
			currentVal := sess.Values["val"].(int)
			r, g, b := hsvToRGB(float64(currentHue), float64(currentSat), float64(currentVal))
			brightness := sess.Values["brightness"].(string)
			message = fmt.Sprintf("color:%d,%d,%d,%s", r, g, b, brightness)
			if err := SendWSMessage(message); err != nil {
				log.Println("WebSocket error:", err)
				return c.String(http.StatusInternalServerError, "Failed to update color on animation change")
			}
		}
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("Session save error:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Println("SetAnimation: animation=", animation)
	return renderControlPanel(c, sess.Values["brightness"].(string), animation, isOn)
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
			currentHue := sess.Values["hue"].(int)
			currentSat := sess.Values["sat"].(int)
			currentVal := sess.Values["val"].(int)
			r, g, b := hsvToRGB(float64(currentHue), float64(currentSat), float64(currentVal))
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
	log.Printf("SetPower: isOn=%v", isOn)
	return renderControlPanel(c, sess.Values["brightness"].(string), sess.Values["animation"].(string), isOn)
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func rgbToHex(r, g, b int) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func hsvToRGB(h, s, v float64) (int, int, int) {
	s /= 100
	v /= 100
	if s == 0 {
		rr := int(v * 255)
		return rr, rr, rr
	}
	h /= 60
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	var rr, gg, bb float64
	switch int(i) % 6 {
	case 0:
		rr, gg, bb = v, t, p
	case 1:
		rr, gg, bb = q, v, p
	case 2:
		rr, gg, bb = p, v, t
	case 3:
		rr, gg, bb = p, q, v
	case 4:
		rr, gg, bb = t, p, v
	case 5:
		rr, gg, bb = v, p, q
	}
	return int(rr * 255), int(gg * 255), int(bb * 255)
}

func rgbToHSV(r, g, b int) (float64, float64, float64) {
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	maxVal := math.Max(rf, math.Max(gf, bf))
	minVal := math.Min(rf, math.Min(gf, bf))
	delta := maxVal - minVal
	var h float64
	if delta == 0 {
		h = 0
	} else if maxVal == rf {
		h = (gf-bf)/delta + 0
		if gf < bf {
			h += 6
		}
	} else if maxVal == gf {
		h = (bf-rf)/delta + 2
	} else {
		h = (rf-gf)/delta + 4
	}
	h *= 60
	var ss float64
	if maxVal != 0 {
		ss = delta / maxVal
	}
	vv := maxVal
	return h, ss * 100, vv * 100
}
