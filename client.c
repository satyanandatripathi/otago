#include <ESP8266WiFi.h>
#include <ESP8266HTTPClient.h>
#include <ESP8266httpUpdate.h>
#include <ArduinoJson.h>
#include <Updater.h>

// WiFi credentials
const char* ssid = "Xebec D Rocks";
const char* password = "0101011111";



// API endpoints
const char* versionUrl = "http://47.237.29.204:5000/version";
const char* firmwareUrl = "http://47.237.29.204:5000/firmware";

// Authorization token
const char* authToken = "9c4a94b98a894d97a4dbd61f734f0f32";

// Current firmware version
String currentVersion = "1.1.2";

void setup() {
  Serial.begin(115200);
  Serial.println("\nStarting ESP8266...");
  Serial.println("Current firmware version: " + currentVersion);

  // Connect to Wi-Fi
  connectToWiFi();
}

void loop() {
  static unsigned long lastCheck = 0;
  if (millis() - lastCheck >= 60000) { // Check for updates every 60 seconds
    lastCheck = millis();
    checkForUpdates();
  }
}

void connectToWiFi() {
  Serial.print("Connecting to Wi-Fi");
  WiFi.begin(ssid, password);

  while (WiFi.status() != WL_CONNECTED) {
    delay(1000);
    Serial.print(".");
  }
  Serial.println("\nConnected to Wi-Fi.");
}

void checkForUpdates() {
  if (WiFi.status() == WL_CONNECTED) {
    WiFiClient client;
    HTTPClient http;

    // Send request to check version
    http.begin(client, versionUrl);
    http.addHeader("Authorization", authToken); // Add Authorization header

    int httpCode = http.GET();

    if (httpCode == HTTP_CODE_OK) {
      String payload = http.getString();
      Serial.println("Server response: " + payload);

      // Parse JSON response
      DynamicJsonDocument doc(1024);
      DeserializationError error = deserializeJson(doc, payload);

      if (error) {
        Serial.println("Failed to parse JSON.");
        return;
      }

      String newVersion = doc["version"];
      String checksum = doc["checksum"];

      Serial.println("Current version: " + currentVersion);
      Serial.println("Latest version: " + newVersion);

      if (newVersion > currentVersion) {
        Serial.println("New firmware version available. Starting update...");
        downloadAndUpdateFirmware(checksum);
      } else {
        Serial.println("Already running the latest version.");
      }
    } else {
      Serial.printf("Failed to connect to server. HTTP code: %d\n", httpCode);
    }

    http.end();
  } else {
    Serial.println("WiFi not connected. Update cannot proceed.");
  }
}

void downloadAndUpdateFirmware(const String& checksum) {
  WiFiClient client;
  HTTPClient http;

  // Send request to download firmware
  http.begin(client, firmwareUrl);
  http.addHeader("Authorization", authToken); // Add Authorization header

  int httpCode = http.GET();
  if (httpCode == HTTP_CODE_OK) {
    int contentLength = http.getSize();
    bool canBegin = Update.begin(contentLength);

    if (canBegin) {
      Serial.println("Begin OTA update...");
      WiFiClient* stream = http.getStreamPtr();
      size_t written = Update.writeStream(*stream);

      if (written == contentLength) {
        Serial.println("Firmware written successfully.");
      } else {
        Serial.printf("Firmware written failed. Written: %d, Expected: %d\n", written, contentLength);
      }

      if (Update.end()) {
        if (Update.isFinished()) {
          Serial.println("Update successfully completed. Rebooting...");
          ESP.restart();
        } else {
          Serial.println("Update not finished.");
        }
      } else {
        Serial.printf("Update failed. Error #: %d\n", Update.getError());
      }
    } else {
      Serial.println("Not enough space to begin OTA.");
    }
  } else {
    Serial.printf("Firmware download failed. HTTP code: %d\n", httpCode);
  }

  http.end();
}
