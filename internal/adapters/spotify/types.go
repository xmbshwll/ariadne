package spotify

type initialState struct {
	Entities spotifyEntities `json:"entities"`
}

type spotifyEntities struct {
	Items map[string]spotifyAlbumEntity `json:"items"`
}

type spotifyAlbumEntity struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Date      spotifyReleaseDate    `json:"date"`
	CoverArt  spotifyCoverArt       `json:"coverArt"`
	Copyright spotifyCopyrightGroup `json:"copyright"`
	Artists   spotifyArtistList     `json:"artists"`
	TracksV2  spotifyTrackList      `json:"tracksV2"`
}

type spotifyReleaseDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type spotifyCoverArt struct {
	Sources []spotifyImage `json:"sources"`
}

type spotifyImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type spotifyCopyrightGroup struct {
	Items []spotifyCopyright `json:"items"`
}

type spotifyCopyright struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type spotifyArtistList struct {
	Items []spotifyArtistItem `json:"items"`
}

type spotifyArtistItem struct {
	Profile spotifyArtistProfile `json:"profile"`
}

type spotifyArtistProfile struct {
	Name string `json:"name"`
}

type spotifyTrackList struct {
	Items []spotifyTrackWrapper `json:"items"`
}

type spotifyTrackWrapper struct {
	Track spotifyTrack `json:"track"`
}

type spotifyTrack struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	DiscNumber  int               `json:"discNumber"`
	TrackNumber int               `json:"trackNumber"`
	Duration    spotifyTrackTime  `json:"duration"`
	Artists     spotifyArtistList `json:"artists"`
}

type spotifyTrackTime struct {
	TotalMilliseconds int `json:"totalMilliseconds"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type APIAlbumResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	ReleaseDate string         `json:"release_date"`
	Label       string         `json:"label"`
	AlbumType   string         `json:"album_type"`
	TotalTracks int            `json:"total_tracks"`
	Images      []APIImage     `json:"images"`
	Artists     []APIArtist    `json:"artists"`
	ExternalIDs APIExternalIDs `json:"external_ids"`
	Copyrights  []apiCopyright `json:"copyrights"`
	Tracks      APITrackPage   `json:"tracks"`
}

type APIImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type APIArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIExternalIDs struct {
	UPC  string `json:"upc"`
	ISRC string `json:"isrc"`
}

type apiCopyright struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type APITrackPage struct {
	Items []APITrack `json:"items"`
}

type APITrack struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DiscNumber  int            `json:"disc_number"`
	TrackNumber int            `json:"track_number"`
	DurationMS  int            `json:"duration_ms"`
	Explicit    bool           `json:"explicit"`
	Artists     []APIArtist    `json:"artists"`
	ExternalIDs APIExternalIDs `json:"external_ids"`
	Album       APITrackAlbum  `json:"album"`
}

type APIAlbumSearchResponse struct {
	Albums APIAlbumSearchPage `json:"albums"`
}

type APIAlbumSearchPage struct {
	Items []APIAlbumSummary `json:"items"`
}

type APIAlbumSummary struct {
	ID string `json:"id"`
}

type APITrackSearchResponse struct {
	Tracks APITrackSearchPage `json:"tracks"`
}

type APITrackSearchPage struct {
	Items []APITrackSearchItem `json:"items"`
}

type APITrackSearchItem struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DurationMS  int            `json:"duration_ms"`
	Explicit    bool           `json:"explicit"`
	Artists     []APIArtist    `json:"artists"`
	ExternalIDs APIExternalIDs `json:"external_ids"`
	Album       APITrackAlbum  `json:"album"`
}

type APITrackAlbum struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ReleaseDate string      `json:"release_date"`
	Images      []APIImage  `json:"images"`
	Artists     []APIArtist `json:"artists"`
}
