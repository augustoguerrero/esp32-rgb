package main

import (
	"encoding/json"
	"esp32-rgb/ui"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HomeHandler renders the editor page.
func HomeHandler(c echo.Context) error {
	return Render(c, ui.Main(ui.EditorPage(cfg.NumLEDs)))
}

// BrowserWSHandler upgrades the connection to WebSocket and relays
// engine events (frames + status) to the browser, and accepts setFrame
// commands from the browser.
func BrowserWSHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// Send initial status immediately
	statusMsg, _ := json.Marshal(engine.Status())
	_ = conn.WriteMessage(websocket.TextMessage, statusMsg)

	// Send current frame
	frame := engine.CurrentFrame()
	ledsJSON := make([][3]uint8, len(frame))
	for i, l := range frame {
		ledsJSON[i] = [3]uint8{l.R, l.G, l.B}
	}
	frameMsg, _ := json.Marshal(map[string]any{"type": "frame", "leds": ledsJSON})
	_ = conn.WriteMessage(websocket.TextMessage, frameMsg)

	// Fan out engine broadcasts to this client
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Read incoming messages from the browser
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg map[string]json.RawMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		typeRaw, ok := msg["type"]
		if !ok {
			continue
		}
		var msgType string
		_ = json.Unmarshal(typeRaw, &msgType)

		switch msgType {
		case "setFrame":
			var payload struct {
				LEDs [][3]uint8 `json:"leds"`
			}
			if err := json.Unmarshal(raw, &payload); err == nil {
				leds := jsonToRGBs(payload.LEDs, cfg.NumLEDs)
				engine.SetLive(leds)
			}
		case "ping":
			pong, _ := json.Marshal(map[string]string{"type": "pong"})
			_ = conn.WriteMessage(websocket.TextMessage, pong)
		}
	}
	<-done
	return nil
}

// ---- strip control ----------------------------------------------------------

func PowerHandler(c echo.Context) error {
	var body struct {
		Power string `json:"power"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	switch body.Power {
	case "on":
		engine.SetPower(true)
	case "off":
		engine.SetPower(false)
	default:
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "power must be 'on' or 'off'"})
	}
	return c.JSON(http.StatusOK, engine.Status())
}

func BrightnessHandler(c echo.Context) error {
	var body struct {
		Brightness int `json:"brightness"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	if body.Brightness < 0 || body.Brightness > 255 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "brightness must be 0-255"})
	}
	engine.SetBrightness(uint8(body.Brightness))
	return c.JSON(http.StatusOK, engine.Status())
}

func StopHandler(c echo.Context) error {
	engine.Stop()
	return c.JSON(http.StatusOK, engine.Status())
}

func PlayHandler(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	if err := engine.Play(id); err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, engine.Status())
}

func SetFrameHandler(c echo.Context) error {
	var body struct {
		LEDs [][3]uint8 `json:"leds"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	leds := jsonToRGBs(body.LEDs, cfg.NumLEDs)
	engine.SetLive(leds)
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

// ---- animation CRUD ---------------------------------------------------------

func ListAnimationsHandler(c echo.Context) error {
	list, err := store.ListAnimations()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if list == nil {
		list = []Animation{}
	}
	return c.JSON(http.StatusOK, list)
}

func GetAnimationHandler(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	anim, err := store.GetAnimation(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, anim)
}

func CreateAnimationHandler(c echo.Context) error {
	var body struct {
		Name   string  `json:"name"`
		FPS    int     `json:"fps"`
		Loop   bool    `json:"loop"`
		Frames []Frame `json:"frames"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	if body.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "name required"})
	}
	if body.FPS <= 0 {
		body.FPS = 30
	}

	id, err := store.CreateAnimation(body.Name, body.FPS, body.Loop)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if len(body.Frames) > 0 {
		if err := store.SetFrames(id, body.Frames); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
	}
	anim, _ := store.GetAnimation(id)
	return c.JSON(http.StatusCreated, anim)
}

func UpdateAnimationHandler(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	var body struct {
		Name   string  `json:"name"`
		FPS    int     `json:"fps"`
		Loop   bool    `json:"loop"`
		Frames []Frame `json:"frames"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	if body.FPS <= 0 {
		body.FPS = 30
	}
	if err := store.UpdateAnimation(id, body.Name, body.FPS, body.Loop); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if body.Frames != nil {
		if err := store.SetFrames(id, body.Frames); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
	}
	anim, _ := store.GetAnimation(id)
	return c.JSON(http.StatusOK, anim)
}

func DeleteAnimationHandler(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	if err := store.DeleteAnimation(id); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// ---- helpers ----------------------------------------------------------------

func jsonToRGBs(src [][3]uint8, numLEDs int) []RGB {
	leds := make([]RGB, numLEDs)
	for i := 0; i < numLEDs && i < len(src); i++ {
		leds[i] = RGB{src[i][0], src[i][1], src[i][2]}
	}
	return leds
}
