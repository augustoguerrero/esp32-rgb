package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Mode describes what the engine is currently doing.
type Mode int

const (
	ModeIdle    Mode = iota // holding last frame, no playback
	ModePlaying             // playing a saved animation
	ModeLive                // browser is pushing frames directly
)

func (m Mode) String() string {
	switch m {
	case ModeIdle:
		return "idle"
	case ModePlaying:
		return "playing"
	case ModeLive:
		return "live"
	}
	return "unknown"
}

// Hub manages all connected browser WebSocket clients.
type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newHub() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

// Subscribe returns a channel that receives JSON messages broadcast by the hub.
func (h *Hub) Subscribe() chan []byte {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber.
func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

// Broadcast sends msg to all current subscribers (non-blocking, drops slow ones).
func (h *Hub) Broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Engine drives animation playback and bridges the server ↔ ESP32 ↔ browsers.
type Engine struct {
	store   *Store
	numLEDs int
	hub     *Hub

	// ESP32 WebSocket connection (managed externally via SendESP32)
	sendESP32 func(msg string) error

	mu          sync.Mutex
	mode        Mode
	current     []RGB
	brightness  uint8
	power       bool
	animationID int64
	stopCh      chan struct{}
}

// NewEngine creates an Engine. sendESP32 is called to push messages to the ESP32.
func NewEngine(store *Store, numLEDs int, hub *Hub, sendESP32 func(string) error) *Engine {
	leds := make([]RGB, numLEDs)
	return &Engine{
		store:     store,
		numLEDs:   numLEDs,
		hub:       hub,
		sendESP32: sendESP32,
		current:   leds,
		brightness: 128,
		power:     true,
	}
}

// Status returns the current engine state as a JSON-serialisable map.
func (e *Engine) Status() map[string]any {
	e.mu.Lock()
	defer e.mu.Unlock()
	return map[string]any{
		"type":        "status",
		"mode":        e.mode.String(),
		"animationId": e.animationID,
		"brightness":  e.brightness,
		"power":       e.power,
	}
}

// SetBrightness updates master brightness and notifies the ESP32.
func (e *Engine) SetBrightness(n uint8) {
	e.mu.Lock()
	e.brightness = n
	e.mu.Unlock()
	_ = e.sendESP32(fmt.Sprintf("brightness:%d", n))
	e.broadcastStatus()
}

// SetPower turns the strip on or off.
func (e *Engine) SetPower(on bool) {
	e.mu.Lock()
	e.power = on
	e.mu.Unlock()
	state := "off"
	if on {
		state = "on"
	}
	_ = e.sendESP32(fmt.Sprintf("power:%s", state))
	e.broadcastStatus()
}

// SetLive switches to live mode and immediately pushes the given frame.
func (e *Engine) SetLive(frame []RGB) {
	e.mu.Lock()
	if e.mode == ModePlaying && e.stopCh != nil {
		close(e.stopCh)
		e.stopCh = nil
	}
	e.mode = ModeLive
	copy(e.current, frame)
	bright := e.brightness
	e.mu.Unlock()

	_ = e.sendESP32(frameCmd(frame, bright))
	e.broadcastFrame(frame)
}

// Stop halts any running animation and enters idle mode.
func (e *Engine) Stop() {
	e.mu.Lock()
	if e.stopCh != nil {
		close(e.stopCh)
		e.stopCh = nil
	}
	e.mode = ModeIdle
	e.animationID = 0
	e.mu.Unlock()
	e.broadcastStatus()
}

// Play loads the animation with the given ID and starts the playback loop.
func (e *Engine) Play(animID int64) error {
	anim, err := e.store.GetAnimation(animID)
	if err != nil {
		return fmt.Errorf("load animation %d: %w", animID, err)
	}
	if len(anim.Frames) == 0 {
		return fmt.Errorf("animation %d has no frames", animID)
	}

	e.mu.Lock()
	if e.stopCh != nil {
		close(e.stopCh)
	}
	stopCh := make(chan struct{})
	e.stopCh = stopCh
	e.mode = ModePlaying
	e.animationID = animID
	e.mu.Unlock()

	e.broadcastStatus()

	go func() {
		frames := anim.Frames
		loop := anim.Loop
		idx := 0
		for {
			frame := frames[idx]
			e.mu.Lock()
			bright := e.brightness
			copy(e.current, frame.LEDs)
			e.mu.Unlock()

			_ = e.sendESP32(frameCmd(frame.LEDs, bright))
			e.broadcastFrame(frame.LEDs)

			delay := time.Duration(frame.DurationMs) * time.Millisecond
			if delay <= 0 {
				delay = 100 * time.Millisecond
			}

			select {
			case <-stopCh:
				return
			case <-time.After(delay):
			}

			idx++
			if idx >= len(frames) {
				if !loop {
					e.mu.Lock()
					e.mode = ModeIdle
					e.animationID = 0
					e.mu.Unlock()
					e.broadcastStatus()
					return
				}
				idx = 0
			}
		}
	}()
	return nil
}

// CurrentFrame returns a copy of the current LED frame.
func (e *Engine) CurrentFrame() []RGB {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]RGB, len(e.current))
	copy(cp, e.current)
	return cp
}

// frameCmd builds the `frame:<brightness>,<hex>` command string.
func frameCmd(leds []RGB, brightness uint8) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("frame:%d,", brightness))
	for _, led := range leds {
		sb.WriteString(fmt.Sprintf("%02X%02X%02X", led.R, led.G, led.B))
	}
	return sb.String()
}

// broadcastFrame sends the current frame to all browser subscribers.
func (e *Engine) broadcastFrame(leds []RGB) {
	ledsJSON := make([][3]uint8, len(leds))
	for i, l := range leds {
		ledsJSON[i] = [3]uint8{l.R, l.G, l.B}
	}
	msg, _ := json.Marshal(map[string]any{
		"type": "frame",
		"leds": ledsJSON,
	})
	e.hub.Broadcast(msg)
}

// broadcastStatus sends the current engine status to all browser subscribers.
func (e *Engine) broadcastStatus() {
	msg, _ := json.Marshal(e.Status())
	e.hub.Broadcast(msg)
}

// ---- ESP32 WebSocket client -------------------------------------------------

var (
	esp32Conn  *websocket.Conn
	esp32Mutex sync.Mutex
)

func initESP32WS(addr string) error {
	var err error
	esp32Conn, _, err = websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		log.Printf("ESP32 WS connect failed: %v", err)
		return err
	}
	log.Println("ESP32 WS connected to", addr)
	return nil
}

// SendESP32 sends a text message to the ESP32, reconnecting if necessary.
func SendESP32(addr, message string) error {
	esp32Mutex.Lock()
	defer esp32Mutex.Unlock()
	if esp32Conn == nil {
		if err := initESP32WS(addr); err != nil {
			return err
		}
	}
	if err := esp32Conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		log.Printf("ESP32 WS write failed: %v", err)
		esp32Conn.Close()
		esp32Conn = nil
		return err
	}
	return nil
}

// ESP32Connected returns whether the ESP32 WebSocket is currently connected.
func ESP32Connected() bool {
	esp32Mutex.Lock()
	defer esp32Mutex.Unlock()
	return esp32Conn != nil
}

// StartESP32Reconnect starts a background goroutine that keeps reconnecting.
func StartESP32Reconnect(addr string) {
	go func() {
		for {
			esp32Mutex.Lock()
			if esp32Conn == nil {
				initESP32WS(addr)
			}
			esp32Mutex.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()
}
