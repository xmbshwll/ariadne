package main

// CLI infrastructure: the leveled logger and the --version report.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/xmbshwll/ariadne"
)

var errUnsupportedLogLevel = errors.New("unsupported log level")

type logLevel int

const (
	logLevelError logLevel = iota
	logLevelWarn
	logLevelInfo
	logLevelDebug
)

const defaultCLILogLevel = logLevelError

type cliLogger struct {
	out   io.Writer
	level logLevel
}

func (level logLevel) String() string {
	switch level {
	case logLevelError:
		return "error"
	case logLevelWarn:
		return "warn"
	case logLevelInfo:
		return "info"
	case logLevelDebug:
		return "debug"
	default:
		return defaultCLILogLevel.String()
	}
}

func parseLogLevel(raw string) (logLevel, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return defaultCLILogLevel, nil
	}

	switch normalized {
	case logLevelError.String():
		return logLevelError, nil
	case logLevelWarn.String():
		return logLevelWarn, nil
	case logLevelInfo.String():
		return logLevelInfo, nil
	case logLevelDebug.String():
		return logLevelDebug, nil
	default:
		return defaultCLILogLevel, fmt.Errorf("%w %q (expected error, warn, info, or debug)", errUnsupportedLogLevel, raw)
	}
}

func newCLILogger(out io.Writer, level logLevel) *cliLogger {
	if out == nil {
		out = io.Discard
	}
	return &cliLogger{out: out, level: level}
}

func newCLILoggerFromArgs(args []string, out io.Writer) (*cliLogger, error) {
	raw, _ := namedFlagValueFromArgs(args, "--log-level")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("ARIADNE_LOG_LEVEL"))
	}
	level, err := parseLogLevel(raw)
	if err != nil {
		return nil, err
	}
	return newCLILogger(out, level), nil
}

func namedFlagValueFromArgs(args []string, name string) (string, bool) {
	for i, arg := range args {
		switch {
		case arg == name:
			if i+1 >= len(args) {
				return "", true
			}
			value := args[i+1]
			if strings.HasPrefix(value, "-") {
				return "", true
			}
			return value, true
		case strings.HasPrefix(arg, name+"="):
			value, _ := strings.CutPrefix(arg, name+"=")
			return value, true
		}
	}
	return "", false
}

func (l *cliLogger) enabled(level logLevel) bool {
	return l != nil && level <= l.level
}

func (l *cliLogger) logf(level logLevel, format string, args ...any) {
	if !l.enabled(level) {
		return
	}
	_, _ = fmt.Fprintf(l.out, "%s %s\n", strings.ToUpper(level.String()), fmt.Sprintf(format, args...))
}

func (l *cliLogger) Debugf(format string, args ...any) {
	l.logf(logLevelDebug, format, args...)
}

func (l *cliLogger) Warnf(format string, args ...any) {
	l.logf(logLevelWarn, format, args...)
}

func logRawCLIConfigValues(logger *cliLogger, lookup func(string) string) {
	if lookup == nil {
		return
	}

	logger.Debugf(
		"effective config SPOTIFY_CLIENT_ID=%q SPOTIFY_CLIENT_SECRET=%q APPLE_MUSIC_STOREFRONT=%q APPLE_MUSIC_KEY_ID=%q APPLE_MUSIC_TEAM_ID=%q APPLE_MUSIC_PRIVATE_KEY_PATH=%q TIDAL_CLIENT_ID=%q TIDAL_CLIENT_SECRET=%q ARIADNE_HTTP_TIMEOUT=%q ARIADNE_TARGET_SERVICES=%q",
		lookup("SPOTIFY_CLIENT_ID"),
		lookup("SPOTIFY_CLIENT_SECRET"),
		lookup("APPLE_MUSIC_STOREFRONT"),
		lookup("APPLE_MUSIC_KEY_ID"),
		lookup("APPLE_MUSIC_TEAM_ID"),
		lookup("APPLE_MUSIC_PRIVATE_KEY_PATH"),
		lookup("TIDAL_CLIENT_ID"),
		lookup("TIDAL_CLIENT_SECRET"),
		lookup("ARIADNE_HTTP_TIMEOUT"),
		lookup("ARIADNE_TARGET_SERVICES"),
	)
}

func logNormalizedCLIConfig(logger *cliLogger, cfg ariadne.Config) {
	wiringConfig := toWiringConfig(cfg)
	logger.Debugf(
		"normalized config spotify_enabled=%t apple_music_auth_enabled=%t tidal_enabled=%t storefront=%q http_timeout=%s target_services=%q",
		wiringConfig.Spotify.Enabled(),
		wiringConfig.AppleMusic.AuthEnabled(),
		wiringConfig.TIDAL.Enabled(),
		cfg.AppleMusicStorefront,
		cfg.HTTPTimeout,
		strings.Join(serviceNames(cfg.TargetServices), ","),
	)
}

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
