# Ariadne Agent Guide

## Responsibility

This repository contains the Go package `ariadne`, a music metadata resolution
library. Agent instructions live in `AGENTS.md`; keep generated paths, lint
config, and documentation aligned with this file.

## Folder Map

- `*.go`: root-package implementation.
- `internal/config/`: project-local `.ariadne/config.yml` normalization for
  runtime credentials and later provider-specific settings.
- `internal/applemusicauth/`: Apple Music Media API JWT exchange.
- `internal/normalize/`: shared canonical text, ISRC, UPC, duration, release
  date, artist, and title normalization used by adapters and Target Search.
- `internal/adapters/adapterutil/`: shared provider adapter helpers for
  metadata query fan-out, ordered concurrent collection, HTTP status handling,
  and deferred Candidate Hydration.
- `internal/resolve/`: Entity Resolution pipeline (Source Input recognition,
  Runtime Hydration, Target Search, ranking, Candidate Hydration) shared by the
  root package through type aliases.
- `internal/wiring/`: the Provider Catalog. Which Music Service can act as
  Source Adapter or Target Search adapter, under which Credential Token, in
  which order, plus the built-in adapter construction. The root package only
  re-exports these decisions.
- `internal/model/`: canonical entity, candidate, and service-name types shared
  by every layer, including Candidate SearchKey rules.
- `internal/httpx/`: shared HTTP client construction.
- `internal/targetsearch/`: album and song Target Search plans that assemble
  service-specific Query Policies and per-query Score Signal weights.
- `internal/score/`: deterministic scoring of candidate albums against source
  album metadata.
- `internal/mocks/`: generated Mockery adapter mocks.
- `cmd/go.mod`: separate module `github.com/xmbshwll/ariadne/cmd` with
  `cmd/ariadne/` CLI.
Read `docs/service-resolution.md` and `CONTEXT.md` before changing resolution
behavior, service support, adapter interfaces, Target Search, Candidate
Hydration, or resolution metadata. They are the contract for those decisions.

## Commands

- `make build` builds the CLI into `bin/ariadne`.
- `make test` runs unit tests for this package.
- `make test-coverage` runs the full test suite with coverage output.
- `make lint` runs `golangci-lint` for the root package and `cmd`.
- `make verify` runs formatting, lint, and race tests (the pre-commit gate).
- `make mocks` regenerates Mockery adapter mocks.
- `go build ./...` builds this package.
- `go test ./...` runs Go unit tests.
- `go test ./... -count=1` reruns tests without cached results.
- `go test ./... -coverprofile=coverage.out` writes coverage data.
- `cd cmd && go run ./ariadne resolve <url> --target <service> --dry-run`.
- `cd cmd && go run ./ariadne config services`.
- `cd cmd && go test ./...` tests the CLI module.
- `cd cmd && go run ./validate-spotify-auth --help` runs the private Spotify
  verification command.
- `cd cmd && go run ./validate-apple-music-official --help` runs the private
  Apple Music MusicKit verification command.
- `cd cmd && go run ./validate-tidal-official --help` runs the private TIDAL
  official API verification command.
- `make build` builds the CLI to `bin/ariadne`.
- `go run ./cmd/validate-spotify-auth --help` will fail because `cmd` is a
  separate module.

## Style

- Use idiomatic Go: short packages, explicit errors, small interfaces, table
  tests where useful.
- Use `internal/assert` when an assertion helper keeps a test table readable.
- Use table-driven `tests := []struct{...}` loops with `t.Run` when the same
  check applies across several resolution or config cases.
- Use explicit case structs for CLI parsing, config normalization, adapter JSON,
  and resolver matching/scoring; tables keep related fixtures aligned and
  readable.
- Use shared test fixtures for repeated service, album, track, candidate, config,
  and HTTP fixture data instead of repeating similar literals.
- CLI tests stub the resolver through the narrow `entityResolver` behavior
  interface and describe the `Resolution` or `SongResolution` they want
  rendered; they do not assemble an Entity Resolution pipeline. The resolver's
  own behavior is proven in `internal/resolve` and `ariadne_test.go`.
- Use table-driven tests for Target Search and Candidate Hydration across
  ISRC/UPC/metadata layers, HTTP status codes, API errors, malformed responses,
  nil/empty payloads, weak candidates, and missing optional metadata such as
  versions, copyrights, labels, genres, audio profiles, external IDs, URLs, and
  artist fields.
