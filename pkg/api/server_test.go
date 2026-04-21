package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func withBackendStubs(t *testing.T) {
	t.Helper()

	origSearchDeezerAll := searchDeezerAllFn
	origSearchSpotifyWeb := searchSpotifyWebFn
	origSearchSpotifyAll := searchSpotifyAllFn
	origGetSpotifyMetadata := getSpotifyMetadataWithFallbackFn
	origParseSpotifyURL := parseSpotifyURLFn
	origGetSpotifyWebPlaylist := getSpotifyWebPlaylistFn
	origParseDeezer := parseDeezerURLExportFn
	origGetDeezerMetadata := getDeezerMetadataFn
	origCheckAvailDeezer := checkAvailabilityFromDeezerIDFn
	origCheckAvail := checkAvailabilityFn
	origDownloadByStrategy := downloadByStrategyFn
	origDownloadTrack := downloadTrackFn
	origDownloadFallback := downloadWithFallbackFn
	origProgress := getAllDownloadProgressFn
	origFetchLyrics := fetchLyricsFn

	searchDeezerAllFn = func(string, int, int) (string, error) { return `{"tracks":[]}`, nil }
	searchSpotifyWebFn = func(string, int) (string, error) { return `{"tracks":[]}`, nil }
	searchSpotifyAllFn = func(string, int, int) (string, error) { return `{"tracks":[]}`, nil }
	getSpotifyMetadataWithFallbackFn = func(string) (string, error) { return `{"track":{"name":"x"}}`, nil }
	parseSpotifyURLFn = func(string) (string, error) { return `{"type":"track","id":"1"}`, nil }
	getSpotifyWebPlaylistFn = func(string) (string, error) { return `{"playlist_info":{"name":"p"},"tracks":[]}`, nil }
	parseDeezerURLExportFn = func(string) (string, error) { return `{"type":"track","id":"1"}`, nil }
	getDeezerMetadataFn = func(string, string) (string, error) { return `{"track":{"name":"x"}}`, nil }
	checkAvailabilityFromDeezerIDFn = func(string) (string, error) { return `{"tidal":true}`, nil }
	checkAvailabilityFn = func(string, string) (string, error) { return `{"tidal":true}`, nil }
	downloadByStrategyFn = func(string) (string, error) { return `{"success":true,"file_path":"/tmp/a.flac"}`, nil }
	downloadTrackFn = func(string) (string, error) { return `{"success":true,"file_path":"/tmp/a.flac"}`, nil }
	downloadWithFallbackFn = func(string) (string, error) { return `{"success":true,"file_path":"/tmp/a.flac"}`, nil }
	getAllDownloadProgressFn = func() string { return `{}` }
	fetchLyricsFn = func(string, string, string) (string, error) { return `{"lyrics":"ok"}`, nil }

	t.Cleanup(func() {
		searchDeezerAllFn = origSearchDeezerAll
		searchSpotifyWebFn = origSearchSpotifyWeb
		searchSpotifyAllFn = origSearchSpotifyAll
		getSpotifyMetadataWithFallbackFn = origGetSpotifyMetadata
		parseSpotifyURLFn = origParseSpotifyURL
		getSpotifyWebPlaylistFn = origGetSpotifyWebPlaylist
		parseDeezerURLExportFn = origParseDeezer
		getDeezerMetadataFn = origGetDeezerMetadata
		checkAvailabilityFromDeezerIDFn = origCheckAvailDeezer
		checkAvailabilityFn = origCheckAvail
		downloadByStrategyFn = origDownloadByStrategy
		downloadTrackFn = origDownloadTrack
		downloadWithFallbackFn = origDownloadFallback
		getAllDownloadProgressFn = origProgress
		fetchLyricsFn = origFetchLyrics
	})
}

func TestClientIdentifier(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:1234"

	if got := clientIdentifier(req); got != "203.0.113.10" {
		t.Fatalf("expected forwarded IP, got %q", got)
	}
}

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	rl.nowFunc = func() time.Time { return now }

	if !rl.allow("a") {
		t.Fatal("first request should be allowed")
	}
	if !rl.allow("a") {
		t.Fatal("second request should be allowed")
	}
	if rl.allow("a") {
		t.Fatal("third request should be blocked")
	}

	now = now.Add(time.Minute + time.Second)
	if !rl.allow("a") {
		t.Fatal("request should be allowed after window expires")
	}
}

func TestStaticHandlerServesRouteAnd404(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(staticDir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("home"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "docs", "index.html"), []byte("docs-page"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "404.html"), []byte("not-found"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer("0")
	h := s.staticHandler(staticDir)

	t.Run("deep route serves route index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "docs-page") {
			t.Fatalf("unexpected body: %q", rr.Body.String())
		}
	})

	t.Run("missing route serves 404 file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/missing", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "not-found") {
			t.Fatalf("unexpected body: %q", rr.Body.String())
		}
	})

	t.Run("post rejected with json error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}

		var resp APIResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("expected JSON response: %v", err)
		}
		if resp.Success {
			t.Fatal("expected success=false")
		}
	})
}

