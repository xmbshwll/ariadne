package wiring

import "github.com/xmbshwll/ariadne/internal/model"

// credentialHints maps a credential-gated Music Service to the environment
// variables that supply its Credential Token. The Provider Catalog owns this
// explanation so callers report a missing token from the decision instead of
// switching on service names.
var credentialHints = map[model.ServiceName]string{
	model.ServiceSpotify: "SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set",
	model.ServiceTIDAL:   "TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set",
}
