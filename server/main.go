package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

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

	log.Printf("Cache updated with %d tracks", len(tracks.Resources))
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		log.Fatal("Required environment variables not found")
	}

	// Initial cache population
	if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
		log.Printf("Warning: Initial cache population failed: %v", err)
	}

	mux := http.NewServeMux()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "https://music.nskien.com"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	// Tracks endpoint
	mux.HandleFunc("/api/tracks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		trackCacheMux.RLock()
		response := trackCache
		trackCacheMux.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Webhook endpoint with enhanced logging and validation
	mux.HandleFunc("/api/webhook", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Webhook: Received request from %s with method %s", r.RemoteAddr, r.Method)
		
		if r.Method != http.MethodPost {
			log.Printf("Webhook: Invalid method %s from %s", r.Method, r.RemoteAddr)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Log headers for debugging
		log.Printf("Webhook: Headers received:")
		for name, values := range r.Header {
			log.Printf("  %s: %v", name, values)
		}

		// Validate Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			log.Printf("Webhook: Invalid Content-Type: %s", contentType)
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
			return
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

		// Validate required fields
		if notification.ResourceType == "" || notification.Type == "" {
			log.Printf("Webhook: Missing required fields")
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Handle different notification types
		switch notification.NotificationType {
		case "upload":
			log.Printf("Webhook: Handling upload notification for %s", notification.PublicID)
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				log.Printf("Webhook: Failed to update cache: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			log.Printf("Webhook: Cache updated successfully for upload: %s", notification.PublicID)

		case "delete":
			log.Printf("Webhook: Handling delete notification")
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				log.Printf("Webhook: Failed to update cache: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			log.Printf("Webhook: Cache updated successfully after delete")

		default:
			log.Printf("Webhook: Unhandled notification type: %s", notification.NotificationType)
		}

		// Send 200 OK response
		w.WriteHeader(http.StatusOK)
		log.Printf("Webhook: Successfully processed request")
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
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
	})

	handler := c.Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"  // Changed to 80 for direct access
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
} 