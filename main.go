package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	firmwareFile  = "firmware.bin"
	versionFile   = "version.txt"
	authTokenFile = "auth_token.txt"
)

var authToken string

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s - From: %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func logResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("Response: %d", rec.statusCode)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func calculateChecksum() (string, error) {
	file, err := os.Open(firmwareFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func isAuthorized(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != authToken {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getVersion(w http.ResponseWriter, r *http.Request) {
	version, err := os.ReadFile(versionFile)
	if err != nil {
		http.Error(w, `{"error": "Version file not found"}`, http.StatusNotFound)
		return
	}

	checksum, err := calculateChecksum()
	if err != nil {
		http.Error(w, `{"error": "Firmware file not found"}`, http.StatusNotFound)
		return
	}

	response := map[string]string{
		"version":  strings.TrimSpace(string(version)),
		"checksum": checksum,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getFirmware(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, firmwareFile)
}

func updateVersion(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	newVersion, ok := body["version"]
	if !ok {
		http.Error(w, `{"error": "Version not provided"}`, http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(versionFile, []byte(newVersion), 0644); err != nil {
		http.Error(w, `{"error": "Failed to update version"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "Version updated successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	token, err := os.ReadFile(authTokenFile)
	if err != nil {
		log.Println("Warning: Authentication token file not found. API will be unprotected!")
		authToken = ""
	} else {
		authToken = strings.TrimSpace(string(token))
	}

	mux := http.NewServeMux()
	mux.Handle("/version", isAuthorized(http.HandlerFunc(getVersion)))
	mux.Handle("/firmware", isAuthorized(http.HandlerFunc(getFirmware)))
	mux.Handle("/update_version", isAuthorized(http.HandlerFunc(updateVersion)))

	handler := logRequest(logResponse(mux))

	log.Println("Server starting on port 5000...")
	if err := http.ListenAndServe(":5000", handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
