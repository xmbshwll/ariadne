package ariadne_test

import (
	"fmt"
	"net/http"

	"github.com/xmbshwll/ariadne"
)

func ExampleDefaultConfig() {
	cfg := ariadne.DefaultConfig()

	fmt.Println(cfg.AppleMusicStorefront)
	fmt.Println(cfg.SpotifyEnabled())
	fmt.Println(cfg.TIDALEnabled())
	// Output:
	// us
	// false
	// false
}

func ExampleLoadConfigFromEnv() {
	cfg := ariadne.LoadConfigFromEnv(func(key string) string {
		switch key {
		case "APPLE_MUSIC_STOREFRONT":
			return "GB"
		case "SPOTIFY_CLIENT_ID":
			return "spotify-client"
		case "SPOTIFY_CLIENT_SECRET":
			return "spotify-secret"
		case "TIDAL_CLIENT_ID":
			return "tidal-client"
		case "TIDAL_CLIENT_SECRET":
			return "tidal-secret"
		default:
			return ""
		}
	})

	fmt.Println(cfg.AppleMusicStorefront)
	fmt.Println(cfg.SpotifyEnabled())
	fmt.Println(cfg.TIDALEnabled())
	// Output:
	// gb
	// true
	// true
}

func ExampleConfig_targetServices() {
	cfg := ariadne.DefaultConfig()
	cfg.TargetServices = []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}

	fmt.Println(cfg.TargetServices[0])
	fmt.Println(cfg.TargetServices[1])
	// Output:
	// spotify
	// appleMusic
}

func ExampleMatchStrengthForScore() {
	fmt.Println(ariadne.MatchStrengthForScore(55))
	fmt.Println(ariadne.MatchStrengthForScore(85))
	// Output:
	// weak
	// probable
}

func ExampleNew_withHTTPClient() {
	resolver := ariadne.New(ariadne.DefaultConfig(), ariadne.WithHTTPClient(&http.Client{}))

	fmt.Println(resolver != nil)
	// Output:
	// true
}

// The remaining Resolver behavior — Entity Resolution over a source adapter and
// Target Search over a target adapter, plus custom ranking weights — is covered
// by the package tests, which build adapters from the internal adapter
// interface. It is not shown as an example here because authoring an adapter is
// no longer part of the public API: callers get the built-in Provider Catalog
// adapters through New and customize behavior with Config and WithHTTPClient.
