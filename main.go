package main

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var wsConn *websocket.Conn
var wsAddr = "ws://192.168.1.74:81/ws" // Replace with ESP32 IP from Serial Monitor

func initWebSocket() error {
	var err error
	wsConn, _, err = websocket.DefaultDialer.Dial(wsAddr, nil)
	if err != nil {
		log.Printf("WebSocket connection failed: %v", err)
		return err
	}
	log.Println("WebSocket connected to", wsAddr)
	return nil
}

func SendWSMessage(message string) error {
	if wsConn == nil {
		if err := initWebSocket(); err != nil {
			return err
		}
	}
	log.Println("Sending WebSocket message:", message)
	err := wsConn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("WebSocket write failed: %v", err)
		wsConn = nil // Reset connection
		return err
	}
	return nil
}

func main() {
	e := echo.New()
	e.Static("/static", "static")
	RegisterRoutes(e)

	// Initialize WebSocket
	if err := initWebSocket(); err != nil {
		log.Println("Initial WebSocket connection failed, will retry on demand")
	}

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
