package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

var (
	cfg    Config
	store  *Store
	engine *Engine
	hub    *Hub
)

func main() {
	cfg = LoadConfig()

	// Ensure the data directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// Open SQLite store
	var err error
	store, err = InitDB(cfg.DBPath, cfg.NumLEDs)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer store.Close()

	// Browser WebSocket hub
	hub = newHub()

	// Animation engine — sends frames to ESP32
	engine = NewEngine(store, cfg.NumLEDs, hub, func(msg string) error {
		return SendESP32(cfg.ESP32WsAddr, msg)
	})

	// Keep ESP32 connection alive
	StartESP32Reconnect(cfg.ESP32WsAddr)

	e := echo.New()
	e.HideBanner = true
	e.Static("/static", "static")
	RegisterRoutes(e, cfg)

	log.Printf("Starting server on %s", cfg.Port)
	if err := e.Start(cfg.Port); err != nil {
		log.Fatal(err)
	}
}
