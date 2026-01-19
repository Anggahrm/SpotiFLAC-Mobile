package backend

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ============================================
// TOTP SECRETS FOR AUTHENTICATION
// ============================================

var totpSecrets = map[int][]int{
	59: {123, 105, 79, 70, 110, 59, 52, 125, 60, 49, 80, 70, 89, 75, 80, 86, 63, 53, 123, 37, 117, 49, 52, 93, 77, 62, 47, 86, 48, 104, 68, 72},
	60: {79, 109, 69, 123, 90, 65, 46, 74, 94, 34, 58, 48, 70, 71, 92, 85, 122, 63, 91, 64, 87, 87},
	61: {44, 55, 47, 42, 70, 40, 34, 114, 76, 74, 50, 111, 120, 97, 75, 76, 94, 102, 43, 69, 49, 120, 118, 80, 64, 78},
}

const totpVersion = 61

// ============================================
// SPOTIFY WEB CLIENT
// ============================================

// SpotifyWebClient handles Spotify internal API interactions
type SpotifyWebClient struct {
	httpClient    *http.Client
	accessToken   string
	clientToken   string
	clientID      string
	deviceID      string
	clientVersion string
	cookies       map[string]string
	initialized   bool
	mu            sync.Mutex
}

var (
	spotifyWebClient *SpotifyWebClient
	spotifyWebOnce   sync.Once
)

// GetSpotifyWebClient returns a singleton SpotifyWebClient
func GetSpotifyWebClient() *SpotifyWebClient {
	spotifyWebOnce.Do(func() {
		spotifyWebClient = &SpotifyWebClient{
			httpClient: &http.Client{Timeout: 30 * time.Second},
			cookies:    make(map[string]string),
		}
	})
	return spotifyWebClient
}

// ============================================
// TOTP GENERATION
// ============================================

func (c *SpotifyWebClient) generateTOTP() (string, int) {
	secretList := totpSecrets[totpVersion]

	// Transform secret
	transformed := make([]int, len(secretList))
	for i, v := range secretList {
		transformed[i] = v ^ ((i % 33) + 9)
	}

	// Join as string of numbers
	var joined string
	for _, v := range transformed {
		joined += fmt.Sprintf("%d", v)
	}

	// Convert to hex
	var hexStr string
	for _, ch := range joined {
		hexStr += fmt.Sprintf("%02x", ch)
	}

	// Convert hex to bytes
	hexBytes := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		var b int
		fmt.Sscanf(hexStr[i:i+2], "%02x", &b)
		hexBytes[i/2] = byte(b)
	}

	// Base32 encode
	secret := base32.StdEncoding.EncodeToString(hexBytes)

	// Generate TOTP code
	counter := time.Now().Unix() / 30
	code := c.generateTOTPCode(secret, counter)

	return code, totpVersion
}

func (c *SpotifyWebClient) generateTOTPCode(secret string, counter int64) string {
	// Decode base32 secret
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "000000"
	}

	// Convert counter to 8 bytes (big-endian)
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))

	// HMAC-SHA1
	h := hmac.New(sha1.New, key)
	h.Write(counterBytes)
	hmacResult := h.Sum(nil)

	// Dynamic truncation
	offset := hmacResult[len(hmacResult)-1] & 0x0f
	code := int(hmacResult[offset]&0x7f)<<24 |
		int(hmacResult[offset+1]&0xff)<<16 |
		int(hmacResult[offset+2]&0xff)<<8 |
		int(hmacResult[offset+3]&0xff)

	// Get 6 digits
	otp := code % 1000000
	return fmt.Sprintf("%06d", otp)
}

// ============================================
// AUTHENTICATION
// ============================================

func (c *SpotifyWebClient) buildCookieHeader() string {
	var parts []string
	for name, value := range c.cookies {
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, "; ")
}

func (c *SpotifyWebClient) extractCookies(resp *http.Response) {
	for _, cookie := range resp.Cookies() {
		c.cookies[cookie.Name] = cookie.Value
		if cookie.Name == "sp_t" {
			c.deviceID = cookie.Value
		}
	}
}

