# Changelog

All notable changes to Ariadne are documented here.

## v1.0.0 - 2026-09-02

### Added
- first-class library stability with semver-go module versioning (v1.0.0 release)
- consolidated adapter interface contracts with pinned capability sets across all providers
- comprehensive error sentinels for all provider adapters (ErrUnsupported, ErrRuntimeDeferred, and service-specific sentinels)
- generic result types ScoredMatchOf[C], MatchResultOf[C], and ResolutionOf[P, E, C] in the resolve package, re-exported at the library root
- adapterutil.NewClientCredentialsTokenSource owning client-credentials token-source wiring defaults (credentials closure, trim-space empty-token check, singleflight key)

### Changed
- resolved long-standing adapter compatibility issues across Spotify, Apple Music, Deezer, TIDAL, Bandcamp, SoundCloud
- unified entity resolution pipeline across all services (album and song flows consolidated onto generic types)
- improved CLI output consistency and formatting (album/song output forks consolidated)
- public adapter interface aliases UPCSearcher, ISRCSearcher, MetadataSearcher, SongISRCSearcher, SongMetadataSearcher as exported type aliases
- per-request target timeouts no longer abort overall resolution while the context remains active

### Fixed
- fixed adapter initialization race conditions
- resolved metadata parsing edge cases across providers (ISRC/UPC/barcode handling)
- corrected URL handling for boundary service formats (SoundCloud client_id discovery, TIDAL query path migration)
- Bandcamp no longer pre-ranks with hardcoded default weights and truncates to five candidates
- empty autocomplete responses exercise HTML fallback path
- MetadataQueryTargetSearch returns initialized empty slice for empty query sets
- SoundCloud metadata search scans all discovered homepage script assets for client_id
- TIDAL metadata Target Search migrated to current GET /v2/searchResults?filter[query]=... contract after TIDAL retired query-in-path search

## v0.7.0 - 2026-08-13

### Breaking

- Provider Catalog query surface collapsed from fifteen pass-through helpers to five: `LookupServiceName`, `Describe`, `DescribeEnabled`, `EvaluateTarget(config, name, entity)`, `TargetServices(config, entity)`, plus the unchanged `SupportsRuntimeSongInputURL`. The album/song/any axis moved from function names into the new exported `EntityShape` parameter; `DescribeService`, `DescribeEnabledService`, `EvaluateTargetServiceRequest`, `EvaluateConfiguredTargetService`, `EvaluateSongTargetService`, and all `Supports*`/`Supported*`/`Enabled*` helpers are removed — capability questions are answered through `Describe` fields.
- Resolver constructor matrix collapsed from six constructors to two: `New(config, ...Option)` (with `WithHTTPClient`) and `NewWithAdapters(AdapterSet{...})`. `NewWithClient`, `NewWithAdaptersAndWeights`, `NewWithEntityAdapters`, and `NewWithEntityAdaptersAndWeights` are removed; adapter lists and ranking weights now travel in one `AdapterSet` struct with zero-value weight defaults.

### Added

- Per-target failure resilience: a failing Target Search no longer aborts the resolution or cancels sibling searches. `MatchResult` and `SongMatchResult` carry the failure in a new exported `Err` field while other services resolve normally; the CLI prints a warning per failed service and exits non-zero only when every target failed, wrapping the underlying errors so `errors.Is` still finds them.
- Generic result types `ScoredMatchOf[C]`, `MatchResultOf[C]`, and `ResolutionOf[P, E, C]` in the resolve package, re-exported at the library root. Existing names (`MatchResult`, `SongMatchResult`, `Resolution`, ...) are aliases to the album/song instantiations — no caller-visible change.
- `adapterutil.NewClientCredentialsTokenSource` owning the client-credentials token-source wiring defaults (credentials closure, trim-space empty-token check, singleflight key) for credentialed Music Service adapters.

### Fixed

- TIDAL metadata Target Search migrated to the current `GET /v2/searchResults?filter[query]=...` contract after TIDAL retired query-in-path search (`INVALID_RESOURCE_ID` on every metadata search); track fetches no longer request the removed `coverArt` include, and song artwork resolves through the `albums.coverArt` include chain.
- Bandcamp adapter no longer pre-ranks with hardcoded default weights and truncates to five candidates: Score Signal ranking lives only in Entity Resolution, so configured weights now govern Bandcamp matches.

