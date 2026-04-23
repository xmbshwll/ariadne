package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/xmbshwll/ariadne"
)

const resolveHelpText = `Resolve a supported music URL across music services.

Usage:
  ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--resolution-timeout=20s] <url>

Positional parameter:
  <url>
    Required.
    Values: a supported album URL from Apple Music, Deezer, Spotify, TIDAL,
    SoundCloud, YouTube Music, Bandcamp, or Amazon Music, or a supported song
    URL from Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, or TIDAL.
    Behavior: when neither --song nor --album is set, Ariadne asks the library
    to auto-detect the resource type from the URL.
    Amazon Music URLs are recognized for parsing, but runtime resolution remains deferred.

Flags:
  --config
    Values: empty string to disable file loading, or a path to a config file.
    Supported file styles: .env-style key=value files, plus Viper-supported structured files such as yaml, yml, json, or toml.
    Default: %s
    Behavior: config file values are loaded first, environment variables override them, and explicit CLI flags override both.

  --log-level
    Values: error, warn, info, debug.
    Default: error.
    Environment override: ARIADNE_LOG_LEVEL.
    Behavior: writes CLI diagnostics to stderr. debug prints effective configuration, including secrets loaded from env or config files.

  --song
    Forces song resolution for the provided URL.
    Mutually exclusive with --album.

  --album
    Forces album resolution for the provided URL.
    Mutually exclusive with --song.

  --verbose, -v
    Values: true, false.
    Default: false.
    false prints compact service-link output only.
    true includes source metadata, per-service summaries, scores, reasons, and alternates.

  --format
    Values:
      json  - indented JSON; best default for scripts and APIs.
      yaml  - YAML rendering of the same payload.
      csv   - compact or verbose CSV depending on --verbose.
    Default: json.

  --services
    Values: comma-separated list drawn from appleMusic, bandcamp, deezer, soundcloud, spotify, tidal, youtubeMusic, ytmusic.
    ytmusic is an alias for youtubeMusic.
    Use this to limit which target services are searched.
    Caveats:
      spotify requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET.
      tidal requires TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET.
      amazonMusic is not a valid target service.

  --min-strength
    Values:
      very_weak - include every retained match.
      weak      - exclude very weak matches.
      probable  - show only stronger likely matches.
      strong    - show only highest-confidence matches.
    Default: very_weak.

  --apple-music-storefront
    Values: an Apple Music storefront country code in ISO 3166-1 alpha-2 form, for example us, gb, de, fr, jp, ca, or au.
    Default: %s.
    Used for Apple Music lookups and searches when the source URL does not already imply a storefront.

  --http-timeout
    Values: a Go duration such as 5s, 15s, 30s, or 1m.
    Default: %s.
    Sets the per-request timeout on Ariadne's default HTTP client for service API and page requests.

  --resolution-timeout
    Values: a Go duration such as 20s, 30s, 1m, or 2m.
    Default: 20s.
    Sets the overall timeout for one resolve command across parsing, source fetch, and target searches.

Notes:
  - Spotify target search is enabled only when SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are set.
  - Apple Music UPC and ISRC target search are enabled when APPLE_MUSIC_KEY_ID, APPLE_MUSIC_TEAM_ID, and APPLE_MUSIC_PRIVATE_KEY_PATH are set.
  - TIDAL source fetch and target search require TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET.
  - Song resolution currently supports Apple Music, Bandcamp, Deezer, SoundCloud, Spotify, and TIDAL.`

func renderRootHelp(w io.Writer, baseConfig ariadne.Config, configPath string) error {
	if _, err := io.WriteString(w, rootHelpTextFor(baseConfig, configPath)); err != nil {
		return fmt.Errorf("%w: %w", errRenderResolveHelp, err)
	}
	return nil
}

func resolveHelpTextFor(baseConfig ariadne.Config, configPath string) string {
	if configPath == "" {
		configPath = `"" (disable file loading)`
	}

	storefrontDefault := "APPLE_MUSIC_STOREFRONT or us"
	if baseConfig.AppleMusicStorefront != "" {
		storefrontDefault = baseConfig.AppleMusicStorefront
	}

	return fmt.Sprintf(resolveHelpText, configPath, storefrontDefault, baseConfig.HTTPTimeout)
}

func rootHelpTextFor(baseConfig ariadne.Config, configPath string) string {
	return strings.Join([]string{
		"Usage:",
		"  ariadne <command> [flags]",
		"",
		"Commands:",
		"  resolve  Resolve a supported album or song URL across services.",
		"",
		resolveHelpTextFor(baseConfig, configPath),
	}, "\n")
}
