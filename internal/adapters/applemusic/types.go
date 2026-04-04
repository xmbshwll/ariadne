package applemusic

type lookupResponse struct {
	ResultCount int          `json:"resultCount"`
	Results     []lookupItem `json:"results"`
}

type lookupItem struct {
	WrapperType            string `json:"wrapperType"`
	CollectionType         string `json:"collectionType"`
	Kind                   string `json:"kind"`
	ArtistID               int64  `json:"artistId"`
	CollectionID           int64  `json:"collectionId"`
	TrackID                int64  `json:"trackId"`
	ArtistName             string `json:"artistName"`
	CollectionName         string `json:"collectionName"`
	CollectionViewURL      string `json:"collectionViewUrl"`
	TrackName              string `json:"trackName"`
	TrackCount             int    `json:"trackCount"`
	DiscNumber             int    `json:"discNumber"`
	TrackNumber            int    `json:"trackNumber"`
	TrackTimeMillis        int    `json:"trackTimeMillis"`
	ReleaseDate            string `json:"releaseDate"`
	ArtworkURL60           string `json:"artworkUrl60"`
	ArtworkURL100          string `json:"artworkUrl100"`
	CollectionExplicitness string `json:"collectionExplicitness"`
	TrackExplicitness      string `json:"trackExplicitness"`
	Copyright              string `json:"copyright"`
	PrimaryGenreName       string `json:"primaryGenreName"`
}
