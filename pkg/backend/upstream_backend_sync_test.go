package backend

import (
	"net/url"
	"strings"
	"testing"
)

func TestUserAgentForURLUsesAppIdentityForZarzAPI(t *testing.T) {
	SetAppVersion("4.2.3")
	t.Cleanup(func() {
		SetAppVersion("")
	})

	u, err := url.Parse("https://api.zarz.moe/v1/test")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	if got := userAgentForURL(u); got != "SpotiFLAC-Mobile/4.2.3" {
		t.Fatalf("expected app user agent, got %q", got)
	}
}

func TestUserAgentForURLFallsBackForOtherHosts(t *testing.T) {
	SetAppVersion("4.2.3")
	t.Cleanup(func() {
		SetAppVersion("")
	})

	u, err := url.Parse("https://song.link/v1-alpha.1/links?url=test")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	got := userAgentForURL(u)
	if strings.HasPrefix(got, "SpotiFLAC-Mobile") {
		t.Fatalf("expected browser user agent for non-zarz host, got %q", got)
	}
	if !strings.Contains(got, "Mozilla/5.0") {
		t.Fatalf("expected browser user agent, got %q", got)
	}
}

func TestTidalTrackAlbumArtistDisplayPrefersMainArtists(t *testing.T) {
	track := &TidalTrack{}
	track.Artist.Name = "Fallback Artist"
	track.Artists = []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{
		{Name: "Guest Artist", Type: "FEATURED"},
		{Name: "Main One", Type: "MAIN"},
		{Name: "Main Two", Type: "MAIN"},
	}

	if got := tidalTrackAlbumArtistDisplay(track); got != "Main One, Main Two" {
		t.Fatalf("expected main artists only, got %q", got)
	}
}

func TestTidalTrackAlbumArtistDisplayFallsBackToPrimaryArtist(t *testing.T) {
	track := &TidalTrack{}
	track.Artist.Name = "Fallback Artist"
	track.Artists = []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{
		{Name: "Guest Artist", Type: "FEATURED"},
	}

	if got := tidalTrackAlbumArtistDisplay(track); got != "Fallback Artist" {
		t.Fatalf("expected fallback artist, got %q", got)
	}
}
