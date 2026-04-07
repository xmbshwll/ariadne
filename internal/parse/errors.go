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

	errMissingBandcampHost = errors.New("missing bandcamp host")
	errBandcampNotAlbumURL = errors.New("bandcamp url is not an album url")

	errUnsupportedDeezerHost  = errors.New("unsupported deezer host")
	errInvalidDeezerAlbumPath = errors.New("invalid deezer album path")
	errDeezerNotAlbumURL      = errors.New("deezer url is not an album url")
	errMissingDeezerAlbumID   = errors.New("missing deezer album id")

	errUnsupportedSoundCloudHost      = errors.New("unsupported soundcloud host")
	errSoundCloudNotAlbumURL          = errors.New("soundcloud url is not an album-like set url")
	errMissingSoundCloudUserOrSetSlug = errors.New("missing soundcloud user or set slug")

	errUnsupportedSpotifyHost = errors.New("unsupported spotify host")
	errSpotifyNotAlbumURL     = errors.New("spotify url is not an album url")
	errMissingSpotifyAlbumID  = errors.New("missing spotify album id")

	errUnsupportedTIDALHost  = errors.New("unsupported tidal host")
	errInvalidTIDALAlbumPath = errors.New("invalid tidal album path")
	errTIDALNotAlbumURL      = errors.New("tidal url is not an album url")
	errMissingTIDALAlbumID   = errors.New("missing tidal album id")

	errUnsupportedYouTubeMusicHost   = errors.New("unsupported youtube music host")
	errMissingYouTubeMusicBrowseID   = errors.New("missing youtube music browse id")
	errMissingYouTubeMusicPlaylistID = errors.New("missing youtube music playlist id")
	errYouTubeMusicNotAlbumURL       = errors.New("youtube music url is not an album url")
)
