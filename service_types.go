package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

// ServiceName identifies a music service known to the library.
type ServiceName = model.ServiceName

// Re-export service name constants so callers don't need to import internal/model.
const (
	// ServiceSpotify identifies Spotify.
	ServiceSpotify ServiceName = "spotify"
	// ServiceAppleMusic identifies Apple Music.
	ServiceAppleMusic ServiceName = "appleMusic"
	// ServiceDeezer identifies Deezer.
	ServiceDeezer ServiceName = "deezer"
	// ServiceSoundCloud identifies SoundCloud.
	ServiceSoundCloud ServiceName = "soundcloud"
	// ServiceBandcamp identifies Bandcamp.
	ServiceBandcamp ServiceName = "bandcamp"
	// ServiceYouTubeMusic identifies YouTube Music.
	ServiceYouTubeMusic ServiceName = "youtubeMusic"
	// ServiceTIDAL identifies TIDAL.
	ServiceTIDAL ServiceName = "tidal"
	// ServiceAmazonMusic identifies Amazon Music.
	ServiceAmazonMusic ServiceName = "amazonMusic"
)

// MatchStrength buckets raw scores into user-facing confidence bands.
type MatchStrength string

// Re-export match strength constants.
const (
	// MatchStrengthVeryWeak indicates a low-confidence match.
	MatchStrengthVeryWeak MatchStrength = "very_weak"
	// MatchStrengthWeak indicates a weak match.
	MatchStrengthWeak MatchStrength = "weak"
	// MatchStrengthProbable indicates a probable match.
	MatchStrengthProbable MatchStrength = "probable"
	// MatchStrengthStrong indicates a strong match.
	MatchStrengthStrong MatchStrength = "strong"
)

// ServiceCapabilities describes Ariadne's built-in runtime support for one service.
type ServiceCapabilities = wiring.Capabilities

// TargetServiceRequestStatus explains whether a requested target service can be used under a Config.
type TargetServiceRequestStatus = wiring.TargetServiceRequestStatus

// Re-export target service request status constants so callers don't need to import the wiring module.
const (
	// TargetServiceRequestAvailable means the service can be used as requested.
	TargetServiceRequestAvailable = wiring.TargetServiceRequestAvailable
	// TargetServiceRequestUnknown means the requested service name or alias is not known.
	TargetServiceRequestUnknown = wiring.TargetServiceRequestUnknown
	// TargetServiceRequestUnsupported means the service is known but does not support the requested target role.
	TargetServiceRequestUnsupported = wiring.TargetServiceRequestUnsupported
	// TargetServiceRequestParseOnly means the service can parse URLs but has no runtime target search capability.
	TargetServiceRequestParseOnly = wiring.TargetServiceRequestParseOnly
	// TargetServiceRequestCredentialsRequired means the target role needs missing credentials.
	TargetServiceRequestCredentialsRequired = wiring.TargetServiceRequestCredentialsRequired
)

// TargetServiceRequestDecision reports Provider Catalog validation for one requested target service.
type TargetServiceRequestDecision = wiring.TargetServiceRequestDecision
