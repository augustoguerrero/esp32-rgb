// WS2815-go.ino
#include <WiFi.h>
#include <WebSocketsServer.h>
#include <FastLED.h>
#include <ArduinoOTA.h>

#define DATA_PIN 18
#define NUM_LEDS 99
#define LED_TYPE WS2815
#define COLOR_ORDER GRB

CRGB leds[NUM_LEDS];
const char* ssid = "STARLINK";
const char* password = "NavegandoPorElEspacio25818";

WebSocketsServer webSocket = WebSocketsServer(81);

enum Animation {
  SOLID_COLOR,
  RAINBOW,
  FADE,
  CHASE,
  TWINKLE,
  SPACE,
  FIRE
};

Animation currentAnimation = RAINBOW;
CRGB solidColor = CRGB(255, 255, 255);
uint8_t brightness = 128;
uint8_t hue = 0;
bool isOn = true;
bool needUpdate = true; // Optimize updates for solid color

void webSocketEvent(uint8_t num, WStype_t type, uint8_t* payload, size_t length) {
  if (type == WStype_TEXT) {
    String message = String((char*)payload);
    Serial.println("Received: " + message);
    if (message.startsWith("color:")) {
      int r, g, b, bright;
      if (sscanf(message.c_str(), "color:%d,%d,%d,%d", &r, &g, &b, &bright) == 4) {
        CRGB newColor = CRGB(r, g, b);
        uint8_t newBrightness = constrain(bright, 0, 255);
        if (isOn && currentAnimation == SOLID_COLOR) {
          CRGB oldColor = solidColor;
          uint8_t oldBrightness = FastLED.getBrightness();
          int steps = 50;
          for (int step = 0; step <= steps; step++) {
            fract8 amt = (step * 255) / steps;
            CRGB currColor = blend(oldColor, newColor, amt);
            uint8_t currB = oldBrightness + ((newBrightness - oldBrightness) * step) / steps;
            FastLED.setBrightness(currB);
            fill_solid(leds, NUM_LEDS, currColor);
            FastLED.show();
            delay(5);
          }
        } else if (isOn) {
          FastLED.setBrightness(newBrightness);
          fill_solid(leds, NUM_LEDS, newColor);
          FastLED.show();
        }
        solidColor = newColor;
        brightness = newBrightness;
        currentAnimation = SOLID_COLOR;
        isOn = true;
        needUpdate = false;
        Serial.printf("Set SOLID_COLOR: R:%d G:%d B:%d Bright:%d\n", r, g, b, brightness);
      } else {
        Serial.println("Invalid color format");
      }
    } else if (message.startsWith("brightness:")) {
      int bright;
      if (sscanf(message.c_str(), "brightness:%d", &bright) == 1) {
        uint8_t newBrightness = constrain(bright, 0, 255);
        if (isOn) {
          uint8_t oldBrightness = FastLED.getBrightness();
          int steps = 50;
          for (int step = 0; step <= steps; step++) {
            uint8_t currB = oldBrightness + ((newBrightness - oldBrightness) * step) / steps;
            FastLED.setBrightness(currB);
            runCurrentAnimation();
            FastLED.show();
            delay(5);
          }
        }
        brightness = newBrightness;
        needUpdate = true;
        Serial.println("Brightness: " + String(brightness));
      } else {
        Serial.println("Invalid brightness format");
      }
    } else if (message.startsWith("power:")) {
      String state = message.substring(6);
      if (state == "on") {
        if (!isOn) {
          int steps = 50;
          for (int step = 0; step <= steps; step++) {
            uint8_t currB = (step * brightness) / steps;
            FastLED.setBrightness(currB);
            runCurrentAnimation();
            FastLED.show();
            delay(5);
          }
          FastLED.setBrightness(brightness);
          isOn = true;
          Serial.println("Power ON, brightness: " + String(brightness));
        }
      } else if (state == "off") {
        if (isOn) {
          uint8_t startB = FastLED.getBrightness();
          int steps = 50;
          for (int step = 0; step <= steps; step++) {
            uint8_t currB = startB - (step * startB) / steps;
            FastLED.setBrightness(currB);
            runCurrentAnimation();
            FastLED.show();
            delay(5);
          }
          fill_solid(leds, NUM_LEDS, CRGB::Black);
          FastLED.setBrightness(brightness);
          FastLED.show();
          isOn = false;
          Serial.println("Power OFF");
        }
      } else {
        Serial.println("Invalid power state: " + state);
      }
    } else if (message.startsWith("animation:")) {
      String anim = message.substring(10);
      Animation newAnim = SOLID_COLOR;
      if (anim == "rainbow") newAnim = RAINBOW;
      else if (anim == "fade") newAnim = FADE;
      else if (anim == "chase") newAnim = CHASE;
      else if (anim == "twinkle") newAnim = TWINKLE;
      else if (anim == "space") newAnim = SPACE;
      else if (anim == "fire") newAnim = FIRE;
      else if (anim == "solid") newAnim = SOLID_COLOR;
      if (isOn && newAnim != currentAnimation) {
        uint8_t startB = FastLED.getBrightness();
        int steps = 25;
        for (int step = 0; step <= steps; step++) {
          uint8_t currB = startB - (step * startB) / steps;
          FastLED.setBrightness(currB);
          runCurrentAnimation();
          FastLED.show();
          delay(5);
        }
        currentAnimation = newAnim;
        for (int step = 0; step <= steps; step++) {
          uint8_t currB = (step * brightness) / steps;
          FastLED.setBrightness(currB);
          runCurrentAnimation();
          FastLED.show();
          delay(5);
        }
        FastLED.setBrightness(brightness);
      } else {
        currentAnimation = newAnim;
      }
      isOn = true;
      needUpdate = true;
      Serial.println("Set animation to: " + anim);
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

void runCurrentAnimation() {
  switch (currentAnimation) {
    case SOLID_COLOR:
      fill_solid(leds, NUM_LEDS, solidColor);
      break;
    case RAINBOW:
      fill_rainbow(leds, NUM_LEDS, hue++, 10);
      break;
    case FADE:
      static uint8_t fadeValue = 0;
      static bool fadeIncreasing = true;
      fill_solid(leds, NUM_LEDS, CHSV(hue, 255, fadeValue));
      if (fadeIncreasing) {
        fadeValue += 5;
        if (fadeValue >= 255) fadeIncreasing = false;
      } else {
        fadeValue -= 5;
        if (fadeValue <= 0) {
          fadeIncreasing = true;
          hue += 30;
        }
      }
      break;
    case CHASE:
      static int chasePos = 0;
      FastLED.clear();
      for (int i = 0; i < 5; i++) {
        int ledPos = (chasePos + i) % NUM_LEDS;
        leds[ledPos] = CHSV(hue++, 255, 255);
      }
      chasePos = (chasePos + 1) % NUM_LEDS;
      break;
    case TWINKLE:
      FastLED.clear();
      for (int i = 0; i < NUM_LEDS / 10; i++) {
        int ledPos = random(NUM_LEDS);
        leds[ledPos] = CHSV(random(255), 255, 255);
      }
      break;
    case SPACE:
      FastLED.clear();
      for (int i = 0; i < NUM_LEDS / 5; i++) {
        int ledPos = random(NUM_LEDS);
        uint8_t starHue = random8(160, 255); // Blues to purples
        leds[ledPos] = CHSV(starHue, 200 + random8(55), 200 + random8(55));
      }
      break;
    case FIRE:
      static byte heat[NUM_LEDS];
      for (int i = 0; i < NUM_LEDS; i++) {
        heat[i] = qsub8(heat[i], random8(0, ((55 * 255) / NUM_LEDS) + 2));
      }
      for (int k = NUM_LEDS - 1; k >= 2; k--) {
        heat[k] = (heat[k - 1] + heat[k - 2] + heat[k - 2]) / 3;
      }
      if (random8() < 120) {
        int y = random8(7);
        heat[y] = qadd8(heat[y], random8(160, 255));
      }
      for (int j = 0; j < NUM_LEDS; j++) {
        byte colorindex = scale8(heat[j], 240);
        leds[j] = HeatColor(colorindex);
      }
      break;
  }
}

void loop() {
  ArduinoOTA.handle();
  webSocket.loop();
  if (!isOn) {
    return;
  }
  runCurrentAnimation();
  FastLED.show();
  delay(20); // Adjust delay for animation speed
}