package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var wsConn *websocket.Conn
var wsMutex sync.Mutex
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
	wsMutex.Lock()
	defer wsMutex.Unlock()
	if wsConn == nil {
		if err := initWebSocket(); err != nil {
			return err
		}
	}
	log.Println("Sending WebSocket message:", message)
	err := wsConn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("WebSocket write failed: %v", err)
		wsConn.Close()
		wsConn = nil
		return err
	}
	return nil
}

func main() {
	e := echo.New()
	e.Static("/static", "static")
	RegisterRoutes(e)

	// Initialize WebSocket with retry
	go func() {
		for {
			wsMutex.Lock()
			if wsConn == nil {
				initWebSocket()
			}
			wsMutex.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
