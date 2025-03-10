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
)

func init() {
	// Set up multi-writer for logging to both stdout and file
	logFile, err := os.OpenFile("/var/log/music-server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening log file: %v", err)
		os.Exit(1)
	}
	
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.Ldate | log.Ltime)
	
	// Log startup message to ensure logging is working
	log.Println("Server initializing with multi-writer logging...")
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
	log.Printf("=== Fetching tracks from Cloudinary ===")
	log.Printf("Cloud Name: %s", cloudName)
	log.Printf("API Key: %s", apiKey)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/video", cloudName)
	log.Printf("Making request to: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	q := req.URL.Query()
	q.Add("type", "upload")
	q.Add("prefix", "my-music/")
	q.Add("max_results", "100")
	req.URL.RawQuery = q.Encode()
	log.Printf("Full request URL: %s", req.URL.String())

	req.SetBasicAuth(apiKey, apiSecret)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Cloudinary API response status: %s", resp.Status)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %v", err)
	}
	log.Printf("Response body: %s", string(body))

	var result CloudinaryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error parsing JSON response: %v", err)
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	log.Printf("Successfully fetched %d tracks", len(result.Resources))
	for i, track := range result.Resources {
		log.Printf("Track %d: %s (format: %s)", i+1, track.PublicID, track.Format)
	}
	log.Printf("=== Fetch complete ===")
	
	return &result, nil
}

// Update cache
func updateCache(cloudName, apiKey, apiSecret string) error {
	log.Println("Starting cache update...")
	tracks, err := fetchTracks(cloudName, apiKey, apiSecret)
	if err != nil {
		log.Printf("Cache update failed: %v", err)
		return err
	}

	trackCacheMux.Lock()
	defer trackCacheMux.Unlock()
	
	log.Printf("Previous cache had %d tracks", len(trackCache.Resources))
	trackCache = *tracks
	log.Printf("New cache has %d tracks", len(tracks.Resources))

	lastFetchMux.Lock()
	lastFetchTime = time.Now()
	lastFetchMux.Unlock()

	log.Println("Cache update completed successfully")
	return nil
}

func main() {
	// Set up logging to stdout
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime)
	
	// Log startup
	log.Println("=== Server starting up ===")

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	log.Printf("=== Cloudinary config ===")
	log.Printf("Cloud Name: %s", cloudName)
	log.Printf("API Key: %s", apiKey)
	log.Printf("API Secret: %s", apiSecret[:5] + "...")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		log.Println("Fatal: Required environment variables not found")
		os.Exit(1)
	}

	// Initial cache population
	log.Println("=== Initial cache population ===")
	if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
		log.Printf("Warning: Initial cache population failed: %v", err)
	}

	mux := http.NewServeMux()

	// Add a logging middleware
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			log.Printf("=== REQUEST START: %s %s from %s ===", r.Method, r.URL.Path, r.RemoteAddr)
			next(w, r)
			log.Printf("=== REQUEST END: %s %s in %v ===", r.Method, r.URL.Path, time.Since(start))
		}
	}

	// Tracks endpoint
	mux.HandleFunc("/api/tracks", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("=== Tracks endpoint hit ===")
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		trackCacheMux.RLock()
		response := trackCache
		trackCacheMux.RUnlock()

		log.Printf("Returning %d tracks from cache", len(response.Resources))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	// Simple test endpoint
	mux.HandleFunc("/test", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("=== Test endpoint hit ===")
		w.Write([]byte("Test endpoint working"))
	}))

	// Webhook endpoint with enhanced logging
	mux.HandleFunc("/api/webhook", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("=== Webhook endpoint hit ===")
		log.Println("DEBUG: =====================")
		log.Println("DEBUG: Webhook handler start")
		log.Println("DEBUG: =====================")
		defer log.Println("DEBUG: Webhook handler end")

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
			log.Printf("Webhook: Error parsing notification: %v", err)
			http.Error(w, "Invalid notification format", http.StatusBadRequest)
			return
		}
		log.Printf("Webhook: Parsed notification: %+v", notification)

		// Always update cache for any webhook call (upload, delete, etc.)
		log.Printf("Webhook: Received %s notification for public_id: %s", notification.NotificationType, notification.PublicID)
		log.Println("Webhook: Starting cache update...")
		if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
			log.Printf("Webhook: Failed to update cache: %v", err)
			http.Error(w, "Failed to update cache", http.StatusInternalServerError)
			return
		}
		log.Println("Webhook: Cache update completed successfully")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"status": "success", "message": "Webhook processed successfully"}
		json.NewEncoder(w).Encode(response)
		log.Println("Webhook: Response sent successfully")
	}))

	// Health check endpoint
	mux.HandleFunc("/health", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("=== Health endpoint hit ===")
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

	// Start HTTP server on port 80
	log.Println("=== HTTP Server starting on port 80 ===")
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Printf("HTTP Server failed: %v", err)
		os.Exit(1)
	}
} 