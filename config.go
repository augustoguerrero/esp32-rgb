package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ESP32WsAddr   string
	SessionSecret string
	Port          string
	DBPath        string
	NumLEDs       int
}

func LoadConfig() Config {
	wsAddr := os.Getenv("ESP32_WS_ADDR")
	if wsAddr == "" {
		wsAddr = "ws://192.168.1.74:81/ws"
	}

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "change-me-in-production"
		log.Println("WARNING: SESSION_SECRET not set, using insecure default")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/animations.db"
	}

	numLEDs := 99
	if s := os.Getenv("NUM_LEDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			numLEDs = n
		}
	}

	return Config{
		ESP32WsAddr:   wsAddr,
		SessionSecret: secret,
		Port:          port,
		DBPath:        dbPath,
		NumLEDs:       numLEDs,
	}
}