- Every provider implements the one `internal/adapters.Adapter` interface and
  declares what it supports through `Capabilities()`. Callers select by
  capability and by Service Identity; they never type-assert an adapter to a
  narrower interface, and an unimplemented method returns
  `adapters.ErrUnsupported`. Providers embed `internal/adapters/base.Unsupported`
  and write only the methods their API really offers.
- Each provider package ships an `adapter_contract_test.go` driven by
  `internal/adapters/adaptertest`: it pins `Service()`, pins `Capabilities()`
  against a literal expectation, and proves every undeclared method answers
  `adapters.ErrUnsupported`. Editing what a provider supports means editing that
  literal on purpose.
- Mockery generates one testify-compatible mock, `MockAdapter`, from
  `internal/adapters.Adapter` into `internal/mocks` as `xxx_mock.go`. Run
  `make mocks` after Adapter interface changes; the one mock satisfies every
  Source Input and Target Search layer assertion.
- The public package exposes no adapter seam: `ariadne.New` plus options is the
  only construction, and `internal/wiring` chooses adapters from the Provider
  Catalog. Tests that need specific adapters use the `export_test.go` seam.
- Every exported identifier in the public `package ariadne` carries a doc
  comment: that package is the library contract. Lint does not enforce this
  (`revive:exported` and `revive:package-comments` are disabled in
  `.golangci.yml`), so it stays true by hand.
- Under `internal/`, document anything that carries domain meaning (services,
  Entity Shapes, Scoring, Target Search layers); provider wire DTOs that only
  mirror a provider JSON payload may stay uncommented.
- Generated mock files use `_test.go` or `mocks/` naming and stay separate from
  production code.
- HTTP clients must close response bodies and use `context.Context` for network
  calls.
- Keep the public package API small: exported types and interfaces should
  describe resolution concepts, not internal HTTP or OAuth plumbing.
- The public package root stays `package ariadne`. Provider implementations
  live under `internal/adapters/<provider>/`.
- Internal `cmd` validation utilities may return structured issues from parse
  and validation helpers so CLI commands can render text and JSON output
  consistently.
- Internal `cmd` adapters and validation artifact types may expose fields
  needed by CLI renderers; the public `ariadne` package still exposes only
  resolution-facing types such as `Resolution`, `SongResolution`, `Candidate`,
  and `Match`.
- CLI JSON output uses one wrapper shape, `{"error": ...}` or
  `{"result": ...}`, for JSON-only commands and command errors.

## Testing Guidelines

- Test files use the external `<package>_test` package so the suite exercises
  the exported interface. White-box tests over unexported wiring are the
  exception and stay in-package; prefer a narrow `export_test.go` seam over
  widening the public `ariadne` API. `cmd` binaries are `package main`, which
  Go cannot import from an external test package, so those tests stay
  in-package.
- Use the standard Go test toolchain and `make test-coverage`.
- Root-package tests cover behavior and integration, not implementation
  details: validation, service normalization, duplicate/empty sets, adapter
  routing, Target Search selection, Candidate Hydration, weak matches, no
  matches, and error wrapping.
- Prefer `github.com/stretchr/testify/assert` and
  `github.com/stretchr/testify/require` over handwritten `if got != want`
  assertions when both assertions and JSON-equivalent fixtures can cover the
  behavior.
- Table tests must include assertion messages with the case name, and fixtures
  must be table-driven with explicit fields for `config.yml` case, CLI case,
  Target Search case, scoring case, adapter case, and resolution case where
  relevant.
- Keep tests meaningful: cover Target Search, Candidate Hydration, scoring,
  aliases, duplicate/empty sets, adapter search paths, and error paths.
- Tests must assert exact public error strings and error codes for failing
  source inputs, adapters, missing optional data, unsupported services,
  credential failures, HTTP errors, malformed responses, nil/empty payloads,
  weak candidates, and missing external IDs, copyrights, labels, genres, audio
  profiles, versions, and URLs.
- Never test only the happy path for adapters. Adapter tests must prove
  credential precedence, query fan-out, malformed responses, HTTP status
  failures, malformed provider payloads, missing optional data, and unsupported
  service errors.
- Never assert only that Target Search or Candidate Hydration returns no error.
  Assert the returned candidates, scores, confidence, hydration state, and
  preserved metadata across `Album`, `Song`, `Candidate`, `Match`,
  `Resolution`, and `SongResolution`.
- Use `httptest` for provider HTTP behavior, and keep provider adapter tests
  aligned with the public interface rather than internal HTTP structs.

## Notes

- `go test ./...` must pass for every change.
- Root-package tests run with race detection in CI, so keep fixtures isolated
  per test.