func (c *SpotifyWebClient) getSessionInfo() error {
	req, err := http.NewRequest("GET", "https://open.spotify.com", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if cookieHeader := c.buildCookieHeader(); cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.extractCookies(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Extract client version from page
	re := regexp.MustCompile(`<script id="appServerConfig" type="text/plain">([^<]+)</script>`)
	matches := re.FindSubmatch(body)
	if len(matches) > 1 {
		decoded, err := base64.StdEncoding.DecodeString(string(matches[1]))
		if err == nil {
			var cfg struct {
				ClientVersion string `json:"clientVersion"`
			}
			if json.Unmarshal(decoded, &cfg) == nil {
				c.clientVersion = cfg.ClientVersion
			}
		}
	}

	return nil
}

func (c *SpotifyWebClient) getAccessToken() error {
	code, version := c.generateTOTP()

	tokenURL := fmt.Sprintf("https://open.spotify.com/api/token?reason=init&productType=web-player&totp=%s&totpVer=%d&totpServer=%s",
		code, version, code)

	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	if cookieHeader := c.buildCookieHeader(); cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.extractCookies(resp)

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get access token: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data struct {
		AccessToken string `json:"accessToken"`
		ClientID    string `json:"clientId"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	c.accessToken = data.AccessToken
	c.clientID = data.ClientID

	return nil
}

func (c *SpotifyWebClient) getClientToken() error {
	if c.clientID == "" || c.deviceID == "" || c.clientVersion == "" {
		if err := c.getSessionInfo(); err != nil {
			return err
		}
		if err := c.getAccessToken(); err != nil {
			return err
		}
	}

	if c.deviceID == "" {
		return errors.New("failed to get device ID from sp_t cookie")
	}

	payload := map[string]interface{}{
		"client_data": map[string]interface{}{
			"client_version": c.clientVersion,
			"client_id":      c.clientID,
			"js_sdk_data": map[string]interface{}{
				"device_brand": "unknown",
				"device_model": "unknown",
				"os":           "windows",
				"os_version":   "NT 10.0",
				"device_id":    c.deviceID,
				"device_type":  "computer",
			},
		},
	}

	payloadJSON, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://clienttoken.spotify.com/v1/clienttoken", strings.NewReader(string(payloadJSON)))
	if err != nil {
		return err
	}

	req.Header.Set("Authority", "clienttoken.spotify.com")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get client token: HTTP %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data struct {
		ResponseType string `json:"response_type"`
		GrantedToken struct {
			Token string `json:"token"`
		} `json:"granted_token"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	if data.ResponseType != "RESPONSE_GRANTED_TOKEN_RESPONSE" {
		return fmt.Errorf("invalid client token response: %s", data.ResponseType)
	}

	c.clientToken = data.GrantedToken.Token
	c.initialized = true

	return nil
}

func (c *SpotifyWebClient) ensureInitialized() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized || c.accessToken == "" || c.clientToken == "" {
		if err := c.getSessionInfo(); err != nil {
			return err
		}
		if err := c.getAccessToken(); err != nil {
			return err
		}
		if err := c.getClientToken(); err != nil {
			return err
		}
	}
	return nil
}

// ============================================
// GRAPHQL QUERY
// ============================================

func (c *SpotifyWebClient) query(payload interface{}) (map[string]interface{}, error) {
	if err := c.ensureInitialized(); err != nil {
		return nil, err
	}

	payloadJSON, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api-partner.spotify.com/pathfinder/v2/query", strings.NewReader(string(payloadJSON)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Client-Token", c.clientToken)
	req.Header.Set("Spotify-App-Version", c.clientVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Token expired, re-initialize
		c.initialized = false
		c.accessToken = ""
		c.clientToken = ""
		if err := c.ensureInitialized(); err != nil {
			return nil, err
		}
		return c.query(payload) // Retry
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func getNestedValue(obj interface{}, path string) interface{} {
	if obj == nil || path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		if current == nil {
			return nil
		}

		// Check if it's an array index
		var idx int
		if n, _ := fmt.Sscanf(part, "%d", &idx); n == 1 {
			if arr, ok := current.([]interface{}); ok && idx < len(arr) {
				current = arr[idx]
			} else {
				return nil
			}
		} else {
			if m, ok := current.(map[string]interface{}); ok {
				current = m[part]
			} else {
				return nil
			}
		}
	}

	return current
}

func getStringValue(obj interface{}, path string) string {
	val := getNestedValue(obj, path)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func getIntValue(obj interface{}, path string) int {
	val := getNestedValue(obj, path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// ============================================
// SEARCH FUNCTION
// ============================================

// SpotifyWebSearchResult represents a search result
type SpotifyWebSearchResult struct {
	ID         string `json:"id"`
	SpotifyID  string `json:"spotify_id"`
	Name       string `json:"name"`
	Artists    string `json:"artists"`
	AlbumName  string `json:"album_name"`
	DurationMS int    `json:"duration_ms"`
	Images     string `json:"images"`
	ItemType   string `json:"item_type"` // track, album, artist, playlist
	ProviderID string `json:"provider_id"`
}

// SearchSpotifyWeb searches Spotify using internal API
func (c *SpotifyWebClient) Search(query string, limit int) ([]SpotifyWebSearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	payload := map[string]interface{}{
		"variables": map[string]interface{}{
			"searchTerm":                   query,
			"offset":                       0,
			"limit":                        limit,
			"numberOfTopResults":           5,
			"includeAudiobooks":            true,
			"includeArtistHasConcertsField": false,
			"includePreReleases":           true,
			"includeAuthors":               false,
		},
		"operationName": "searchDesktop",
		"extensions": map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": "fcad5a3e0d5af727fb76966f06971c19cfa2275e6ff7671196753e008611873c",
			},
		},
	}

	response, err := c.query(payload)
	if err != nil {
		return nil, err
	}

	var results []SpotifyWebSearchResult

	searchData := getNestedValue(response, "data.searchV2")
	if searchData == nil {
		return results, nil
	}

	// Parse tracks
	tracksData := getNestedValue(searchData, "tracksV2.items")
	if tracksData == nil {
		tracksData = getNestedValue(searchData, "tracks.items")
	}

	if tracks, ok := tracksData.([]interface{}); ok {
		for _, item := range tracks {
			var trackData interface{}

			// Handle different API structures
			if itemData := getNestedValue(item, "item.data"); itemData != nil {
				trackData = itemData
			} else if t := getNestedValue(item, "track"); t != nil {
				trackData = t
			} else if d := getNestedValue(item, "data"); d != nil {
				trackData = d
			}

			if trackData == nil {
				continue
			}

			trackID := getStringValue(trackData, "id")
			if trackID == "" {
				uri := getStringValue(trackData, "uri")
				if uri != "" {
					parts := strings.Split(uri, ":")
					trackID = parts[len(parts)-1]
				}
			}
			if trackID == "" {
				continue
			}

			// Get artists
			var artistNames []string
			if artistItems := getNestedValue(trackData, "artists.items"); artistItems != nil {
				if artists, ok := artistItems.([]interface{}); ok {
					for _, a := range artists {
						name := getStringValue(a, "profile.name")
						if name != "" {
							artistNames = append(artistNames, name)
						}
					}
				}
			}

			trackName := getStringValue(trackData, "name")
			if trackName == "" {
				continue
			}

			albumName := getStringValue(trackData, "albumOfTrack.name")
			coverURL := getStringValue(trackData, "albumOfTrack.coverArt.sources.0.url")
			durationMS := getIntValue(trackData, "duration.totalMilliseconds")
			if durationMS == 0 {
				durationMS = getIntValue(trackData, "trackDuration.totalMilliseconds")
			}

			results = append(results, SpotifyWebSearchResult{
				ID:         trackID,
				SpotifyID:  trackID,
				Name:       trackName,
				Artists:    strings.Join(artistNames, ", "),
				AlbumName:  albumName,
				DurationMS: durationMS,
				Images:     coverURL,
				ItemType:   "track",
				ProviderID: "spotify-web",
			})
		}
	}

	// Parse albums (limit to 5)
	albumsData := getNestedValue(searchData, "albums.items")
	if albumsData == nil {
		albumsData = getNestedValue(searchData, "albumsV2.items")
	}

	if albums, ok := albumsData.([]interface{}); ok {
		count := 0
		for _, item := range albums {
			if count >= 5 {
				break
			}

			albumData := getNestedValue(item, "item.data")
			if albumData == nil {
				albumData = getNestedValue(item, "data")
				if albumData == nil {
					albumData = item
				}
			}

			uri := getStringValue(albumData, "uri")
			if uri == "" {
				continue
			}
			parts := strings.Split(uri, ":")
			albumID := parts[len(parts)-1]

			albumName := getStringValue(albumData, "name")
			if albumName == "" {
				continue
			}

			// Get artists
			var artistNames []string
			if artistItems := getNestedValue(albumData, "artists.items"); artistItems != nil {
				if artists, ok := artistItems.([]interface{}); ok {
					for _, a := range artists {
						name := getStringValue(a, "profile.name")
						if name != "" {
							artistNames = append(artistNames, name)
						}
					}
				}
			}

			coverURL := getStringValue(albumData, "coverArt.sources.0.url")

			results = append(results, SpotifyWebSearchResult{
				ID:         albumID,
				Name:       albumName,
				Artists:    strings.Join(artistNames, ", "),
				Images:     coverURL,
				ItemType:   "album",
				ProviderID: "spotify-web",
			})
			count++
		}
	}

	return results, nil
}

// ============================================
// ISRC ENRICHMENT
// ============================================

// EnrichISRC gets real ISRC via SongLink -> Deezer
func EnrichISRC(spotifyID string) string {
	// Try SongLink first
	deezerURL := getDeezerURLFromSongLink(spotifyID)
	if deezerURL != "" {
		isrc := getISRCFromDeezer(deezerURL)
		if isrc != "" {
			return isrc
		}
	}
	return spotifyID // Fallback to Spotify ID
}

func getDeezerURLFromSongLink(spotifyID string) string {
	spotifyURL := "https://open.spotify.com/track/" + spotifyID
	apiURL := "https://api.song.link/v1-alpha.1/links?url=" + url.QueryEscape(spotifyURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		LinksByPlatform struct {
			Deezer struct {
				URL string `json:"url"`
			} `json:"deezer"`
		} `json:"linksByPlatform"`
	}
	if json.Unmarshal(body, &data) != nil {
		return ""
	}

	return data.LinksByPlatform.Deezer.URL
}

func getISRCFromDeezer(deezerURL string) string {
	re := regexp.MustCompile(`/track/(\d+)`)
	matches := re.FindStringSubmatch(deezerURL)
	if len(matches) < 2 {
		return ""
	}

	trackID := matches[1]
	apiURL := "https://api.deezer.com/track/" + trackID

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		ISRC string `json:"isrc"`
	}
	if json.Unmarshal(body, &data) != nil {
		return ""
	}

	return data.ISRC
}

// ============================================
// PLAYLIST TYPES AND FUNCTIONS
// ============================================

// SpotifyWebPlaylistTrack represents a track in a playlist
type SpotifyWebPlaylistTrack struct {
	ID         string `json:"id"`
	SpotifyID  string `json:"spotify_id"`
	Name       string `json:"name"`
	Artists    string `json:"artists"`
	AlbumName  string `json:"album_name"`
	DurationMS int    `json:"duration_ms"`
	Images     string `json:"images"`
	TrackNum   int    `json:"track_number"`
	DiscNum    int    `json:"disc_number"`
	ISRC       string `json:"isrc"`
}

// SpotifyWebPlaylistInfo represents playlist metadata
type SpotifyWebPlaylistInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Images      string `json:"images"`
	TotalTracks int    `json:"total_tracks"`
	Followers   int    `json:"followers"`
}

// SpotifyWebPlaylistResponse is the full playlist response
type SpotifyWebPlaylistResponse struct {
	PlaylistInfo SpotifyWebPlaylistInfo    `json:"playlist_info"`
	Tracks       []SpotifyWebPlaylistTrack `json:"tracks"`
}

// GetPlaylist fetches playlist data using Spotify's internal GraphQL API
// Uses the same hash and variables as the working spotify-web extension
func (c *SpotifyWebClient) GetPlaylist(playlistID string) (*SpotifyWebPlaylistResponse, error) {
	allTracks := []SpotifyWebPlaylistTrack{}
	offset := 0
	limit := 1000 // Extension uses 1000
	var playlistInfo *SpotifyWebPlaylistInfo
	var totalCount int

	for {
		// Use exact same payload as the spotify-web extension (index.js line 506-520)
		payload := map[string]interface{}{
			"variables": map[string]interface{}{
				"uri":                       "spotify:playlist:" + playlistID,
				"offset":                    offset,
				"limit":                     limit,
				"enableWatchFeedEntrypoint": false, // Critical: extension includes this!
			},
			"operationName": "fetchPlaylist",
			"extensions": map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"version": 1,
					// Use the EXACT hash from the working extension
					"sha256Hash": "bb67e0af06e8d6f52b531f97468ee4acd44cd0f82b988e15c2ea47b1148efc77",
				},
			},
		}

		result, err := c.query(payload)
		if err != nil {
			// If first page fails, try alternative methods
			if offset == 0 {
				return c.getPlaylistAlternative(playlistID)
			}
			break
		}

		// Parse this page
		resultBytes, err := json.Marshal(result)
		if err != nil {
			if offset == 0 {
				return nil, err
			}
			break
		}

		pageResp, err := c.parsePlaylistResponse(resultBytes, playlistID)
		if err != nil {
			if offset == 0 {
				return nil, err
			}
			break
		}

		// Store playlist info from first page
		if playlistInfo == nil {
			playlistInfo = &pageResp.PlaylistInfo
			totalCount = pageResp.PlaylistInfo.TotalTracks
		}

		// Append tracks
		allTracks = append(allTracks, pageResp.Tracks...)

		// Check if we have all tracks
		if len(pageResp.Tracks) == 0 || len(allTracks) >= totalCount || len(pageResp.Tracks) < limit {
			break
		}

		offset += limit
	}

	// Build final response
	if playlistInfo == nil {
		return c.getPlaylistAlternative(playlistID)
	}

	playlistInfo.TotalTracks = len(allTracks)
	return &SpotifyWebPlaylistResponse{
		PlaylistInfo: *playlistInfo,
		Tracks:       allTracks,
	}, nil
}

