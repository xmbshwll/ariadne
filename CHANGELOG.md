# Changelog

All notable changes to Ariadne are documented here.

## Unreleased

## v0.4.2 - 2026-04-13

### Fixed

- SoundCloud metadata search now scans all discovered homepage script assets when extracting the transient web `client_id`, avoiding failures when SoundCloud moves the token later in the asset list

## v0.4.1 - 2026-04-13

### Added

- CLI log levels through `--log-level` and `ARIADNE_LOG_LEVEL`, including debug output for effective config values during troubleshooting

### Changed

- Spotify track hydration now uses parallel single-track `/v1/tracks/{id}` requests instead of the deprecated batch track endpoint
- normal successful CLI runs stay quiet unless debug logging is explicitly enabled
- CLI help and docs now include the logging flag in command examples and configuration guidance

### Fixed

- Spotify album and song source hydration no longer fails when the deprecated `Get Several Tracks` endpoint returns `403 Forbidden`
- Bandcamp URL parsing now rejects non-Bandcamp hosts, preventing unrelated `/track/...` URLs from being misclassified as Bandcamp sources

## v0.4.0 - 2026-04-10

### Added

- first-class song resolution across Spotify, Apple Music, Deezer, TIDAL, Bandcamp, and SoundCloud
- generic library entry point via `Resolver.Resolve(...)` alongside explicit `ResolveSong(...)`
- metadata-first second-wave song support for Bandcamp and SoundCloud
- YouTube Music song URL parsing during second-wave evaluation work

### Changed

- the CLI now uses `ariadne resolve [--song|--album] <url>` instead of a separate `resolve-song` command
- `ariadne resolve` now defers entity auto-detection to the library when no entity flag is provided
- public docs, examples, and service support tables now cover album and song resolution separately
- the CLI now supports an overall `--resolution-timeout` separate from the per-request `--http-timeout`

### Fixed

- `Resolver.ResolveAlbum(...)` and `Resolver.ResolveSong(...)` now fail safely for nil or partially initialized resolvers instead of panicking
- song source adapters that incorrectly return `(nil, nil)` now produce a descriptive resolver error instead of triggering nil dereferences
- `--min-strength` filtering now prunes weak alternates for album output as well as song output
- service-name normalization now accepts canonical public names like `appleMusic` and `youtubeMusic` reliably in addition to aliases
- verbose YAML CLI output now uses the same explicit snake_case field names as JSON output
- `--config ""` now disables config-file loading cleanly when passed as a separate CLI token
- empty Spotify and TIDAL song metadata searches now short-circuit before credential checks instead of failing unnecessarily
- SoundCloud track canonicalization no longer invents album-artist metadata when the source payload has no album title

### Limitations

- YouTube Music song runtime resolution is still not implemented; only URL parsing is currently available

## v0.3.1 - 2026-04-09

### Added

- README documentation describing the matching pipeline, scoring signals, and confidence bands

### Changed

- metadata search now tries alternate album title variants, including parenthetical alternates and stripped title forms
- improved Spotify and Apple Music resolution for releases whose source titles use mixed-script or parenthetical naming such as `ΘΕΛΗΜΑ (Thelema)`
- added test coverage for title-search variants and adapter metadata query generation

## v0.3.0 - 2026-04-07

### Added

- configurable per-request HTTP timeout through library config, environment, and CLI flags
- committed package-local `testdata` fixtures for SoundCloud and YouTube Music adapter tests
- cmd-local validation helpers for sample URL loading and output directory handling
- parallel target-service resolution to reduce end-to-end resolve latency

### Changed

- improved cross-service matching for compound artist credits such as `A + B` and `A feat. B`
- cleaned up CLI error output so the root underlying error is shown instead of repeated wrapper prefixes
- moved CI-critical test fixtures out of ignored `service-samples` paths and into committed package `testdata`
- changed validation commands to require explicit sample input and write to temporary directories by default unless `--out-dir` is provided
- clarified contributor and configuration docs around test fixtures, validation artifacts, and timeout configuration

## v0.2.0 - 2026-04-07

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