func TestFileDownloadTraversalRejectedByBaseSanitization(t *testing.T) {
	s := NewServer("0")
	t.Run("plain traversal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/files/../../etc/passwd", nil)
		rr := httptest.NewRecorder()
		s.fileDownloadHandler(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected sanitized missing file 404, got %d", rr.Code)
		}
	})

	t.Run("encoded traversal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/files/%2e%2e%2fetc%2fpasswd", nil)
		rr := httptest.NewRecorder()
		s.fileDownloadHandler(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected sanitized missing file 404, got %d", rr.Code)
		}
	})
}

func TestProgressRateLimit(t *testing.T) {
	s := NewServer("0")
	s.progressLimiter = newRateLimiter(1, time.Minute)
	fixed := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	s.progressLimiter.nowFunc = func() time.Time { return fixed }

	req1 := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	req1.RemoteAddr = "127.0.0.1:9999"
	rr1 := httptest.NewRecorder()
	s.progressHandler(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	req2.RemoteAddr = "127.0.0.1:9999"
	rr2 := httptest.NewRecorder()
	s.progressHandler(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request expected 429, got %d", rr2.Code)
	}
}

func TestDownloadRateLimit(t *testing.T) {
	withBackendStubs(t)

	s := NewServer("0")
	s.downloadLimiter = newRateLimiter(1, time.Minute)
	fixed := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	s.downloadLimiter.nowFunc = func() time.Time { return fixed }

	body := strings.NewReader(`{"track_name":"A","artist_name":"B"}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/download", body)
	req1.RemoteAddr = "127.0.0.1:9998"
	rr1 := httptest.NewRecorder()
	s.downloadHandler(rr1, req1)
	if rr1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request should not be rate limited")
	}

	body2 := strings.NewReader(`{"track_name":"A","artist_name":"B"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/download", body2)
	req2.RemoteAddr = "127.0.0.1:9998"
	rr2 := httptest.NewRecorder()
	s.downloadHandler(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request expected 429, got %d", rr2.Code)
	}
}

func TestSimpleHandlersAndValidationBranches(t *testing.T) {
	s := NewServer("0")
	withBackendStubs(t)

	t.Run("build handler routes health", func(t *testing.T) {
		h := s.buildHandler()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("cors middleware preflight", func(t *testing.T) {
		h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		req := httptest.NewRequest(http.MethodOptions, "/health", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("expected CORS header set")
		}
	})

	t.Run("health handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()
		s.healthHandler(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("api not found handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
		rr := httptest.NewRecorder()
		s.apiNotFoundHandler(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("search validations", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPost, "/api/search?q=a", nil)
		rrMethod := httptest.NewRecorder()
		s.searchHandler(rrMethod, reqMethod)
		if rrMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rrMethod.Code)
		}

		reqMissing := httptest.NewRequest(http.MethodGet, "/api/search", nil)
		rrMissing := httptest.NewRecorder()
		s.searchHandler(rrMissing, reqMissing)
		if rrMissing.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrMissing.Code)
		}

		reqInvalid := httptest.NewRequest(http.MethodGet, "/api/search?q=a&source=invalid", nil)
		rrInvalid := httptest.NewRecorder()
		s.searchHandler(rrInvalid, reqInvalid)
		if rrInvalid.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrInvalid.Code)
		}
	})

	t.Run("metadata validations", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPost, "/api/metadata", nil)
		rrMethod := httptest.NewRecorder()
		s.metadataHandler(rrMethod, reqMethod)
		if rrMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rrMethod.Code)
		}

		reqMissing := httptest.NewRequest(http.MethodGet, "/api/metadata", nil)
		rrMissing := httptest.NewRecorder()
		s.metadataHandler(rrMissing, reqMissing)
		if rrMissing.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrMissing.Code)
		}

		reqInvalid := httptest.NewRequest(http.MethodGet, "/api/metadata?url=https://example.com", nil)
		rrInvalid := httptest.NewRecorder()
		s.metadataHandler(rrInvalid, reqInvalid)
		if rrInvalid.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrInvalid.Code)
		}
	})

	t.Run("parse-url validations", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPost, "/api/parse-url", nil)
		rrMethod := httptest.NewRecorder()
		s.parseURLHandler(rrMethod, reqMethod)
		if rrMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rrMethod.Code)
		}

		reqMissing := httptest.NewRequest(http.MethodGet, "/api/parse-url", nil)
		rrMissing := httptest.NewRecorder()
		s.parseURLHandler(rrMissing, reqMissing)
		if rrMissing.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrMissing.Code)
		}

		reqInvalid := httptest.NewRequest(http.MethodGet, "/api/parse-url?url=https://example.com", nil)
		rrInvalid := httptest.NewRecorder()
		s.parseURLHandler(rrInvalid, reqInvalid)
		if rrInvalid.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrInvalid.Code)
		}
	})

	t.Run("availability validations", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPost, "/api/availability", nil)
		rrMethod := httptest.NewRecorder()
		s.availabilityHandler(rrMethod, reqMethod)
		if rrMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rrMethod.Code)
		}

		reqMissing := httptest.NewRequest(http.MethodGet, "/api/availability", nil)
		rrMissing := httptest.NewRecorder()
		s.availabilityHandler(rrMissing, reqMissing)
		if rrMissing.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrMissing.Code)
		}
	})

	t.Run("progress method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/progress", nil)
		rr := httptest.NewRecorder()
		s.progressHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("download invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/download", strings.NewReader("{bad"))
		rr := httptest.NewRecorder()
		s.downloadHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("lyrics validations", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPost, "/api/lyrics", nil)
		rrMethod := httptest.NewRecorder()
		s.lyricsHandler(rrMethod, reqMethod)
		if rrMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rrMethod.Code)
		}

		reqMissing := httptest.NewRequest(http.MethodGet, "/api/lyrics", nil)
		rrMissing := httptest.NewRecorder()
		s.lyricsHandler(rrMissing, reqMissing)
		if rrMissing.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrMissing.Code)
		}
	})

	t.Run("file download validations", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPost, "/api/files/song.flac", nil)
		rrMethod := httptest.NewRecorder()
		s.fileDownloadHandler(rrMethod, reqMethod)
		if rrMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rrMethod.Code)
		}

		reqMissing := httptest.NewRequest(http.MethodGet, "/api/files/", nil)
		rrMissing := httptest.NewRecorder()
		s.fileDownloadHandler(rrMissing, reqMissing)
		if rrMissing.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rrMissing.Code)
		}
	})
}