### Changed

- Entity Resolution de-generalized: album and song pipelines are concrete top-to-bottom flows in their own Entity Resolution Policy modules instead of one generic orchestrator with seven function parameters.
- `adapterutil` Metadata Query collection consolidated to one module (`MetadataQuerySearch`); duplicate collector and query-alias helpers removed.
- CLI output album/song forks (strength filtering, match mapping, link building, CSV rows and writers) consolidated onto the generic result types; per-entity policy differences stay in small wrappers.
- Test suites converted to table-driven form where tests shared a shape.

## v0.6.2 - 2026-05-12

### Fixed

- Credential token refresh now retries transient HTTP `502` / `503` / `504` responses from token endpoints with exponential backoff (3 attempts, 250ms initial). Prevents "context deadline exceeded" or "unexpected token status" errors when auth APIs are temporarily unavailable.

### Changed

- HTTP status errors now carry a typed `HTTPStatusCode()` method via the `HTTPStatusError` interface, replacing fragile string-based status code matching with `errors.As` checks.
- All raw HTTP status code integers and string-based status checks replaced with `net/http` constants (`http.StatusServiceUnavailable`, `http.StatusGatewayTimeout`, etc.).

## v0.6.1 - 2026-05-12

### Fixed

- Spotify album source hydration now retries transient Web API `502` / `503` / `504` responses and skips transient per-track detail failures instead of failing the whole album when optional ISRC enrichment is flaky

## v0.6.0 - 2026-05-12

### Added

- public adapter interface aliases `UPCSearcher`, `ISRCSearcher`, `MetadataSearcher`, `SongISRCSearcher`, and `SongMetadataSearcher` as exported type aliases for internal resolve types
- `MetadataQueryTargetSearch` generic helper in `adapterutil` that owns query iteration, candidate identity, hydration, deduplication, limits, and continuation policy
- explicit album and song Entity Resolution Policy modules behind the shared resolver core, unifying orchestration while keeping entity-specific Source Input, Target Search, ranking, result construction, and album Identifier Enrichment behavior local
- dedicated `internal/targetsearch` package for Target Search error classification so adapter-specific availability failures and recoverable request timeouts share one vocabulary
- Provider Catalog as source of truth for service aliases, capabilities, parse-only song input wiring, credential-gated adapter construction, and default resolver ordering
- Bandcamp fuzzy autocomplete endpoint as first search attempt before HTML fallback, preserving resolution when public search page serves a client challenge
- Amazon Music and YouTube Music registered as parse-only song sources so song Source Input reaches deferred Runtime Hydration sentinels instead of falling through as unsupported URLs
- service lookup key normalization through `servicesByLookupKey` map in the Provider Catalog, enabling reliable alias-based service name resolution without external lookup helpers
- default CLI module dependency on released `v0.5.0` library and workspace fixes for local development against the current root checkout

### Changed

- public adapter interfaces (`SourceAdapter`, `SongSourceAdapter`, `TargetAdapter`, `SongTargetAdapter`) changed to type aliases for their internal `resolve` counterparts, removing the public/internal adapter conversion bridge layer
- Apple Music, Spotify, and Deezer metadata Target Search paths now share the `MetadataQueryTargetSearch` helper, keeping only service-specific search endpoint and hydration logic in each adapter
- Bandcamp target search pipeline moved behind a small internal module so album and song metadata search share autocomplete discovery, HTML fallback, hydration, and collection instead of wiring callbacks at each call site
- Target Search Plan collects per-layer results explicitly instead of mixing layer execution, recoverable timeout handling, filtering, fatal error wrapping, and deduplication in one loop
- entity resolution callback pipeline replaced with explicit album and song flows; shared Source Input, Target Search Plan, concurrent search, and Apple Music Identifier Enrichment modules remain in place
- `Resolver.New` constructor now takes adapter slices directly, eliminating `resolveSourceAdapters`, `resolveSongSourceAdapters`, `resolveTargetAdapters`, and `resolveSongTargetAdapters` conversion wrappers
- removed placeholder `public_types.go` file; API types now live in focused files as documented
- Deezer and Apple Music `Search`/`BuildCandidate` closure signatures accept `context.Context` for the shared metadata search path
- SoundCloud `client_id` discovery failures now treated as unavailable search results instead of aborting multi-service resolution
- private/global-heavy tests replaced with narrower behavior-level seams across adapter contract and resolver constructor tests
- workspace `replace` directive removed from `go.work`; CLI module pinned to published library version `v0.5.0`

