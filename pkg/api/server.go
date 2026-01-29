package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zarz/spotiflac-bot/pkg/backend"
)

// Server represents the API server
type Server struct {
	port string
}

// NewServer creates a new API server
func NewServer(port string) *Server {
	return &Server{port: port}
}

// APIResponse is a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, APIResponse{Success: false, Error: message})
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Start starts the API server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.healthHandler)

	// Search endpoints
	mux.HandleFunc("/api/search", s.searchHandler)

	// Metadata endpoints
	mux.HandleFunc("/api/metadata", s.metadataHandler)
	mux.HandleFunc("/api/parse-url", s.parseURLHandler)

	// Availability check
	mux.HandleFunc("/api/availability", s.availabilityHandler)

	// Download endpoints
	mux.HandleFunc("/api/download", s.downloadHandler)
	mux.HandleFunc("/api/progress", s.progressHandler)

	// File download endpoint
	mux.HandleFunc("/api/files/", s.fileDownloadHandler)

	// Lyrics
	mux.HandleFunc("/api/lyrics", s.lyricsHandler)

	// Static file server for web frontend
	staticDir := "static"
	if _, err := os.Stat(staticDir); err == nil {
		fs := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Check if the requested file exists
			path := filepath.Join(staticDir, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				// Serve index.html for SPA routing
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	fmt.Printf("API server starting on port %s\n", s.port)
	return http.ListenAndServe(":"+s.port, corsMiddleware(mux))
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]string{"status": "healthy"})
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	source := r.URL.Query().Get("source")
	if source == "" {
		source = "deezer" // Default to Deezer as it doesn't require API key
	}

	trackLimit, _ := strconv.Atoi(r.URL.Query().Get("track_limit"))
	if trackLimit <= 0 {
		trackLimit = 10
	}

	artistLimit, _ := strconv.Atoi(r.URL.Query().Get("artist_limit"))
	if artistLimit <= 0 {
		artistLimit = 3
	}

	var result string
	var err error

	switch source {
	case "deezer":
		result, err = backend.SearchDeezerAll(query, trackLimit, artistLimit)
	case "spotify", "spotify_web":
		// Use Spotify Web (internal API) - same as Telegram bot, no rate limits
		result, err = backend.SearchSpotifyWeb(query, trackLimit)
	case "spotify_api":
		// Use official Spotify API (may be rate limited)
		result, err = backend.SearchSpotifyAll(query, trackLimit, artistLimit)
	default:
		writeError(w, http.StatusBadRequest, "Invalid source. Use 'deezer', 'spotify', or 'spotify_api'")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse search results")
		return
	}

	writeSuccess(w, data)
}

func (s *Server) metadataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		writeError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	var result string
	var err error

	// Detect if it's a Spotify or Deezer URL
	if strings.Contains(urlParam, "spotify.com") || strings.Contains(urlParam, "spotify:") {
		result, err = backend.GetSpotifyMetadataWithDeezerFallback(urlParam)
	} else if strings.Contains(urlParam, "deezer.com") {
		// Parse Deezer URL to get type and ID
		parsed, parseErr := backend.ParseDeezerURLExport(urlParam)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "Invalid Deezer URL: "+parseErr.Error())
			return
		}

		var parsedData map[string]string
		if err := json.Unmarshal([]byte(parsed), &parsedData); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse URL data")
			return
		}

		result, err = backend.GetDeezerMetadata(parsedData["type"], parsedData["id"])
	} else {
		writeError(w, http.StatusBadRequest, "URL must be a Spotify or Deezer link")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse metadata")
		return
	}

	writeSuccess(w, data)
}

func (s *Server) parseURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		writeError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	var result string
	var err error

	if strings.Contains(urlParam, "spotify.com") || strings.Contains(urlParam, "spotify:") {
		result, err = backend.ParseSpotifyURL(urlParam)
	} else if strings.Contains(urlParam, "deezer.com") {
		result, err = backend.ParseDeezerURLExport(urlParam)
	} else {
		writeError(w, http.StatusBadRequest, "URL must be a Spotify or Deezer link")
		return
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse URL")
		return
	}

	writeSuccess(w, data)
}

