package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	gobackend "github.com/zarz/spotiflac_android/go_backend"
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

	// Lyrics
	mux.HandleFunc("/api/lyrics", s.lyricsHandler)

	fmt.Printf("API server starting on port %s\n", s.port)
	return http.ListenAndServe(":"+s.port, mux)
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
		result, err = gobackend.SearchDeezerAll(query, trackLimit, artistLimit)
	case "spotify":
		result, err = gobackend.SearchSpotifyAll(query, trackLimit, artistLimit)
	default:
		writeError(w, http.StatusBadRequest, "Invalid source. Use 'deezer' or 'spotify'")
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
		result, err = gobackend.GetSpotifyMetadataWithDeezerFallback(urlParam)
	} else if strings.Contains(urlParam, "deezer.com") {
		// Parse Deezer URL to get type and ID
		parsed, parseErr := gobackend.ParseDeezerURLExport(urlParam)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "Invalid Deezer URL: "+parseErr.Error())
			return
		}

		var parsedData map[string]string
		if err := json.Unmarshal([]byte(parsed), &parsedData); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse URL data")
			return
		}

		result, err = gobackend.GetDeezerMetadata(parsedData["type"], parsedData["id"])
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
		result, err = gobackend.ParseSpotifyURL(urlParam)
	} else if strings.Contains(urlParam, "deezer.com") {
		result, err = gobackend.ParseDeezerURLExport(urlParam)
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
		result, err = gobackend.CheckAvailabilityFromDeezerID(deezerID)
	} else if spotifyID != "" || isrc != "" {
		result, err = gobackend.CheckAvailability(spotifyID, isrc)
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
		result, err = gobackend.DownloadTrack(string(requestJSON))
	} else {
		// Download with fallback
		result, err = gobackend.DownloadWithFallback(string(requestJSON))
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
	result := gobackend.GetAllDownloadProgress()

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

	result, err := gobackend.FetchLyrics(spotifyID, trackName, artistName)
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
