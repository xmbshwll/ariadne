# Contributing

Thanks for contributing to Ariadne.

This repository contains a public Go library and a CLI built on top of it. The best contributions are small, tested, and clear about which module they affect.

## Repository layout

- `github.com/xmbshwll/ariadne` — root library module
- `github.com/xmbshwll/ariadne/cmd` — CLI module
- `cmd/ariadne` — executable package
- `internal/` — library implementation details
- `docs/` — user and maintainer documentation
- `internal/**/testdata/` — committed CI fixtures for adapter tests

## What you need

- Go 1.26+
- `golangci-lint`
- service credentials only if you plan to run the connector validation tools

## Getting started

Clone the repository, then run the main checks:

```bash
make test
make lint
make verify
```

The repository uses a `go.work` workspace, so local development covers the root library and the `cmd` module together.

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

- `make build` — build the CLI binary
- `make test` — run tests for both modules
- `make test-race` — run race-enabled tests for both modules
- `make lint` — run `golangci-lint` for both modules
- `make verify` — run formatting, linting, and race tests
- `make test-release` — run tests with `GOWORK=off` so each module is checked independently
- `make verify-release` — run the release-oriented split-module checks
- `make deps` — tidy dependencies in both modules

## Working on connectors

Most connector changes should come with at least one of the following:

- unit tests
- updated fixture-backed tests
- refreshed validation artifacts when runtime behavior intentionally changed

Keep the distinction clear:

- committed CI test fixtures belong under package-local `testdata/`
- validation commands should write to an explicit `--out-dir` or their default temp directory

Useful references:

- [`docs/configuration.md`](./docs/configuration.md)
- [`docs/service-resolution.md`](./docs/service-resolution.md)
- [`service-validation-checklist.md`](./service-validation-checklist.md)

### Credential-gated validation tools

These maintainer commands generate recorded artifacts for official integrations.
By default they write to a temporary directory and print the path; pass `--out-dir` if you want to keep the artifacts somewhere specific:

```bash
make validate-spotify-auth
make validate-apple-music-official
make validate-tidal-official
```

They require the corresponding environment variables described in [`docs/configuration.md`](./docs/configuration.md).

Never commit private credentials, `.env`, or `.p8` files.

## Documentation expectations

If you change public behavior, update the relevant docs in the same PR:

- `README.md` for first-time users
- `docs/configuration.md` for env/config changes
- `docs/service-resolution.md` for connector behavior changes
- `CHANGELOG.md` for release-facing summaries
- example tests when public API usage changes

## Pull requests

Before opening a PR:

1. run `make verify`
2. add or update tests for the changed behavior
3. update docs if public behavior changed
4. keep the scope focused

In the PR description, call out:

- which module changed: root library, `cmd`, or both
- whether connector fixtures changed
- whether credentials are required to reproduce anything

## Releases

Ariadne uses separate module versioning for the library and the CLI.

- library tags: `vX.Y.Z`
- CLI tags: `cmd/vX.Y.Z`

For the full release checklist, see [`docs/releasing.md`](./docs/releasing.md).

## Code style

- keep public APIs documented
- prefer focused changes over broad refactors
- follow existing naming and package patterns
- avoid adding new dependencies unless they clearly improve the project

## Questions

If a change affects module layout, release flow, or connector behavior, explain that decision in the PR instead of leaving it implicit.
