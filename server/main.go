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
	
	// Log startup message to ensure logging is working
	log.Println("Server initializing...")
}

// Helper function to format log messages consistently
func logf(format string, v ...interface{}) {
	// Add newline if not present
	if len(format) == 0 || format[len(format)-1] != '\n' {
		format += "\n"
	}
	// Use log.Printf instead of fmt.Printf to ensure proper journald integration
	log.Printf(format, v...)
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
	logf("Fetching tracks from Cloudinary (cloud_name: %s)", cloudName)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/video", cloudName)
	logf("Making request to: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logf("Error creating request: %v", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("type", "upload")
	q.Add("prefix", "my-music/")
	q.Add("max_results", "100")
	req.URL.RawQuery = q.Encode()
	logf("Full request URL: %s", req.URL.String())

	req.SetBasicAuth(apiKey, apiSecret)

	resp, err := client.Do(req)
	if err != nil {
		logf("Error making request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	logf("Cloudinary API response status: %s", resp.Status)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logf("Error reading response body: %v", err)
		return nil, err
	}
	logf("Response body: %s", string(body))

	var result CloudinaryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logf("Error parsing JSON response: %v", err)
		return nil, err
	}

	logf("Successfully fetched %d tracks", len(result.Resources))
	return &result, nil
}

// Update cache
func updateCache(cloudName, apiKey, apiSecret string) error {
	logf("Starting cache update...")
	tracks, err := fetchTracks(cloudName, apiKey, apiSecret)
	if err != nil {
		logf("Cache update failed: %v", err)
		return err
	}

	trackCacheMux.Lock()
	defer trackCacheMux.Unlock()
	
	logf("Previous cache had %d tracks", len(trackCache.Resources))
	trackCache = *tracks
	logf("New cache has %d tracks", len(tracks.Resources))

	lastFetchMux.Lock()
	lastFetchTime = time.Now()
	lastFetchMux.Unlock()

	logf("Cache update completed successfully")
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
			log.Printf("REQUEST: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			next(w, r)
			log.Printf("COMPLETED: %s %s in %v", r.Method, r.URL.Path, time.Since(start))
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
		log.Println("DEBUG: Entering webhook handler")
		defer log.Println("DEBUG: Exiting webhook handler")

		if r.Method != http.MethodPost {
			log.Printf("Webhook: Rejected %s method (only POST allowed)", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Log headers for debugging
		log.Println("Webhook: Headers received:")
		for name, values := range r.Header {
			log.Printf("  %s: %v", name, values)
		}

		// Read and log raw body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Webhook: Error reading body: %v", err)
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		log.Printf("Webhook: Raw body received: %s", string(body))

		// Parse notification
		var notification CloudinaryNotification
		if err := json.Unmarshal(body, &notification); err != nil {
			log.Printf("Webhook: Error parsing JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Webhook: Parsed notification: %+v", notification)

		// Check if this is a test request from curl
		if notification.PublicID == "test" {
			log.Println("Webhook: Detected test request, updating cache anyway")
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				log.Printf("Webhook: Failed to update cache for test: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			log.Println("Webhook: Cache updated successfully for test request")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Check if this is a relevant notification (resource_type should be video)
		if notification.ResourceType != "video" {
			logf("Webhook: Ignoring non-video resource: %s", notification.ResourceType)
			w.WriteHeader(http.StatusOK)
			return
		}

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
			// For any notification type, update the cache anyway
			logf("Webhook: Handling notification type: %s", notification.NotificationType)
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				logf("Webhook: Failed to update cache: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			logf("Webhook: Cache updated successfully for notification type: %s", notification.NotificationType)
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