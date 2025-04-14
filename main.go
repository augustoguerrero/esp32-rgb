package main

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var wsConn *websocket.Conn
var wsAddr = "ws://192.168.1.74:81/ws"

// initWebSocket initializes the WebSocket connection to the ESP32
func initWebSocket() {
	var err error
	wsConn, _, err = websocket.DefaultDialer.Dial(wsAddr, nil)
	if err != nil {
		log.Printf("WebSocket connection failed: %v", err)
	}
}

// SendWSMessage sends a WebSocket message to the ESP32
func SendWSMessage(message string) error {
	if wsConn == nil {
		initWebSocket()
	}
	return wsConn.WriteMessage(websocket.TextMessage, []byte(message))
}

func main() {
	e := echo.New()

	// Serve static files (TailwindCSS)
	e.Static("/static", "static")

	// Register routes
	RegisterRoutes(e)

	// Start server
	log.Println("Server starting on :8080")
	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