func (s *Server) availabilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	spotifyID := r.URL.Query().Get("spotify_id")
	isrc := r.URL.Query().Get("isrc")
	deezerID := r.URL.Query().Get("deezer_id")

	var result string
	var err error

	if deezerID != "" {
		result, err = backend.CheckAvailabilityFromDeezerID(deezerID)
	} else if spotifyID != "" || isrc != "" {
		result, err = backend.CheckAvailability(spotifyID, isrc)
	} else {
		writeError(w, http.StatusBadRequest, "Either spotify_id, isrc, or deezer_id is required")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse availability data")
		return
	}

	writeSuccess(w, data)
}

func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var requestBody struct {
		ISRC                 string `json:"isrc"`
		Service              string `json:"service"`
		SpotifyID            string `json:"spotify_id"`
		TrackName            string `json:"track_name"`
		ArtistName           string `json:"artist_name"`
		AlbumName            string `json:"album_name"`
		AlbumArtist          string `json:"album_artist"`
		CoverURL             string `json:"cover_url"`
		OutputDir            string `json:"output_dir"`
		FilenameFormat       string `json:"filename_format"`
		Quality              string `json:"quality"`
		EmbedLyrics          bool   `json:"embed_lyrics"`
		EmbedMaxQualityCover bool   `json:"embed_max_quality_cover"`
		TrackNumber          int    `json:"track_number"`
		DiscNumber           int    `json:"disc_number"`
		TotalTracks          int    `json:"total_tracks"`
		ReleaseDate          string `json:"release_date"`
		ItemID               string `json:"item_id"`
		DurationMS           int    `json:"duration_ms"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}

	// Set defaults
	if requestBody.OutputDir == "" {
		requestBody.OutputDir = "/tmp/spotiflac_downloads"
	}
	if requestBody.FilenameFormat == "" {
		requestBody.FilenameFormat = "{artist} - {title}"
	}
	if requestBody.Quality == "" {
		requestBody.Quality = "LOSSLESS"
	}

	// Convert to JSON for the backend
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to serialize request")
		return
	}

	var result string
	if requestBody.Service != "" {
		// Download from specific service
		result, err = backend.DownloadTrack(string(requestJSON))
	} else {
		// Download with fallback
		result, err = backend.DownloadWithFallback(string(requestJSON))
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse download result")
		return
	}

	writeSuccess(w, data)
}

func (s *Server) progressHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	itemID := r.URL.Query().Get("item_id")

	// Get all download progress
	result := backend.GetAllDownloadProgress()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse progress data")
		return
	}

	// If itemID is specified, filter to return only that item's progress
	if itemID != "" {
		if itemProgress, ok := data[itemID]; ok {
			writeSuccess(w, map[string]interface{}{itemID: itemProgress})
			return
		}
		// Item not found, return empty
		writeSuccess(w, map[string]interface{}{})
		return
	}

	writeSuccess(w, data)
}

func (s *Server) lyricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	spotifyID := r.URL.Query().Get("spotify_id")
	trackName := r.URL.Query().Get("track_name")
	artistName := r.URL.Query().Get("artist_name")

	if trackName == "" || artistName == "" {
		writeError(w, http.StatusBadRequest, "track_name and artist_name are required")
		return
	}

	result, err := backend.FetchLyrics(spotifyID, trackName, artistName)
	if err != nil {
		writeError(w, http.StatusNotFound, "Lyrics not found: "+err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse lyrics")
		return
	}

	writeSuccess(w, data)
}

func (s *Server) fileDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "Filename is required")
		return
	}

	// Sanitize filename to prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join("/tmp/spotiflac_downloads", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}
