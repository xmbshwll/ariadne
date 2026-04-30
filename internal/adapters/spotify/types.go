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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type apiAlbumResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	ReleaseDate string         `json:"release_date"`
	Label       string         `json:"label"`
	AlbumType   string         `json:"album_type"`
	TotalTracks int            `json:"total_tracks"`
	Images      []apiImage     `json:"images"`
	Artists     []apiArtist    `json:"artists"`
	ExternalIDs apiExternalIDs `json:"external_ids"`
	Copyrights  []apiCopyright `json:"copyrights"`
	Tracks      apiTrackPage   `json:"tracks"`
}

type apiImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type apiArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiExternalIDs struct {
	UPC  string `json:"upc"`
	ISRC string `json:"isrc"`
}

type apiCopyright struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type apiTrackPage struct {
	Items []apiTrack `json:"items"`
}

type apiTrack struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DiscNumber  int            `json:"disc_number"`
	TrackNumber int            `json:"track_number"`
	DurationMS  int            `json:"duration_ms"`
	Explicit    bool           `json:"explicit"`
	Artists     []apiArtist    `json:"artists"`
	ExternalIDs apiExternalIDs `json:"external_ids"`
	Album       apiTrackAlbum  `json:"album"`
}

type apiAlbumSearchResponse struct {
	Albums apiAlbumSearchPage `json:"albums"`
}

type apiAlbumSearchPage struct {
	Items []apiAlbumSummary `json:"items"`
}

type apiAlbumSummary struct {
	ID string `json:"id"`
}

type apiTrackSearchResponse struct {
	Tracks apiTrackSearchPage `json:"tracks"`
}

type apiTrackSearchPage struct {
	Items []apiTrackSearchItem `json:"items"`
}

type apiTrackSearchItem struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DurationMS  int            `json:"duration_ms"`
	Explicit    bool           `json:"explicit"`
	Artists     []apiArtist    `json:"artists"`
	ExternalIDs apiExternalIDs `json:"external_ids"`
	Album       apiTrackAlbum  `json:"album"`
}

type apiTrackAlbum struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ReleaseDate string      `json:"release_date"`
	Images      []apiImage  `json:"images"`
	Artists     []apiArtist `json:"artists"`
}
