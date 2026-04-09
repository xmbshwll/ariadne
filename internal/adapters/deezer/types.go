package deezer

type albumResponse struct {
	ID             int           `json:"id"`
	Title          string        `json:"title"`
	UPC            string        `json:"upc"`
	Link           string        `json:"link"`
	Cover          string        `json:"cover"`
	CoverMedium    string        `json:"cover_medium"`
	CoverBig       string        `json:"cover_big"`
	CoverXL        string        `json:"cover_xl"`
	Label          string        `json:"label"`
	NBTracks       int           `json:"nb_tracks"`
	Duration       int           `json:"duration"`
	ReleaseDate    string        `json:"release_date"`
	TracklistURL   string        `json:"tracklist"`
	ExplicitLyrics bool          `json:"explicit_lyrics"`
	Artist         artistRef     `json:"artist"`
	Contributors   []contributor `json:"contributors"`
}

type tracksResponse struct {
	Data []trackResponse `json:"data"`
}

type albumSearchResponse struct {
	Data []albumResponse `json:"data"`
}

type trackLookupResponse struct {
	ID             int            `json:"id"`
	Title          string         `json:"title"`
	Link           string         `json:"link"`
	ISRC           string         `json:"isrc"`
	Album          albumLookupRef `json:"album"`
	Artist         artistRef      `json:"artist"`
	Duration       int            `json:"duration"`
	TrackPosition  int            `json:"track_position"`
	DiskNumber     int            `json:"disk_number"`
	ExplicitLyrics bool           `json:"explicit_lyrics"`
}

type trackResponse struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Duration       int       `json:"duration"`
	TrackPosition  int       `json:"track_position"`
	DiskNumber     int       `json:"disk_number"`
	ISRC           string    `json:"isrc"`
	Artist         artistRef `json:"artist"`
	ExplicitLyrics bool      `json:"explicit_lyrics"`
}

type albumLookupRef struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Link         string `json:"link"`
	TracklistURL string `json:"tracklist"`
	Cover        string `json:"cover"`
	CoverMedium  string `json:"cover_medium"`
	CoverBig     string `json:"cover_big"`
	CoverXL      string `json:"cover_xl"`
	ReleaseDate  string `json:"release_date"`
}

type artistRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type contributor struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