func TestBackendHandlerSuccessAndErrorBranches(t *testing.T) {
	t.Run("search success and backend error", func(t *testing.T) {
		withBackendStubs(t)
		s := NewServer("0")

		reqOK := httptest.NewRequest(http.MethodGet, "/api/search?q=hello&source=deezer", nil)
		rrOK := httptest.NewRecorder()
		s.searchHandler(rrOK, reqOK)
		if rrOK.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrOK.Code)
		}

		searchDeezerAllFn = func(string, int, int) (string, error) { return "", errors.New("boom") }
		reqErr := httptest.NewRequest(http.MethodGet, "/api/search?q=hello&source=deezer", nil)
		rrErr := httptest.NewRecorder()
		s.searchHandler(rrErr, reqErr)
		if rrErr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rrErr.Code)
		}
	})

	t.Run("parse-url and metadata deezer success", func(t *testing.T) {
		withBackendStubs(t)
		s := NewServer("0")

		reqParse := httptest.NewRequest(http.MethodGet, "/api/parse-url?url=https://open.spotify.com/track/x", nil)
		rrParse := httptest.NewRecorder()
		s.parseURLHandler(rrParse, reqParse)
		if rrParse.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrParse.Code)
		}

		reqMeta := httptest.NewRequest(http.MethodGet, "/api/metadata?url=https://www.deezer.com/track/1", nil)
		rrMeta := httptest.NewRecorder()
		s.metadataHandler(rrMeta, reqMeta)
		if rrMeta.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrMeta.Code)
		}
	})

	t.Run("availability branches and lyrics success", func(t *testing.T) {
		withBackendStubs(t)
		s := NewServer("0")

		reqAvailSpotify := httptest.NewRequest(http.MethodGet, "/api/availability?spotify_id=1", nil)
		rrAvailSpotify := httptest.NewRecorder()
		s.availabilityHandler(rrAvailSpotify, reqAvailSpotify)
		if rrAvailSpotify.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrAvailSpotify.Code)
		}

		reqAvailDeezer := httptest.NewRequest(http.MethodGet, "/api/availability?deezer_id=1", nil)
		rrAvailDeezer := httptest.NewRecorder()
		s.availabilityHandler(rrAvailDeezer, reqAvailDeezer)
		if rrAvailDeezer.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrAvailDeezer.Code)
		}

		reqLyrics := httptest.NewRequest(http.MethodGet, "/api/lyrics?track_name=a&artist_name=b", nil)
		rrLyrics := httptest.NewRecorder()
		s.lyricsHandler(rrLyrics, reqLyrics)
		if rrLyrics.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrLyrics.Code)
		}
	})

	t.Run("download success and progress filter", func(t *testing.T) {
		withBackendStubs(t)
		s := NewServer("0")

		reqDL := httptest.NewRequest(http.MethodPost, "/api/download", strings.NewReader(`{"track_name":"A","artist_name":"B"}`))
		// avoid rate limit interference
		reqDL.RemoteAddr = "127.0.0.1:10001"
		rrDL := httptest.NewRecorder()
		s.downloadHandler(rrDL, reqDL)
		if rrDL.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrDL.Code)
		}

		getAllDownloadProgressFn = func() string { return `{"abc":{"progress":50}}` }
		reqProgress := httptest.NewRequest(http.MethodGet, "/api/progress?item_id=abc", nil)
		reqProgress.RemoteAddr = "127.0.0.1:10002"
		rrProgress := httptest.NewRecorder()
		s.progressHandler(rrProgress, reqProgress)
		if rrProgress.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rrProgress.Code)
		}
	})
}
