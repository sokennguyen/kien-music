package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func init() {
	// Initialize structured logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		AddSource: true,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
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

	slog.Info("cache updated", "track_count", len(tracks.Resources))
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("env file not found")
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		slog.Error("required environment variables not found")
		os.Exit(1)
	}

	// Initial cache population
	if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
		slog.Warn("initial cache population failed", "error", err)
	}

	mux := http.NewServeMux()

	// Add a logging middleware
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			slog.Info("request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			next(w, r)
			slog.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start),
			)
		}
	}

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

	// Webhook endpoint with enhanced logging
	mux.HandleFunc("/api/webhook", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("webhook handler entered")
		defer slog.Debug("webhook handler exited")

		if r.Method != http.MethodPost {
			slog.Warn("webhook rejected", 
				"method", r.Method,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Log headers for debugging
		headers := make(map[string][]string)
		for name, values := range r.Header {
			headers[name] = values
		}
		slog.Debug("webhook headers received", "headers", headers)

		// Read and log raw body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("webhook body read failed", "error", err)
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		slog.Debug("webhook body received", "body", string(body))

		// Parse notification
		var notification CloudinaryNotification
		if err := json.Unmarshal(body, &notification); err != nil {
			slog.Error("webhook json parse failed", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		slog.Info("webhook notification received", 
			"type", notification.NotificationType,
			"resource_type", notification.ResourceType,
			"public_id", notification.PublicID,
		)

		// Validate required fields
		if notification.ResourceType == "" || notification.Type == "" {
			slog.Error("webhook missing required fields",
				"resource_type", notification.ResourceType,
				"type", notification.Type,
			)
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Handle different notification types
		switch notification.NotificationType {
		case "upload":
			slog.Info("handling upload notification", "public_id", notification.PublicID)
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				slog.Error("cache update failed", "error", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			slog.Info("cache updated after upload", "public_id", notification.PublicID)

		case "delete":
			slog.Info("handling delete notification")
			if err := updateCache(cloudName, apiKey, apiSecret); err != nil {
				slog.Error("cache update failed", "error", err)
				http.Error(w, "Failed to update cache", http.StatusInternalServerError)
				return
			}
			slog.Info("cache updated after delete")

		default:
			slog.Warn("unhandled notification type", "type", notification.NotificationType)
		}

		w.WriteHeader(http.StatusOK)
		slog.Info("webhook processed successfully")
	}))

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
		port = "80"
	}

	slog.Info("server starting", "port", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
} 