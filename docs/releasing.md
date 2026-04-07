# Releasing Ariadne

Ariadne is published as two Go modules from one repository:

- library module: `github.com/xmbshwll/ariadne`
- CLI module: `github.com/xmbshwll/ariadne/cmd`

Because the CLI depends on the library, release order matters when both change together.

## Before you tag anything

Run the standard checks:

```bash
make verify
make verify-release
```

These checks confirm that:

- the repository still works in normal workspace-based development
- the root library module passes with `GOWORK=off`
- the `cmd` module passes with `GOWORK=off`
- the CLI still builds from the `cmd` module directly

Also check the working tree:

```bash
git status --short
```

Do not cut release tags from a dirty tree.

## Release scenarios

### Library-only release

Use this when only the root library changed.

1. update `CHANGELOG.md`
2. run verification
3. create and push the root tag

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

This publishes the library module and the corresponding pkg.go.dev version.

### CLI-only release

Use this when only the `cmd` module changed and it does not need a newer library release.

1. confirm `cmd/go.mod` points at the intended library version
2. run verification
3. create and push the CLI submodule tag

```bash
git tag cmd/vX.Y.Z
git push origin cmd/vX.Y.Z
```

This publishes:

- module `github.com/xmbshwll/ariadne/cmd`
- install path `github.com/xmbshwll/ariadne/cmd/ariadne`

### Library and CLI release together

Use this when both modules changed and the CLI should depend on the new library release.

#### Step 1: release the library first

1. update `CHANGELOG.md`
2. run verification
3. tag and push the root module

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

Optionally confirm the version is visible through the module proxy:

```bash
go list -m github.com/xmbshwll/ariadne@vX.Y.Z
```

#### Step 2: switch the CLI module from local development mode to the released library version

During development, `cmd/go.mod` uses a local `replace`:

```go
require github.com/xmbshwll/ariadne v0.0.0
replace github.com/xmbshwll/ariadne => ..
```

For the CLI release, temporarily change that to:

```go
require github.com/xmbshwll/ariadne vX.Y.Z
```

and remove the `replace` line.

Then tidy and verify from the CLI module without the workspace:

```bash
cd cmd
go mod tidy
GOWORK=off go test ./...
GOWORK=off go build ./...
cd ..
```

That is the state the published CLI module should be released from.

#### Step 3: tag the CLI module

```bash
git tag cmd/vX.Y.Z
git push origin cmd/vX.Y.Z
```

#### Step 4: restore local development mode if needed

If you want to keep developing in the split-module workspace layout after the release, restore `cmd/go.mod` to:

```go
require github.com/xmbshwll/ariadne v0.0.0
replace github.com/xmbshwll/ariadne => ..
```

Then commit that change separately on the branch you continue working from.

## Recommended checks after publishing

### Library module

```bash
go list -m github.com/xmbshwll/ariadne@vX.Y.Z
```

### CLI module

```bash
go list -m github.com/xmbshwll/ariadne/cmd@cmd/vX.Y.Z
```

### CLI install path

```bash
go install github.com/xmbshwll/ariadne/cmd/ariadne@latest
```

### pkg.go.dev

Open:

- `https://pkg.go.dev/github.com/xmbshwll/ariadne`

If indexing is slow, request a fetch manually.

## Tag format reminder

- library tag: `vX.Y.Z`
- CLI tag: `cmd/vX.Y.Z`

Do not tag the CLI module with a plain `vX.Y.Z`. Go submodules require the directory prefix in the tag name.
