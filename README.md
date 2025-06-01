# ESP32 RGB LED Strip Controller

This project is a web-based application for controlling an ESP32-connected RGB LED strip. It allows users to adjust color, brightness, power state, and animations through a responsive UI. The app leverages Go for the backend, HTMX for dynamic updates, TailwindCSS for styling, Templ for server-side rendering, and WebSocket for real-time communication with the ESP32.

## Features
- **Power Toggle**: Turn the LED strip on/off.
- **Color Selection**: Choose from a grid of predefined colors so it can be used on a touch display.
- **Brightness Control**: Adjust brightness (0-255) via a slider; minimum sets strip off.
- **Animation Modes**: Select Solid Color, Rainbow, Fade, Chase, or Twinkle.
- **Real-Time Updates**: Changes are sent instantly to the ESP32 via WebSocket.
- **Dynamic UI**: HTMX enables seamless updates without full page reloads.
- **Responsive Design**: TailwindCSS ensures a clean, mobile-friendly interface.
- **Session Persistence**: Saves user settings (color, brightness, animation, power state) across sessions.

## Tech Stack
- **Backend**: Go with Echo framework for routing and session management.
- **Frontend**: Templ for server-side rendering, HTMX for dynamic client-side interactions, TailwindCSS for styling.
- **ESP32**: Arduino-based firmware using FastLED for LED control and WebSocket for communication.
- **WebSocket**: Enables real-time bidirectional communication between the server and ESP32.

## Prerequisites
- **Go**: Version 1.18 or higher.
- **TailwindCSS**: Precompiled binary (no Node.js/npm required).
- **Arduino IDE**: For uploading ESP32 firmware.
- **ESP32 Board**: Compatible with WS2815 LED strips.
- **WiFi Network**: For ESP32 connectivity (configured with SSID and password).
- **Hardware**:
    - WS2815 LED strip (99 LEDs, configurable).
    - ESP32 development board.
    - Suitable power supply for the LED strip.

## Installation

### ESP32 Firmware
1. **Install Arduino Libraries**:
    - `WiFi.h`
    - `WebSocketsServer` (by Markus Sattler)
    - `FastLED`
    - `ArduinoOTA`

2. **Configure `sketch_apr10a.ino`**:
    - Update WiFi credentials:
      ```cpp
      const char* ssid = "YOUR_SSID";
      const char* password = "YOUR_PASSWORD";
      ```
    - Adjust LED strip settings if needed:
      ```cpp
      #define DATA_PIN 18
      #define NUM_LEDS 99
      #define LED_TYPE WS2815
      #define COLOR_ORDER GRB
      ```

3. **Upload to ESP32**:
    - Connect the ESP32 to your computer.
    - Open `sketch_apr10a.ino` in the Arduino IDE.
    - Select your ESP32 board and port, then upload the sketch.

4. **Verify Connection**:
    - Use the Serial Monitor (115200 baud) to check WiFi connection and IP address.
    - The ESP32 hosts a WebSocket server at `ws://<ESP32_IP>:81/ws`.

### Backend (Go Server)
1. **Clone the Repository**:
   ```bash
   git clone https://github.com/augustoguerrero/esp32-rgb.git
   cd esp32-rgb
   ```

2. **Install Go Dependencies**:
   ```bash
   go mod tidy
   ```

3. **Install Templ**:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```

4. **Generate Templ Components**:
   ```bash
   templ generate
   ```

5. **Compile TailwindCSS**:
   Download the precompiled TailwindCSS binary for your platform from [TailwindCSS releases](https://github.com/tailwindlabs/tailwindcss/releases). Place it in the project directory and run:
   ```bash
   ./tailwindcss -i ./input.css -o ./static/output.css
   ```
   Replace `./tailwindcss` with the appropriate binary name (e.g., `tailwindcss-linux-x64`).

6. **Run the Server**:
   ```bash
   go run .
   ```
   The server runs on `http://localhost:8080`.

