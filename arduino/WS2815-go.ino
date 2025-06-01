#include <WiFi.h>
#include <WebSocketsServer.h>
#include <FastLED.h>
#include <ArduinoOTA.h>

#define DATA_PIN 18
#define NUM_LEDS 99
#define LED_TYPE WS2815
#define COLOR_ORDER GRB

CRGB leds[NUM_LEDS];
const char* ssid = "xxx";
const char* password = "xxx";

WebSocketsServer webSocket = WebSocketsServer(81);

enum Animation {
  SOLID_COLOR,
  RAINBOW,
  FADE,
  CHASE,
  TWINKLE
};

Animation currentAnimation = SOLID_COLOR;
CRGB solidColor = CRGB(255, 255, 255);
uint8_t brightness = 120;
uint8_t hue = 0;
bool isOn = true;

void webSocketEvent(uint8_t num, WStype_t type, uint8_t* payload, size_t length) {
  if (type == WStype_TEXT) {
    String message = String((char*)payload);
    Serial.println("Received: " + message);
    if (message.startsWith("color:")) {
      int r, g, b, bright;
      if (sscanf(message.c_str(), "color:%d,%d,%d,%d", &r, &g, &b, &bright) == 4) {
        solidColor = CRGB(r, g, b);
        brightness = constrain(bright, 0, 255);
        FastLED.setBrightness(brightness);
        currentAnimation = SOLID_COLOR;
        isOn = true;
        Serial.printf("Set SOLID_COLOR: R:%d G:%d B:%d Bright:%d\n", r, g, b, brightness);
      } else {
        Serial.println("Invalid color format");
      }
    } else if (message.startsWith("brightness:")) {
      int bright;
      if (sscanf(message.c_str(), "brightness:%d", &bright) == 1) {
        brightness = constrain(bright, 0, 255);
        FastLED.setBrightness(brightness);
        isOn = true;
        Serial.println("Brightness: " + String(brightness));
      } else {
        Serial.println("Invalid brightness format");
      }
    } else if (message.startsWith("power:")) {
      String state = message.substring(6);
      if (state == "on") {
        isOn = true;
        FastLED.setBrightness(brightness);
        Serial.println("Power ON, brightness: " + String(brightness));
      } else if (state == "off") {
        isOn = false;
        FastLED.setBrightness(0);
        FastLED.show();
        Serial.println("Power OFF");
      } else {
        Serial.println("Invalid power state: " + state);
      }
    } else if (message.startsWith("animation:")) {
      String anim = message.substring(10);
      if (anim == "rainbow") {
        currentAnimation = RAINBOW;
        Serial.println("Set RAINBOW");
      } else if (anim == "fade") {
        currentAnimation = FADE;
        Serial.println("Set FADE");
      } else if (anim == "chase") {
        currentAnimation = CHASE;
        Serial.println("Set CHASE");
      } else if (anim == "twinkle") {
        currentAnimation = TWINKLE;
        Serial.println("Set TWINKLE");
      } else {
        currentAnimation = SOLID_COLOR;
        Serial.println("Set SOLID_COLOR (default)");
      }
      isOn = true;
    } else {
      Serial.println("Unknown message");
    }
  } else if (type == WStype_DISCONNECTED) {
    Serial.println("Client disconnected");
  } else if (type == WStype_CONNECTED) {
    Serial.println("Client connected");
  }
}

void setup() {
  Serial.begin(115200);
  FastLED.addLeds<LED_TYPE, DATA_PIN, COLOR_ORDER>(leds, NUM_LEDS);
  FastLED.setBrightness(brightness);
  fill_solid(leds, NUM_LEDS, CRGB::White); // Initial state
  FastLED.show();

  WiFi.begin(ssid, password);
  Serial.print("Connecting to WiFi");
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.println("\nWiFi Connected: " + WiFi.localIP().toString());

  ArduinoOTA.setHostname("ESP32_LED_Strip");
  ArduinoOTA.begin();
  Serial.println("OTA Ready");

  webSocket.begin();
  webSocket.onEvent(webSocketEvent);
  Serial.println("WebSocket Server Started on port 81");
}

void rainbowAnimation() {
  fill_rainbow(leds, NUM_LEDS, hue++, 10);
  FastLED.show();
  delay(20);
}

void fadeAnimation() {
  static uint8_t value = 0;
  static bool increasing = true;
  fill_solid(leds, NUM_LEDS, CHSV(hue, 255, value));
  if (increasing) {
    value += 5;
    if (value >= 255) increasing = false;
  } else {
    value -= 5;
    if (value <= 0) {
      increasing = true;
      hue += 30;
    }
  }
  FastLED.show();
  delay(20);
}

void chaseAnimation() {
  static int pos = 0;
  FastLED.clear();
  for (int i = 0; i < 5; i++) {
    int ledPos = (pos + i) % NUM_LEDS;
    leds[ledPos] = CHSV(hue++, 255, 255);
  }
  FastLED.show();
  pos = (pos + 1) % NUM_LEDS;
  delay(50);
}

void twinkleAnimation() {
  FastLED.clear();
  for (int i = 0; i < NUM_LEDS / 10; i++) {
    int ledPos = random(NUM_LEDS);
    leds[ledPos] = CHSV(random(255), 255, 255);
  }
  FastLED.show();
  delay(100);
}

void loop() {
  ArduinoOTA.handle();
  webSocket.loop();
  if (!isOn) {
    FastLED.setBrightness(0);
    FastLED.show();
    return;
  }
  switch (currentAnimation) {
    case SOLID_COLOR:
      fill_solid(leds, NUM_LEDS, solidColor);
      FastLED.show();
      delay(20);
      break;
    case RAINBOW:
      rainbowAnimation();
      break;
    case FADE:
      fadeAnimation();
      break;
    case CHASE:
      chaseAnimation();
      break;
    case TWINKLE:
      twinkleAnimation();
      break;
  }
}