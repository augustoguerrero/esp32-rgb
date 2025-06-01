# ESP32 RGB LED Strip Controller

This project is a web-based application to control an ESP32-connected RGB LED strip, allowing users to adjust color, brightness, power state, and animations via a responsive UI. It uses Go for the backend, TailwindCSS for styling, Templ for server-side rendering, and WebSocket for real-time communication with the ESP32.

## Features
- **Power Toggle**: Turn the LED strip on or off.
- **Color Selection**: Choose from a grid of predefined colors (reds, greens, blues, yellows, purples, pinks, cyans).
- **Brightness Control**: Adjust brightness (0-255) via a slider; setting to 0 turns off the strip.
- **Animation Modes**: Select from Solid Color, Rainbow, Fade, Chase, or Twinkle animations.
- **Real-Time Updates**: Changes are sent instantly to the ESP32 via WebSocket.
- **Responsive UI**: Built with TailwindCSS for a clean, mobile-friendly interface.
- **Session Management**: Persists user settings (color, brightness, animation, power state) across page refreshes.

## Tech Stack
- **Backend**: Go with Echo framework for routing and session management.
- **Frontend**: Templ for server-side rendered components, TailwindCSS for styling, htmx for dynamic updates without JavaScript.
- **ESP32**: Arduino-based firmware using FastLED for LED control and WebSocket for communication.
- **WebSocket**: Real-time bidirectional communication between the Go server and ESP32.

## Prerequisites
- **Go**: Version 1.18 or higher.
- **Node.js**: For TailwindCSS compilation.
- **Arduino IDE**: To upload the ESP32 firmware.
- **ESP32 Board**: Compatible with WS2815 LED strips.
- **WiFi Network**: For ESP32 connectivity (configured with SSID and password).
- **Hardware**:
    - WS2815 LED strip (99 LEDs, configurable).
    - ESP32 development board.
    - Power supply suitable for the LED strip.

## Installation

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
   Install Node.js dependencies and build CSS:
   ```bash
   npm install
   npx tailwindcss -i ./input.css -o ./static/output.css
   ```

6. **Run the Server**:
   ```bash
   go run .
   ```
   The server starts on `http://localhost:8080`.

### ESP32 Firmware
1. **Install Arduino Libraries**:
    - `WiFi.h`
    - `WebSocketsServer` (by Markus Sattler)
    - `FastLED`
    - `ArduinoOTA`

2. **Configure `sketch_apr10a.ino`**:
    - Update `ssid` and `password` to match your WiFi credentials:
      ```cpp
      const char* ssid = "YOUR_SSID";
      const char* password = "YOUR_PASSWORD";
      ```
    - Adjust `NUM_LEDS`, `DATA_PIN`, `LED_TYPE`, and `COLOR_ORDER` if using a different LED strip:
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
    - Open the Serial Monitor (115200 baud) to confirm WiFi connection and IP address.
    - The ESP32 hosts a WebSocket server at `ws://<ESP32_IP>:81/ws`.

### Configuration
- **WebSocket Address**:
  Update `wsAddr` in `main.go` to match the ESP32's IP address:
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
   Open `http://localhost:8080` in a browser.

2. **Control the LED Strip**:
    - **Power Toggle**: Check/uncheck to turn the strip on/off (sends `brightness:0` when off).
    - **Color Grid**: Click a color button to set a solid color (resets animation to "solid").
    - **Brightness Slider**: Adjust brightness; sliding to 0 turns off the strip and unchecks the power toggle.
    - **Animation Selector**: Choose an animation mode (Solid Color, Rainbow, Fade, Chase, Twinkle).

3. **Monitor ESP32**:
   Use the Serial Monitor to debug WebSocket messages received by the ESP32.

## File Structure
- **Go Backend**:
    - `main.go`: Initializes Echo server, WebSocket connection, and static file serving.
    - `routes.go`: Defines routes (`/`, `/set-color`, `/set-brightness`, `/set-animation`, `/set-power`) and session middleware.
    - `handlers.go`: Handles HTTP requests, manages sessions, and sends WebSocket messages.
    - `render.go`: Renders Templ components with Echo.
- **Frontend**:
    - `ui/components.templ`: Templ components for UI (PowerToggle, ColorGrid, Brightness sliders, Animation selectors, ControlPanel).
    - `ui/layouts.templ`: Main layout wrapper for pages.
    - `input.css` / `static/output.css`: TailwindCSS input and compiled output.
    - `static/`: Directory for serving CSS and other static assets.
- **ESP32 Firmware**:
    - `sketch_apr10a.ino`: ESP32 code for WiFi, WebSocket, OTA updates, and LED control with FastLED.

## How It Works
- **Web Interface**:
    - The UI is rendered server-side using Templ and styled with TailwindCSS.
    - htmx handles dynamic updates by sending POST requests to `/set-*` endpoints, replacing the page content with updated HTML.
    - Session data (color, brightness, animation, power state) is stored using `gorilla/sessions`.

- **Backend**:
    - Echo routes handle requests and update session values.
    - Handlers (`HomeHandler`, `SetColorHandler`, etc.) render the `ControlPanel` component with current settings.
    - WebSocket messages are sent to the ESP32 in formats like `color:R,G,B,brightness`, `brightness:value`, or `animation:name`.

- **ESP32**:
    - Runs a WebSocket server to receive commands.
    - Parses messages to set color (`color:R,G,B,brightness`), brightness (`brightness:value`), power state (`brightness:0` for off), or animation (`animation:name`).
    - Uses FastLED to control the LED strip with animations (Solid Color, Rainbow, Fade, Chase, Twinkle).

## Troubleshooting
- **Server Fails to Start**:
    - Ensure port `8080` is free.
    - Check for missing Go dependencies (`go mod tidy`).
- **Templ Errors**:
    - Run `templ generate` and fix syntax issues in `.templ` files.
- **ESP32 Not Responding**:
    - Verify the ESP32's IP address and update `wsAddr` in `main.go`.
    - Check Serial Monitor for WiFi or WebSocket errors.
    - Ensure the ESP32 is on the same network as the server.
- **UI Not Updating**:
    - Confirm htmx attributes (`hx-post`, `hx-target`, `hx-swap`) in `components.templ`.
    - Check browser console for JavaScript errors.
- **LEDs Not Changing**:
    - Verify WebSocket messages in the ESP32 Serial Monitor.
    - Ensure `DATA_PIN` and LED configuration match your hardware.

## Known Issues
- **Brightness Slider at Minimum**:
    - Sliding to 0 turns off the strip and syncs the power toggle to "off".
    - Fixed by ensuring `SetBrightnessHandler` updates `isOn` and sends `brightness:0`.
- **Power Button**:
    - Sends `brightness:0` for off state to align with slider behavior.
    - Restores previous brightness/color/animation when turned on.

## Contributing
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature-name`).
3. Commit changes (`git commit -m "Add feature"`).
4. Push to the branch (`git push origin feature-name`).
5. Open a pull request.

## License
MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgments
- [Echo](https://echo.labstack.com/) for the web framework.
- [Templ](https://github.com/a-h/templ) for server-side rendering.
- [TailwindCSS](https://tailwindcss.com/) for styling.
- [FastLED](https://github.com/FastLED/FastLED) for LED control.
- [htmx](https://htmx.org/) for dynamic UI updates.