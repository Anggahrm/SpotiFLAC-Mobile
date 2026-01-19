package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zarz/spotiflac-bot/pkg/backend"
)

func main() {
	// Test playlist URL
	playlistURL := "https://open.spotify.com/playlist/4IOLQFZ586IYvPSUOPZ2Ts?si=qviASNOaTMuavrx9H9uMqw"
	if len(os.Args) > 1 {
		playlistURL = os.Args[1]
	}

	fmt.Printf("Testing playlist URL: %s\n\n", playlistURL)

	// Fetch playlist using existing Spotify API with fallback
	fmt.Println("Fetching playlist via GetSpotifyMetadataWithDeezerFallback...")
	playlistJSON, err := backend.GetSpotifyMetadataWithDeezerFallback(playlistURL)
	if err != nil {
		fmt.Printf("GetSpotifyMetadataWithDeezerFallback failed: %v\n", err)
		
		// Try Spotify Web API
		fmt.Println("\nTrying Spotify Web API...")
		parseResult, parseErr := backend.ParseSpotifyURL(playlistURL)
		if parseErr != nil {
			fmt.Printf("Parse error: %v\n", parseErr)
			os.Exit(1)
		}

		var parsed struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal([]byte(parseResult), &parsed); err != nil {
			fmt.Printf("Parse JSON error: %v\n", err)
			os.Exit(1)
		}

		playlistJSON, err = backend.GetSpotifyWebPlaylist(parsed.ID)
		if err != nil {
			fmt.Printf("Spotify Web API also failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Try parsing as old format first
	var oldFormat struct {
		PlaylistInfo struct {
			Tracks struct {
				Total int `json:"total"`
			} `json:"tracks"`
			Owner struct {
				DisplayName string `json:"display_name"`
				Name        string `json:"name"`
			} `json:"owner"`
		} `json:"playlist_info"`
		Tracks []struct {
			SpotifyID  string `json:"spotify_id"`
			Name       string `json:"name"`
			Artists    string `json:"artists"`
			DurationMS int    `json:"duration_ms"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(playlistJSON), &oldFormat); err == nil && len(oldFormat.Tracks) > 0 {
		fmt.Printf("\n📋 Playlist: %s\n", oldFormat.PlaylistInfo.Owner.Name)
		fmt.Printf("👤 Owner: %s\n", oldFormat.PlaylistInfo.Owner.DisplayName)
		fmt.Printf("🎵 Total tracks: %d\n\n", oldFormat.PlaylistInfo.Tracks.Total)

		fmt.Println("Tracks:")
		for i, track := range oldFormat.Tracks {
			if i >= 15 {
				fmt.Printf("... and %d more tracks\n", len(oldFormat.Tracks)-15)
				break
			}
			duration := track.DurationMS / 1000
			fmt.Printf("%2d. %s - %s (%d:%02d)\n", i+1, track.Name, track.Artists, duration/60, duration%60)
		}
		return
	}

	// Try new format
	var newFormat struct {
		PlaylistInfo struct {
			Name        string `json:"name"`
			Owner       string `json:"owner"`
			TotalTracks int    `json:"total_tracks"`
		} `json:"playlist_info"`
		Tracks []struct {
			SpotifyID  string `json:"spotify_id"`
			ID         string `json:"id"`
			Name       string `json:"name"`
			Artists    string `json:"artists"`
			DurationMS int    `json:"duration_ms"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(playlistJSON), &newFormat); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		fmt.Printf("Raw: %s\n", playlistJSON[:min(500, len(playlistJSON))])
		os.Exit(1)
	}

	fmt.Printf("\n📋 Playlist: %s\n", newFormat.PlaylistInfo.Name)
	fmt.Printf("👤 Owner: %s\n", newFormat.PlaylistInfo.Owner)
	fmt.Printf("🎵 Total tracks: %d\n\n", newFormat.PlaylistInfo.TotalTracks)

	fmt.Println("Tracks:")
	for i, track := range newFormat.Tracks {
		if i >= 15 {
			fmt.Printf("... and %d more tracks\n", len(newFormat.Tracks)-15)
			break
		}
		duration := track.DurationMS / 1000
		fmt.Printf("%2d. %s - %s (%d:%02d)\n", i+1, track.Name, track.Artists, duration/60, duration%60)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
