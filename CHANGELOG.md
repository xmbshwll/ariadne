# Changelog

All notable changes to Ariadne are documented here.

## Unreleased

### Added

- more public example coverage for `go doc` and pkg.go.dev readers

### Changed

- documented the repository as separate library and CLI Go modules
- expanded the README with clearer installation, usage, configuration, and error-handling guidance
- added a release playbook for split-module publishing in `docs/releasing.md`
- moved the Cobra-based CLI to Viper-backed configuration loading with flag, environment, and config-file precedence
- expanded CLI help text with more detail about flags, parameters, and accepted values
- tightened linting across the repository and updated the codebase to pass stricter checks, including `wrapcheck` and `err113`
- simplified recently touched CLI, parser, adapter, and validation code without changing behavior
- clarified public resolver error handling in package docs and user-facing documentation

## v0.1.0 - 2026-04-03

### Added

- the public `ariadne` Go package for reusable library consumption
- the default resolver wiring for Spotify, Apple Music, Deezer, Bandcamp, SoundCloud, YouTube Music, TIDAL, and deferred Amazon Music URL handling
- official Apple Music identifier search with generated MusicKit tokens from `.p8` credentials
- the official TIDAL adapter with client-credentials auth
- the experimental SoundCloud adapter using public page hydration and public-facing `api-v2` playlist search
- the experimental YouTube Music adapter using public HTML extraction and hydrated metadata search
- parse-only Amazon Music support with explicit deferred errors
- fixture-backed CLI resolve tests
- resolver ranking fixtures for SoundCloud and YouTube Music
- detailed service-resolution documentation

### Stable services

- Spotify
- Apple Music
- Deezer

### Experimental services

- Bandcamp
- SoundCloud
- YouTube Music
- TIDAL

### Deferred services

- Amazon Music

### Notes

- Spotify target search requires app credentials.
- Apple Music identifier search requires `.p8` key material.
- TIDAL source and target runtime support require client credentials.
- SoundCloud and YouTube Music rely on public web extraction and remain experimental by design.
