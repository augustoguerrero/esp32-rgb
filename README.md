# ESP32 RGB LED Strip Editor
<img width="1878" height="424" alt="image" src="https://github.com/user-attachments/assets/569e38da-2fbf-488c-a566-5df7c7d27dd3" />

A web-based animation editor for controlling a WS2815 LED strip via an ESP32. All animation logic runs on the Go server — the ESP32 acts as a dumb pixel renderer that displays whatever the server sends.

## Architecture

```
Browser ←── WS /ws ──→ Go Server ←── WS ──→ ESP32
   (paint, timeline,    (engine,     (FastLED,
    live preview)        SQLite)      frame render)
```

- **Go server** is the brain: manages animation state, runs playback loops, stores custom animations in SQLite, and streams frames to the ESP32 at up to 30fps.
- **ESP32** just calls `FastLED.show()` on whatever RGB frame it receives. No internal animation logic.
- **Browser** connects over WebSocket for live preview and sends paint/frame commands directly.

## Features

- **Animation editor** — paint individual LEDs by clicking/dragging on a 99-LED strip visualizer
- **4 paint tools** — Paint, Fill All, Gradient, Clear
- **HSV color picker** — hue/saturation/value sliders + hex input
- **Frame timeline** — build multi-frame animations with per-frame durations, mini-canvas thumbnails
- **Live preview** — every edit is pushed to the real strip in real time over WebSocket
- **Local playback** — preview animations in the browser at configurable FPS with loop support
- **Save / load** — animations persisted to SQLite; load them back into the editor or play them directly on the strip
- **Power & brightness** — global controls in the header, sent to ESP32 immediately
- **OTA firmware updates** — ESP32 runs ArduinoOTA; hostname `ESP32_LED_Strip`

## Tech Stack

- **Backend**: Go 1.25, Echo v4, Templ, gorilla/websocket, modernc.org/sqlite (CGO-free)
- **Frontend**: Vanilla JS, TailwindCSS (CDN), no framework
- **ESP32**: Arduino, FastLED, WebSocketsServer, ArduinoOTA

## Hardware

- ESP32 development board
- WS2815 LED strip (default: 99 LEDs, configurable)
- 12V power supply for the strip
- Data pin: GPIO 18

## Quick Start

### 1. Flash ESP32 firmware

Open `arduino/WS2815-go.ino` in the Arduino IDE. Update the WiFi credentials:

```cpp
const char* ssid     = "YOUR_SSID";
const char* password = "YOUR_PASSWORD";
```

Install the required libraries (Library Manager):
- `WebSocketsServer` by Markus Sattler
- `FastLED`
- `ArduinoOTA`

Upload via USB. After the first flash, subsequent updates can be done via OTA (hostname: `ESP32_LED_Strip`, port 3232).

### 2. Run the server

**Local:**
```bash
cp .env.example .env   # edit ESP32_WS_ADDR to match your ESP32's IP
make run               # runs templ generate + go build + ./bin/server
```

**Docker:**
```bash
cp .env.example .env
docker compose up --build
```

Open `http://localhost:8080`.

## Configuration

All settings are env vars (see `.env.example`):

| Variable | Default | Description |
|---|---|---|
| `ESP32_WS_ADDR` | `ws://192.168.1.74:81/ws` | ESP32 WebSocket URL |
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `data/animations.db` | SQLite database path |
| `NUM_LEDS` | `99` | Number of LEDs on the strip |
| `SESSION_SECRET` | `change-me-in-production` | Cookie signing key |

## ESP32 Command Protocol

The server sends text commands to the ESP32 over WebSocket:

| Command | Description |
|---|---|
| `frame:<brightness>,<hex>` | Set all LEDs — hex is `NUM_LEDS × 6` chars (`RRGGBB` per LED) |
| `pixel:<idx>,<R>,<G>,<B>` | Set a single LED (0-indexed) |
| `brightness:<0-255>` | Change master brightness |
| `power:on` / `power:off` | Fade in / fade out |

## REST API

| Method | Path | Body | Description |
|---|---|---|---|
| `POST` | `/api/power` | `{"power":"on"\|"off"}` | Toggle strip power |
| `POST` | `/api/brightness` | `{"brightness":0-255}` | Set brightness |
| `POST` | `/api/stop` | — | Stop playback |
| `POST` | `/api/play/:id` | — | Play saved animation |
| `POST` | `/api/frame` | `{"leds":[[r,g,b],...]}` | Push a static frame |
| `GET` | `/api/animations` | — | List all saved animations |
| `POST` | `/api/animations` | `{name, fps, loop, frames:[...]}` | Create animation |
| `GET` | `/api/animations/:id` | — | Get animation with frames |
| `PUT` | `/api/animations/:id` | `{name, fps, loop, frames:[...]}` | Update animation |
| `DELETE` | `/api/animations/:id` | — | Delete animation |

WebSocket: `GET /ws` — browser connects here for live frame streaming and control.

## File Structure

```
├── arduino/
│   └── WS2815-go.ino       ESP32 firmware (dumb pixel renderer)
├── ui/
│   ├── layouts.templ        HTML shell with editor.js
│   └── components.templ     EditorPage component
├── static/
│   ├── css/output.css       Compiled TailwindCSS
│   └── js/editor.js         Full editor UI (vanilla JS)
├── data/                    SQLite database (git-ignored)
├── config.go                Env var loading
├── store.go                 SQLite animation persistence
├── engine.go                Animation engine + ESP32 WS client + browser WS hub
├── main.go                  Server entry point
├── routes.go                Route registration
├── handlers.go              HTTP + WebSocket handlers
├── render.go                Templ render helper
├── Makefile
├── Dockerfile
└── docker-compose.yml
```

## Makefile Targets

```bash
make build        # templ generate + go build
make run          # build + run locally
make docker-build # build Docker image
make docker-up    # docker compose up --build
make docker-down  # docker compose down
```

## Troubleshooting

**ESP32 not connecting**
- Check `ESP32_WS_ADDR` — confirm the IP with the Arduino Serial Monitor (115200 baud)
- Ensure the ESP32 and the machine running the server are on the same network

**Build errors**
- Run `templ generate` before `go build` (or just use `make build`)
- Run `go mod tidy` if dependencies are missing

**LEDs not updating**
- Check `/health` endpoint — `esp32: true` means the server is connected
- Watch the ESP32 Serial Monitor for incoming `frame:` commands

**OTA upload fails**
- Confirm the ESP32 is running and reachable at port 3232
- The OTA hostname is `ESP32_LED_Strip` (mDNS: `ESP32_LED_Strip.local`)
