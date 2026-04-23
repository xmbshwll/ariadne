# Releasing Ariadne

Ariadne is published as two Go modules from one repository:

- library module: `github.com/xmbshwll/ariadne`
- CLI module: `github.com/xmbshwll/ariadne/cmd`

If both modules changed, release order matters because CLI depends on library.

## Before any release

Run standard checks:

```bash
make verify
make verify-release
```

Then confirm tree is clean:

```bash
git status --short
```

Do not tag from dirty tree.

## Tag names

- library tag: `vX.Y.Z`
- CLI tag: `cmd/vX.Y.Z`

CLI tags must include `cmd/` prefix. Plain `vX.Y.Z` only releases root library module.

## Release scenarios

### 1. Library-only release

Use this when only root module changed.

Steps:

1. update `CHANGELOG.md`
2. run verification
3. create and push library tag

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

After push, Go module proxy and pkg.go.dev will pick up new library version.

### 2. CLI-only release

Use this when only `cmd` module changed and it should keep depending on already-published library version.

Steps:

1. confirm `cmd/go.mod` points at intended library version
2. run verification
3. create and push CLI tag

```bash
git tag cmd/vX.Y.Z
git push origin cmd/vX.Y.Z
```

This publishes CLI module at:

- module path: `github.com/xmbshwll/ariadne/cmd`
- install path: `github.com/xmbshwll/ariadne/cmd/ariadne`

### 3. Library and CLI release together

Use this when both modules changed and CLI should depend on new library version.

#### Step 1: release library first

1. update `CHANGELOG.md`
2. run verification
3. tag and push root library module

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

Optional check:

```bash
go list -m github.com/xmbshwll/ariadne@vX.Y.Z
```

#### Step 2: update CLI to released library version

Set `cmd/go.mod` to new root version:

```go
require github.com/xmbshwll/ariadne vX.Y.Z
```

Then verify CLI module without workspace:

```bash
cd cmd
go mod tidy
GOWORK=off go test ./...
GOWORK=off go build ./...
cd ..
```

#### Step 3: release CLI module

```bash
git tag cmd/vX.Y.Z
git push origin cmd/vX.Y.Z
```

## Why `make verify-release` exists

`make verify-release` checks release-shaped workflows, not only normal workspace development.

It verifies that:

- root library module passes with `GOWORK=off`
- CLI module can be tested independently
- CLI module can be built independently
- when root changes are not tagged yet, CLI verification can still run against current root checkout through temporary local `replace`

That keeps pre-release verification useful before new library tag exists.

## Post-release checks

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

If indexing is slow, request refresh manually from pkg.go.dev.
