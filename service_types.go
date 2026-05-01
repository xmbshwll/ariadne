package ariadne

// ServiceName identifies a music service known to the library.
type ServiceName string

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
type ServiceCapabilities struct {
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

// TargetServiceRequestStatus explains whether a requested target service can be used under a Config.
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
	Service ServiceName
	Status  TargetServiceRequestStatus
}