### Configuration
- **WebSocket Address**:
  Update `wsAddr` in `main.go` with the ESP32's IP address:
  ```go
  var wsAddr = "ws://<ESP32_IP>:81/ws"
  ```
- **Session Secret**:
  Replace the session key in `routes.go` with a secure value:
  ```go
  e.Use(session.Middleware(sessions.NewCookieStore([]byte("your-secret-key"))))
  ```

## Usage
1. **Access the Web Interface**:
   Visit `http://localhost:8080` in a browser.

2. **Control the LED Strip**:
    - **Power Toggle**: Check/uncheck to turn the strip on/off (off sends `brightness:0`).
    - **Color Grid**: Click a color to set a solid color (sets animation to "solid").
    - **Brightness Slider**: Adjust brightness; sliding to 0 turns off the strip and unchecks the power toggle.
    - **Animation Selector**: Choose from Solid Color, Rainbow, Fade, Chase, or Twinkle.

3. **Monitor ESP32**:
   Check the Serial Monitor for WebSocket messages received by the ESP32.

## File Structure
- **Go Backend**:
    - `main.go`: Sets up Echo server, WebSocket connection, and static file serving.
    - `routes.go`: Defines routes (`/`, `/set-color`, `/set-brightness`, `/set-animation`, `/set-power`) and session middleware.
    - `handlers.go`: Processes HTTP requests, manages sessions, and sends WebSocket messages.
    - `render.go`: Renders Templ components with Echo.
- **Frontend**:
    - `ui/components.templ`: Templ components for UI (PowerToggle, ColorGrid, Brightness sliders, Animation selectors, ControlPanel).
    - `ui/layouts.templ`: Main layout wrapper.
    - `input.css` / `static/output.css`: TailwindCSS input and compiled output.
    - `static/`: Serves CSS and static assets.
- **ESP32 Firmware**:
    - `sketch_apr10a.ino`: ESP32 code for WiFi, WebSocket, OTA updates, and LED control with FastLED.

## How It Works
- **Web Interface**:
    - Templ renders server-side HTML components, styled with TailwindCSS.
    - HTMX enables dynamic updates by sending POST requests to `/set-*` endpoints, replacing page content without full reloads.
    - Session data (color, brightness, animation, power state) is stored using `gorilla/sessions`.

- **Backend**:
    - Echo routes handle requests and update session values.
    - Handlers (`HomeHandler`, `SetColorHandler`, etc.) render the `ControlPanel` component with current settings.
    - WebSocket messages are sent to the ESP32 (e.g., `color:R,G,B,brightness`, `brightness:value`, `animation:name`).

- **ESP32**:
    - Hosts a WebSocket server to receive commands.
    - Parses messages to set color, brightness, power state (`brightness:0` for off), or animation.
    - Uses FastLED to drive the LED strip with animations (Solid Color, Rainbow, Fade, Chase, Twinkle).

## Troubleshooting
- **Server Fails to Start**:
    - Ensure port `8080` is free.
    - Verify Go dependencies (`go mod tidy`).
- **Templ Errors**:
    - Run `templ generate` and fix `.templ` file syntax.
- **ESP32 Not Responding**:
    - Confirm the ESP32's IP address in `main.go`.
    - Check Serial Monitor for WiFi/WebSocket errors.
    - Ensure the ESP32 is on the same network.
- **UI Not Updating**:
    - Verify HTMX attributes (`hx-post`, `hx-target`, `hx-swap`) in `components.templ`.
    - Check browser console for errors.
- **LEDs Not Changing**:
    - Monitor WebSocket messages in the ESP32 Serial Monitor.
    - Confirm `DATA_PIN` and LED settings match hardware.


## Contributing
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature-name`).
3. Commit changes (`git commit -m "Add feature"`).
4. Push to the branch (`git push origin feature-name`).
5. Open a pull request.
