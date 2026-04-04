# Changelog

## Unreleased

## v0.1.0 - 2026-04-03

### Added
- public `ariadne` Go package for reusable library consumption
- default library resolver wiring for Spotify, Apple Music, Deezer, Bandcamp, SoundCloud, YouTube Music, TIDAL, and deferred Amazon Music URL handling
- official Apple Music identifier search with generated MusicKit tokens from `.p8` credentials
- official TIDAL adapter with client-credentials auth
- experimental SoundCloud adapter using public page hydration and public-facing `api-v2` playlist search
- experimental YouTube Music adapter using public HTML extraction and hydrated metadata search
- parse-only Amazon Music runtime support with explicit deferred errors
- fixture-backed CLI resolve tests
- resolver ranking fixtures for SoundCloud and YouTube Music
- dedicated docs explaining runtime resolution behavior by service

### Supported connectors
- Spotify
- Apple Music
- Deezer

### Experimental connectors
- Bandcamp
- SoundCloud
- YouTube Music
- TIDAL

### Deferred connectors
- Amazon Music

### Notes
- Spotify target search requires app credentials.
- Apple Music identifier search requires `.p8` key material.
- TIDAL source and target runtime support require client credentials.
- SoundCloud and YouTube Music rely on brittle public web extraction and remain experimental by design.
