package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func init() {
	// Configure log to write to stdout with timestamps
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime)
	// Don't use log.Lshortfile as it makes the logs harder to read in journalctl
}

// Helper function to format log messages consistently
func logf(format string, v ...interface{}) {
	// Add newline if not present
	if len(format) == 0 || format[len(format)-1] != '\n' {
		format += "\n"
	}
	fmt.Printf(format, v...)
}

type CloudinaryResource struct {
	AssetID  string `json:"asset_id"`
	PublicID string `json:"public_id"`
	Format   string `json:"format"`
	Type     string `json:"type"`
}

type CloudinaryResponse struct {
	Resources []CloudinaryResource `json:"resources"`
}

type CloudinaryNotification struct {
	NotificationType     string    `json:"notification_type"`
	Timestamp           string    `json:"timestamp,omitempty"`
	RequestID           string    `json:"request_id,omitempty"`
	AssetID            string    `json:"asset_id,omitempty"`
	PublicID           string    `json:"public_id"`
	ResourceType       string    `json:"resource_type"`
	Type              string    `json:"type"`
	Version           int64     `json:"version,omitempty"`
	Format            string    `json:"format,omitempty"`
	NotificationContext struct {
		TriggeredAt  string `json:"triggered_at"`
		TriggeredBy struct {
			Source string `json:"source"`
			ID     string `json:"id"`
		} `json:"triggered_by"`
	} `json:"notification_context"`
}

// Global cache
var (
	trackCache     CloudinaryResponse
	trackCacheMux  sync.RWMutex
	lastFetchTime  time.Time
	lastFetchMux   sync.RWMutex
)

// Fetch tracks from Cloudinary
func fetchTracks(cloudName, apiKey, apiSecret string) (*CloudinaryResponse, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET",
		"https://api.cloudinary.com/v1_1/"+cloudName+"/resources/video",
		nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("type", "upload")
	q.Add("prefix", "my-music/")
	q.Add("max_results", "100")
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth(apiKey, apiSecret)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CloudinaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update cache
func updateCache(cloudName, apiKey, apiSecret string) error {
	tracks, err := fetchTracks(cloudName, apiKey, apiSecret)
	if err != nil {
		return err
	}

	trackCacheMux.Lock()
	trackCache = *tracks
	trackCacheMux.Unlock()

	lastFetchMux.Lock()
	lastFetchTime = time.Now()
	lastFetchMux.Unlock()

	logf("Cache updated with %d tracks", len(tracks.Resources))
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		logf("Warning: .env file not found")
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		logf("Fatal: Required environment variables not found")
		os.Exit(1)
	}

	// Initial cache population
	if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
		logf("Warning: Initial cache population failed: %v", err)
	}

	mux := http.NewServeMux()

	// Add a logging middleware
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logf("REQUEST: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			next(w, r)
			logf("COMPLETED: %s %s in %v", r.Method, r.URL.Path, time.Since(start))
		}
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "https://music.nskien.com", "https://music-meta.nskien.com"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	// Tracks endpoint
	mux.HandleFunc("/api/tracks", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		trackCacheMux.RLock()
		response := trackCache
		trackCacheMux.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	// Webhook endpoint with enhanced logging
	mux.HandleFunc("/api/webhook", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		logf("DEBUG: Entering webhook handler")
		defer logf("DEBUG: Exiting webhook handler")

		if r.Method != http.MethodPost {
			logf("Webhook: Rejected %s method (only POST allowed)", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Log headers for debugging
		logf("Webhook: Headers received:")
		for name, values := range r.Header {
			logf("  %s: %v", name, values)
		}

		// Read and log raw body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logf("Webhook: Error reading body: %v", err)
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		logf("Webhook: Raw body received: %s", string(body))

		// Parse notification
		var notification CloudinaryNotification
		if err := json.Unmarshal(body, &notification); err != nil {
			logf("Webhook: Error parsing JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		logf("Webhook: Parsed notification: %+v", notification)

		// Handle different notification types
		switch notification.NotificationType {
		case "upload":
			logf("Webhook: Handling upload notification for %s", notification.PublicID)
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				logf("Webhook: Failed to update cache: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			logf("Webhook: Cache updated successfully for upload: %s", notification.PublicID)

		case "delete":
			logf("Webhook: Handling delete notification")
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				logf("Webhook: Failed to update cache: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			logf("Webhook: Cache updated successfully after delete")

		default:
			logf("Webhook: Unhandled notification type: %s", notification.NotificationType)
		}

		w.WriteHeader(http.StatusOK)
		logf("Webhook: Successfully processed request")
	}))

	// Health check endpoint
	mux.HandleFunc("/health", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		lastFetchMux.RLock()
		lastFetch := lastFetchTime
		lastFetchMux.RUnlock()

		response := struct {
			Status        string    `json:"status"`
			LastFetch    time.Time `json:"last_fetch"`
			CachedTracks int       `json:"cached_tracks"`
		}{
			Status:     "healthy",
			LastFetch:  lastFetch,
			CachedTracks: len(trackCache.Resources),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	handler := c.Handler(mux)

	// Start HTTP server on port 80
	go func() {
		logf("HTTP Server starting on port 80")
		if err := http.ListenAndServe(":80", handler); err != nil {
			logf("HTTP Server failed: %v", err)
		}
	}()

	// Start HTTPS server on port 443
	logf("HTTPS Server starting on port 443")
	if err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/music-meta.nskien.com/fullchain.pem", "/etc/letsencrypt/live/music-meta.nskien.com/privkey.pem", handler); err != nil {
		logf("Fatal: HTTPS Server failed to start: %v", err)
		os.Exit(1)
	}
} 