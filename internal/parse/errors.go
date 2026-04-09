package parse

import "errors"

var (
	errUnsupportedAmazonMusicHost = errors.New("unsupported amazon music host")
	errAmazonMusicNotAlbumURL     = errors.New("amazon music url is not an album url")
	errMissingAmazonMusicAlbumID  = errors.New("missing amazon music album id")

	errUnsupportedAppleMusicHost            = errors.New("unsupported apple music host")
	errInvalidAppleMusicAlbumPath           = errors.New("invalid apple music album path")
	errAppleMusicNotAlbumURL                = errors.New("apple music url is not an album url")
	errMissingAppleMusicStorefrontOrAlbumID = errors.New("missing storefront or album id")
	errMissingAppleMusicTrackID             = errors.New("missing apple music track id")

	errMissingBandcampHost = errors.New("missing bandcamp host")
	errBandcampNotAlbumURL = errors.New("bandcamp url is not an album url")
	errBandcampNotSongURL  = errors.New("bandcamp url is not a song url")

	errUnsupportedDeezerHost  = errors.New("unsupported deezer host")
	errInvalidDeezerPath      = errors.New("invalid deezer path")
	errInvalidDeezerAlbumPath = errors.New("invalid deezer album path")
	errDeezerNotAlbumURL      = errors.New("deezer url is not an album url")
	errDeezerNotSongURL       = errors.New("deezer url is not a song url")
	errMissingDeezerAlbumID   = errors.New("missing deezer album id")
	errMissingDeezerTrackID   = errors.New("missing deezer track id")

	errUnsupportedSoundCloudHost        = errors.New("unsupported soundcloud host")
	errSoundCloudNotAlbumURL            = errors.New("soundcloud url is not an album-like set url")
	errSoundCloudNotSongURL             = errors.New("soundcloud url is not a song url")
	errMissingSoundCloudUserOrSetSlug   = errors.New("missing soundcloud user or set slug")
	errMissingSoundCloudUserOrTrackSlug = errors.New("missing soundcloud user or track slug")

	errUnsupportedSpotifyHost = errors.New("unsupported spotify host")
	errSpotifyNotAlbumURL     = errors.New("spotify url is not an album url")
	errSpotifyNotSongURL      = errors.New("spotify url is not a song url")
	errMissingSpotifyAlbumID  = errors.New("missing spotify album id")
	errMissingSpotifyTrackID  = errors.New("missing spotify track id")

	errUnsupportedTIDALHost  = errors.New("unsupported tidal host")
	errInvalidTIDALPath      = errors.New("invalid tidal path")
	errInvalidTIDALAlbumPath = errors.New("invalid tidal album path")
	errTIDALNotAlbumURL      = errors.New("tidal url is not an album url")
	errTIDALNotSongURL       = errors.New("tidal url is not a song url")
	errMissingTIDALAlbumID   = errors.New("missing tidal album id")
	errMissingTIDALTrackID   = errors.New("missing tidal track id")

	errUnsupportedYouTubeMusicHost   = errors.New("unsupported youtube music host")
	errMissingYouTubeMusicBrowseID   = errors.New("missing youtube music browse id")
	errMissingYouTubeMusicPlaylistID = errors.New("missing youtube music playlist id")
	errMissingYouTubeMusicVideoID    = errors.New("missing youtube music video id")
	errYouTubeMusicNotAlbumURL       = errors.New("youtube music url is not an album url")
	errYouTubeMusicNotSongURL        = errors.New("youtube music url is not a song url")
)