// getPlaylistAlternative tries different GraphQL endpoint for playlist
func (c *SpotifyWebClient) getPlaylistAlternative(playlistID string) (*SpotifyWebPlaylistResponse, error) {
	// Try with a different, older hash that might work
	payload := map[string]interface{}{
		"variables": map[string]interface{}{
			"uri":                       "spotify:playlist:" + playlistID,
			"offset":                    0,
			"limit":                     1000,
			"enableWatchFeedEntrypoint": false,
		},
		"operationName": "fetchPlaylistContents",
		"extensions": map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": "63c7a404af2831d94e44bb1ec51570b4e6e6499a37c93a28ee4dc3fd6ea9ae48",
			},
		},
	}

	result, err := c.query(payload)
	if err != nil {
		// Last resort: use public API
		return c.getPlaylistPublicAPI(playlistID)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return c.parsePlaylistResponse(resultBytes, playlistID)
}

// getPlaylistPublicAPI uses Spotify's public playlist API
func (c *SpotifyWebClient) getPlaylistPublicAPI(playlistID string) (*SpotifyWebPlaylistResponse, error) {
	if err := c.ensureInitialized(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s?fields=id,name,description,owner,images,followers,tracks.total,tracks.items(track(id,name,artists,album,duration_ms,disc_number,track_number,external_ids))", playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("playlist API failed with status %d", resp.StatusCode)
	}

	// Parse public API response
	var data struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Owner       struct {
			DisplayName string `json:"display_name"`
		} `json:"owner"`
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
		Followers struct {
			Total int `json:"total"`
		} `json:"followers"`
		Tracks struct {
			Total int `json:"total"`
			Items []struct {
				Track struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
					Album struct {
						Name   string `json:"name"`
						Images []struct {
							URL string `json:"url"`
						} `json:"images"`
					} `json:"album"`
					DurationMS  int `json:"duration_ms"`
					DiscNumber  int `json:"disc_number"`
					TrackNumber int `json:"track_number"`
					ExternalIds struct {
						ISRC string `json:"isrc"`
					} `json:"external_ids"`
				} `json:"track"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse playlist response: %w", err)
	}

	// Build response
	result := &SpotifyWebPlaylistResponse{
		PlaylistInfo: SpotifyWebPlaylistInfo{
			ID:          data.ID,
			Name:        data.Name,
			Description: data.Description,
			Owner:       data.Owner.DisplayName,
			TotalTracks: data.Tracks.Total,
			Followers:   data.Followers.Total,
		},
	}

	if len(data.Images) > 0 {
		result.PlaylistInfo.Images = data.Images[0].URL
	}

	for _, item := range data.Tracks.Items {
		if item.Track.ID == "" {
			continue // Skip local tracks
		}

		artists := make([]string, len(item.Track.Artists))
		for i, a := range item.Track.Artists {
			artists[i] = a.Name
		}

		trackImage := ""
		if len(item.Track.Album.Images) > 0 {
			trackImage = item.Track.Album.Images[0].URL
		}

		result.Tracks = append(result.Tracks, SpotifyWebPlaylistTrack{
			ID:         item.Track.ID,
			SpotifyID:  item.Track.ID,
			Name:       item.Track.Name,
			Artists:    strings.Join(artists, ", "),
			AlbumName:  item.Track.Album.Name,
			DurationMS: item.Track.DurationMS,
			Images:     trackImage,
			TrackNum:   item.Track.TrackNumber,
			DiscNum:    item.Track.DiscNumber,
			ISRC:       item.Track.ExternalIds.ISRC,
		})
	}

	return result, nil
}

// parsePlaylistResponse parses GraphQL playlist response
func (c *SpotifyWebClient) parsePlaylistResponse(body []byte, playlistID string) (*SpotifyWebPlaylistResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	// Navigate to playlist data
	playlistData := getNestedValue(raw, "data.playlistV2")
	if playlistData == nil {
		playlistData = getNestedValue(raw, "data.playlist")
	}
	if playlistData == nil {
		return nil, fmt.Errorf("playlist data not found in response")
	}

	playlist, ok := playlistData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid playlist data format")
	}

	// Extract playlist info
	result := &SpotifyWebPlaylistResponse{
		PlaylistInfo: SpotifyWebPlaylistInfo{
			ID: playlistID,
		},
	}

	if name, ok := playlist["name"].(string); ok {
		result.PlaylistInfo.Name = name
	}
	if desc, ok := playlist["description"].(string); ok {
		result.PlaylistInfo.Description = desc
	}

	// Get owner
	if ownerData := getNestedValue(playlist, "ownerV2.data"); ownerData != nil {
		if owner, ok := ownerData.(map[string]interface{}); ok {
			if name, ok := owner["name"].(string); ok {
				result.PlaylistInfo.Owner = name
			}
		}
	} else if ownerData := getNestedValue(playlist, "owner"); ownerData != nil {
		if owner, ok := ownerData.(map[string]interface{}); ok {
			if name, ok := owner["name"].(string); ok {
				result.PlaylistInfo.Owner = name
			}
		}
	}

	// Get images
	if images := getNestedValue(playlist, "images.items"); images != nil {
		if imgArr, ok := images.([]interface{}); ok && len(imgArr) > 0 {
			if img, ok := imgArr[0].(map[string]interface{}); ok {
				// Try different image sources
				for _, key := range []string{"url", "sources"} {
					if url, ok := img[key].(string); ok && url != "" {
						result.PlaylistInfo.Images = url
						break
					}
					if sources, ok := img[key].([]interface{}); ok && len(sources) > 0 {
						if src, ok := sources[0].(map[string]interface{}); ok {
							if url, ok := src["url"].(string); ok {
								result.PlaylistInfo.Images = url
								break
							}
						}
					}
				}
			}
		}
	}

	// Get tracks
	contentData := getNestedValue(playlist, "content.items")
	if contentData == nil {
		contentData = getNestedValue(playlist, "tracks.items")
	}

	if items, ok := contentData.([]interface{}); ok {
		result.PlaylistInfo.TotalTracks = len(items)

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			// Get track data - could be nested under itemV2.data or just data
			trackData := getNestedValue(itemMap, "itemV2.data")
			if trackData == nil {
				trackData = getNestedValue(itemMap, "data")
			}
			if trackData == nil {
				trackData = getNestedValue(itemMap, "track")
			}
			if trackData == nil {
				continue
			}

			track, ok := trackData.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract track ID from URI
			trackID := ""
			if uri, ok := track["uri"].(string); ok && strings.HasPrefix(uri, "spotify:track:") {
				trackID = strings.TrimPrefix(uri, "spotify:track:")
			}
			if trackID == "" {
				if id, ok := track["id"].(string); ok {
					trackID = id
				}
			}
			if trackID == "" {
				continue
			}

			trackInfo := SpotifyWebPlaylistTrack{
				ID:        trackID,
				SpotifyID: trackID,
			}

			if name, ok := track["name"].(string); ok {
				trackInfo.Name = name
			}

			// Get artists
			var artists []string
			if artistsData := getNestedValue(track, "artists.items"); artistsData != nil {
				if arr, ok := artistsData.([]interface{}); ok {
					for _, a := range arr {
						if artist, ok := a.(map[string]interface{}); ok {
							// Check profile.name or just name
							if profile := getNestedValue(artist, "profile.name"); profile != nil {
								if name, ok := profile.(string); ok {
									artists = append(artists, name)
								}
							} else if name, ok := artist["name"].(string); ok {
								artists = append(artists, name)
							}
						}
					}
				}
			}
			trackInfo.Artists = strings.Join(artists, ", ")

			// Get album
			if albumData := getNestedValue(track, "albumOfTrack"); albumData != nil {
				if album, ok := albumData.(map[string]interface{}); ok {
					if name, ok := album["name"].(string); ok {
						trackInfo.AlbumName = name
					}
					// Get album image
					if coverArt := getNestedValue(album, "coverArt.sources"); coverArt != nil {
						if sources, ok := coverArt.([]interface{}); ok && len(sources) > 0 {
							if src, ok := sources[0].(map[string]interface{}); ok {
								if url, ok := src["url"].(string); ok {
									trackInfo.Images = url
								}
							}
						}
					}
				}
			} else if albumData := getNestedValue(track, "album"); albumData != nil {
				if album, ok := albumData.(map[string]interface{}); ok {
					if name, ok := album["name"].(string); ok {
						trackInfo.AlbumName = name
					}
				}
			}

			// Get duration - extension uses trackDuration.totalMilliseconds for playlists
			if duration := getNestedValue(track, "trackDuration.totalMilliseconds"); duration != nil {
				if d, ok := duration.(float64); ok {
					trackInfo.DurationMS = int(d)
				}
			} else if duration := getNestedValue(track, "duration.totalMilliseconds"); duration != nil {
				if d, ok := duration.(float64); ok {
					trackInfo.DurationMS = int(d)
				}
			} else if duration, ok := track["duration_ms"].(float64); ok {
				trackInfo.DurationMS = int(duration)
			}

			// Get track number
			if num := getNestedValue(track, "trackNumber"); num != nil {
				if n, ok := num.(float64); ok {
					trackInfo.TrackNum = int(n)
				}
			}

			// Get disc number
			if num := getNestedValue(track, "discNumber"); num != nil {
				if n, ok := num.(float64); ok {
					trackInfo.DiscNum = int(n)
				}
			}

			result.Tracks = append(result.Tracks, trackInfo)
		}
	}

	return result, nil
}

// ============================================
// EXPORTED FUNCTIONS
// ============================================

// GetSpotifyWebPlaylist fetches a playlist using internal API
func GetSpotifyWebPlaylist(playlistID string) (string, error) {
	client := GetSpotifyWebClient()
	result, err := client.GetPlaylist(playlistID)
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// SearchSpotifyWeb is the exported function for searching Spotify via internal API
func SearchSpotifyWeb(query string, limit int) (string, error) {
	client := GetSpotifyWebClient()
	results, err := client.Search(query, limit)
	if err != nil {
		return "", err
	}

	// Convert to JSON format expected by bot
	response := struct {
		Tracks []SpotifyWebSearchResult `json:"tracks"`
	}{
		Tracks: results,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
