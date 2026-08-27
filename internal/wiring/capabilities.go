// Package wiring owns the Provider Catalog: the single module that decides which
// Music Service can act as Source Adapter or Target Search adapter, under which
// Credential Token, in which order. The public package only re-exports its
// decisions; adapter construction never lives there.
package wiring

import (
	"net/http"

	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
)

// Capabilities describes built-in runtime support for one Music Service.
type Capabilities struct {
	// Aliases are additional names accepted by LookupServiceName.
	Aliases []string
	// SupportsAlbumSource reports whether the service can parse and fetch album source URLs at runtime.
	SupportsAlbumSource bool
	// SupportsAlbumTarget reports whether the service has a built-in album target adapter.
	SupportsAlbumTarget bool
	// SupportsSongSource reports whether the service can parse and fetch song source URLs at runtime.
	SupportsSongSource bool
	// SupportsSongTarget reports whether the service has a built-in song target adapter.
	SupportsSongTarget bool
	// SupportsRuntimeSongInputURL reports whether the built-in runtime song pipeline can parse song URLs for this service.
	SupportsRuntimeSongInputURL bool
}

// EntityShape selects the entity shape a Provider Catalog query applies to.
// The zero value queries any target Capability.
type EntityShape string

const (
	// EntityShapeAny queries services with any target Capability.
	EntityShapeAny EntityShape = ""
	// EntityShapeAlbum queries the album target Capability.
	EntityShapeAlbum EntityShape = "album"
	// EntityShapeSong queries the song target Capability.
	EntityShapeSong EntityShape = "song"
)

// TargetServiceRequestStatus explains whether a requested target service can be used under a config.
type TargetServiceRequestStatus string

const (
	// TargetServiceRequestAvailable means the service can be used as requested.
	TargetServiceRequestAvailable TargetServiceRequestStatus = "available"
	// TargetServiceRequestUnknown means the requested service name or alias is not known.
	TargetServiceRequestUnknown TargetServiceRequestStatus = "unknown"
	// TargetServiceRequestUnsupported means the service is known but does not support the requested target role.
	TargetServiceRequestUnsupported TargetServiceRequestStatus = "unsupported"
	// TargetServiceRequestParseOnly means the service can parse URLs but has no runtime target search capability.
	TargetServiceRequestParseOnly TargetServiceRequestStatus = "parseOnly"
	// TargetServiceRequestCredentialsRequired means the target role needs missing credentials.
	TargetServiceRequestCredentialsRequired TargetServiceRequestStatus = "credentialsRequired"
)

// TargetServiceRequestDecision reports Provider Catalog validation for one requested target service.
type TargetServiceRequestDecision struct {
	Service model.ServiceName
	Status  TargetServiceRequestStatus
	// Message is a human-readable explanation for unavailable decisions.
	Message string
	// CredentialHint names the Credential Token a credential-gated service is
	// missing, set when Status is TargetServiceRequestCredentialsRequired.
	CredentialHint string
}

// DefaultResolverAdapters builds the built-in adapter sets from the Provider
// Catalog bindings and order, limited to targetServices when that list is not
// empty. This is the only wiring entry point the public constructors need.
func DefaultResolverAdapters(client *http.Client, cfg config.Config, targetServices []model.ServiceName) ResolverAdapters {
	return buildResolverAdapters(client, cfg, defaultBindings, defaultOrder, targetServices)
}

func spotifyEnabled(cfg config.Config) bool {
	return cfg.Spotify.Enabled()
}

func tidalEnabled(cfg config.Config) bool {
	return cfg.TIDAL.Enabled()
}
