// WS2815-go.ino — Dumb pixel renderer
// All animation logic lives on the Go server.
// This firmware just renders whatever it receives.
//
// Commands (text over WebSocket):
//   frame:<brightness>,<hex>   — set all LEDs (brightness 0-255, hex = NUM_LEDS*6 chars RRGGBB each)
//   pixel:<idx>,<R>,<G>,<B>   — set one LED (0-indexed)
//   brightness:<0-255>         — change master brightness, re-show current frame
//   power:on | power:off       — fade in/out

#include <WiFi.h>
#include <WebSocketsServer.h>
#include <FastLED.h>
#include <ArduinoOTA.h>

#define DATA_PIN    18
#define NUM_LEDS    99
#define LED_TYPE    WS2815
#define COLOR_ORDER GRB

CRGB leds[NUM_LEDS];

const char* ssid     = "STARLINK";
const char* password = "NavegandoPorElEspacio25818";

WebSocketsServer webSocket = WebSocketsServer(81);

uint8_t brightness = 128;
bool    isOn       = true;

// ---- helpers ----------------------------------------------------------------

static uint8_t hexNibble(char c) {
  if (c >= '0' && c <= '9') return c - '0';
  if (c >= 'a' && c <= 'f') return c - 'a' + 10;
  if (c >= 'A' && c <= 'F') return c - 'A' + 10;
  return 0;
}

static uint8_t hexByte(const char* s, int offset) {
  return (hexNibble(s[offset]) << 4) | hexNibble(s[offset + 1]);
}

static void fadeTo(uint8_t target, int steps = 40) {
  uint8_t start = FastLED.getBrightness();
  for (int i = 0; i <= steps; i++) {
    FastLED.setBrightness(start + ((int)(target - start) * i) / steps);
    FastLED.show();
    delay(5);
  }
}

// ---- command handlers -------------------------------------------------------

void handleFrame(const char* payload) {
  // frame:<brightness>,<hex>
  int bright;
  if (sscanf(payload + 6, "%d,", &bright) != 1) {
    Serial.println("frame: bad brightness");
    return;
  }
  // find the comma separator
  const char* comma = strchr(payload + 6, ',');
  if (!comma) { Serial.println("frame: no comma"); return; }
  const char* hex = comma + 1;

  int expectedLen = NUM_LEDS * 6;
  if ((int)strlen(hex) < expectedLen) {
    Serial.printf("frame: hex too short (%d < %d)\n", (int)strlen(hex), expectedLen);
    return;
  }

  for (int i = 0; i < NUM_LEDS; i++) {
    int o = i * 6;
    leds[i] = CRGB(hexByte(hex, o), hexByte(hex, o + 2), hexByte(hex, o + 4));
  }

  uint8_t newBright = (uint8_t)constrain(bright, 0, 255);
  brightness = newBright;
  FastLED.setBrightness(isOn ? brightness : 0);
  FastLED.show();
}

void handlePixel(const char* payload) {
  // pixel:<idx>,<R>,<G>,<B>
  int idx, r, g, b;
  if (sscanf(payload + 6, "%d,%d,%d,%d", &idx, &r, &g, &b) != 4) {
    Serial.println("pixel: bad format");
    return;
  }
  if (idx < 0 || idx >= NUM_LEDS) return;
  leds[idx] = CRGB(r, g, b);
  FastLED.show();
}

void handleBrightness(const char* payload) {
  // brightness:<n>
  int bright;
  if (sscanf(payload + 11, "%d", &bright) != 1) return;
  brightness = (uint8_t)constrain(bright, 0, 255);
  if (isOn) FastLED.setBrightness(brightness);
  FastLED.show();
  Serial.printf("brightness: %d\n", brightness);
}

void handlePower(const char* payload) {
  // power:on | power:off
  const char* state = payload + 6;
  if (strncmp(state, "on", 2) == 0 && !isOn) {
    fadeTo(brightness);
    isOn = true;
    Serial.println("power: on");
  } else if (strncmp(state, "off", 3) == 0 && isOn) {
    fadeTo(0);
    isOn = false;
    Serial.println("power: off");
  }
}

// ---- WebSocket event --------------------------------------------------------

void webSocketEvent(uint8_t num, WStype_t type, uint8_t* payload, size_t length) {
  if (type == WStype_TEXT) {
    const char* msg = (const char*)payload;
    Serial.printf("ws[%u]: %s\n", num, msg);

    if      (strncmp(msg, "frame:",      6)  == 0) handleFrame(msg);
    else if (strncmp(msg, "pixel:",      6)  == 0) handlePixel(msg);
    else if (strncmp(msg, "brightness:", 11) == 0) handleBrightness(msg);
    else if (strncmp(msg, "power:",      6)  == 0) handlePower(msg);
    else Serial.println("unknown command");

  } else if (type == WStype_CONNECTED) {
    Serial.printf("ws[%u]: client connected\n", num);
  } else if (type == WStype_DISCONNECTED) {
    Serial.printf("ws[%u]: client disconnected\n", num);
  }
}

// ---- setup / loop -----------------------------------------------------------

void setup() {
  Serial.begin(115200);
  FastLED.addLeds<LED_TYPE, DATA_PIN, COLOR_ORDER>(leds, NUM_LEDS);
  FastLED.setBrightness(brightness);
  fill_solid(leds, NUM_LEDS, CRGB::Black);
  FastLED.show();

  WiFi.begin(ssid, password);
  Serial.print("Connecting to WiFi");
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.println("\nWiFi connected: " + WiFi.localIP().toString());

  ArduinoOTA.setHostname("ESP32_LED_Strip");
  ArduinoOTA.begin();
  Serial.println("OTA ready");

  webSocket.begin();
  webSocket.onEvent(webSocketEvent);
  Serial.println("WebSocket server started on port 81");
}

void loop() {
  ArduinoOTA.handle();
  webSocket.loop();
  // No animation loop — just process incoming commands
}
