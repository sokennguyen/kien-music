package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
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
	PublicID     string `json:"public_id"`
	ResourceType string `json:"resource_type"`
	Type         string `json:"type"`
	Notification string `json:"notification_type"`
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
	webhookSecret := os.Getenv("CLOUDINARY_WEBHOOK_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" || webhookSecret == "" {
		log.Fatal("Required environment variables not found")
	}

	// Initial cache population
	if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
		log.Printf("Warning: Initial cache population failed: %v", err)
	}

	mux := http.NewServeMux()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://your-production-domain.com"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	// Tracks endpoint - returns cached data
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

	// Webhook endpoint for Cloudinary notifications
	mux.HandleFunc("/api/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify webhook signature
		timestamp := r.Header.Get("X-Cld-Timestamp")
		signature := r.Header.Get("X-Cld-Signature")
		if timestamp == "" || signature == "" {
			http.Error(w, "Missing required headers", http.StatusBadRequest)
			return
		}

		// Calculate expected signature
		payload := timestamp + webhookSecret
		h := sha1.New()
		h.Write([]byte(payload))
		expectedSignature := hex.EncodeToString(h.Sum(nil))

		if signature != expectedSignature {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Parse notification
		var notification CloudinaryNotification
		if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Only update cache for relevant changes
		if notification.ResourceType == "video" && 
		   notification.Type == "upload" && 
		   notification.Notification == "resource_created" {
			// Update cache
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				log.Printf("Error updating cache: %v", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
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
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
} 