### Fixed

- Bandcamp album and song targets now resolve when the public search page serves a client challenge, by falling back to hydrated JSON-LD candidates from autocomplete results
- Song targets stay non-fatal when target adapters do not implement song search interfaces; those adapters now produce zero Target Search layers instead of panicking during resolver construction
- `MetadataQueryTargetSearch` returns an initialized empty slice for empty query sets, preserving caller expectations around nil-vs-initialized semantics
- empty autocomplete responses genuinely exercise the HTML fallback path in Bandcamp search, covered by a focused regression test
- Bandcamp autocomplete fetch errors wrapped before returning, satisfying `wrapcheck` lint enforcement
- per-request target timeouts no longer abort overall resolution while the context remains active
- release verification ignores stale vendor state when using the temporary replace modfile during release testing

## v0.5.0 - 2026-05-03

### Added

- public `ErrRuntimeDeferred` sentinel for parseable services whose runtime hydration is intentionally deferred, with service-specific Amazon Music and YouTube Music sentinels for narrower `errors.Is` branches
- Provider Catalog request-decision APIs that distinguish available, unknown, unsupported, parse-only, and credential-gated target service requests

### Changed

- CLI target-service parsing and song-target validation now use Provider Catalog decisions, keeping alias lookup, credential checks, parse-only handling, and unsupported-target errors aligned
- internal resolver orchestration now runs through shared Entity Resolution Policy, Source Input, Target Search Plan, Identifier Enrichment, and Score Signal modules for stronger locality without changing public resolution behavior
- Music Service adapters now share Metadata Query collection, Page Extraction, HTTP exchange, Credential Token, and URL-path helpers while keeping service-specific search, hydration, and canonical mapping logic local
- built-in Music Service support is declared through a clearer Provider Catalog binding grammar that keeps capabilities, runtime URL parsing, credential gating, and Adapter wiring together
- public model and result types now use focused type-alias files instead of generated bridge conversions, preserving exported names while removing pass-through conversion layers

### Fixed

- Amazon Music Source Input now registers as a deferred Runtime Hydration source, returning `ErrRuntimeDeferred` / `ErrAmazonMusicDeferred` instead of `ErrUnsupportedURL`
- Apple Music URL parsing now validates storefront path segments and escapes path components for non-ASCII or reserved characters
- Page Extraction regex helpers now accept patterns with multiple capture groups while preserving not-found behavior when no group exists
- shared HTTP exchange now accepts all successful 2xx responses instead of only `200 OK`
- Apple Music Identifier Enrichment now preserves source track fields while copying only missing UPC/ISRC identifiers, and keeps stronger existing Apple Music matches when cascaded search is weaker

## v0.4.4 - 2026-04-23

### Changed

- rewrote the main user and maintainer docs for clearer CLI usage, configuration, service-resolution, contribution, and release guidance
- simplified recent CLI and validation code paths without changing behavior, reducing duplication in help text, service parsing, validation runners, and validation artifact writing

### Fixed

- validation commands now honor Ariadne's configured HTTP timeout for Apple Music, Spotify, and TIDAL API requests instead of relying on `http.DefaultClient`
- Apple Music official validation now searches all sampled track ISRCs, merges the results, and reports optional artifacts based on files that were actually written
- Spotify and Apple Music validation commands now parse sample URLs before creating output directories, avoiding leaked temp directories on invalid input
- CLI parsing now handles nil unknown-command errors safely and ignores empty segments in `--services` lists such as `spotify,,tidal`
- TIDAL validation artist-name lookup now ignores non-artist included resources when relationship IDs collide across resource types

## v0.4.3 - 2026-04-14

### Fixed

- Deezer album hydration now accepts album payloads that provide inline `tracks.data` but omit the `tracklist` URL, preventing valid Deezer matches like Saosin's `Starting Over Again` from being dropped during resolution

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
