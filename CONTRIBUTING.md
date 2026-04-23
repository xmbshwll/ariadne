# Contributing

Thanks for contributing to Ariadne.

This repository contains:

- root library module: `github.com/xmbshwll/ariadne`
- CLI module: `github.com/xmbshwll/ariadne/cmd`

Best changes are small, tested, and clear about which module they affect.

## Repository layout

- `cmd/ariadne` — CLI entrypoint
- `internal/` — library implementation details
- `docs/` — user and maintainer documentation
- `internal/**/testdata/` — committed fixtures used by CI

## What you need

- Go `1.26+`
- `golangci-lint`
- service credentials only if you need to run validation commands for Spotify, Apple Music, or TIDAL

## Getting started

Clone repository, then run standard checks:

```bash
make test
make lint
make verify
```

Ariadne uses `go.work`, so local development covers both modules together.

## Common commands

```bash
make build
make test
make test-race
make test-release
make lint
make lint-fix
make fmt
make verify
make verify-release
make deps
```

What they do:

- `make build` — build CLI binary
- `make test` — run tests in both modules
- `make test-race` — run race-enabled tests in both modules
- `make test-release` — run tests with `GOWORK=off` so each module is checked independently
- `make lint` — run `golangci-lint` in both modules
- `make lint-fix` — run lint autofixes where possible
- `make fmt` — run `gofmt`
- `make verify` — run formatting, linting, and race tests
- `make verify-release` — run release-oriented module verification
- `make deps` — tidy dependencies in both modules

## Working on code

A few project expectations:

- keep public behavior covered by tests
- keep diffs focused
- update docs when public behavior changes
- follow existing package and naming patterns
- avoid new dependencies unless they clearly improve project

## Working on connectors

Connector changes should usually come with at least one of these:

- unit tests
- updated fixture-backed tests
- refreshed validation artifacts when runtime behavior intentionally changed

Keep this split clear:

- committed CI fixtures belong in package-local `testdata/`
- validation command output belongs in temp directories or explicit `--out-dir` locations

Useful references:

- [`docs/configuration.md`](./docs/configuration.md)
- [`docs/service-resolution.md`](./docs/service-resolution.md)

## Validation commands

These commands are mainly for integration debugging and connector verification.

```bash
make validate-spotify-auth
make validate-apple-music-official
make validate-tidal-official
```

They write artifacts to a temporary directory by default and print that path. Pass `--out-dir` if you want to keep output.

They require matching credentials from [`docs/configuration.md`](./docs/configuration.md).

Never commit private credentials, `.env`, or `.p8` files.

## Documentation expectations

If public behavior changes, update docs in same PR:

- `README.md` for first-time users
- `docs/configuration.md` for config or credential changes
- `docs/service-resolution.md` for connector behavior changes
- `CHANGELOG.md` for release-facing summaries
- example tests when public API usage changes

## Pull requests

Before opening PR:

1. run `make verify`
2. add or update tests for changed behavior
3. update docs if public behavior changed
4. keep scope focused

In PR description, call out:

- which module changed: library, CLI, or both
- whether connector fixtures changed
- whether credentials are required to reproduce anything

## Releases

Ariadne uses separate module versioning:

- library tags: `vX.Y.Z`
- CLI tags: `cmd/vX.Y.Z`

Full release guide: [`docs/releasing.md`](./docs/releasing.md)

## Questions

If change affects module layout, release flow, or connector behavior, explain decision in PR instead of leaving it implicit.
