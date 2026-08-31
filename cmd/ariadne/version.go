package main

import (
	"runtime/debug"
	"strings"
)

// versionInfo answers "what am I running": the CLI's own module version and
// the library version the binary was built against. Both modules release
// separately (vX.Y.Z and cmd/vX.Y.Z), so a user needs both to reason about a
// binary's behavior.
//
// The values come from the build info embedded by the Go toolchain, so they
// are correct for go install, go build, and module-published binaries without
// release-time ldflags. A binary built from a source checkout without module
// versions reports "devel" for that side.
type versionInfo struct {
	CLI     string
	Library string
}

const (
	cliModulePath     = "github.com/xmbshwll/ariadne/cmd"
	libraryModulePath = "github.com/xmbshwll/ariadne"
)

func currentVersion() versionInfo {
	info := versionInfo{CLI: "devel", Library: "devel"}
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		info.CLI = buildInfo.Main.Version
	}
	for _, dep := range buildInfo.Deps {
		switch dep.Path {
		case libraryModulePath:
			// Library versions are module tags like v0.7.0; keep the prefix.
			info.Library = dep.Version
		case cliModulePath:
			info.CLI = strings.TrimPrefix(dep.Version, "v")
		}
	}
	return info
}

// String renders the version line the --version flag prints. The library
// version keeps its "v" prefix so it is copy-pasteable against module tags.
func (v versionInfo) String() string {
	return "ariadne CLI " + v.CLI + " (library " + v.Library + ")"
}
