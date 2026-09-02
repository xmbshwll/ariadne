package bandcamp

type fuzzySearchResponse struct {
	Results []fuzzySearchResult `json:"results"`
}

type fuzzySearchResult struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	BandName string `json:"band_name"`
	URL      string `json:"url"`
}

type SchemaAlbum struct {
	ID            string              `json:"@id"`
	Type          string              `json:"@type"`
	Name          string              `json:"name"`
	DatePublished string              `json:"datePublished"`
	Image         any                 `json:"image"`
	ByArtist      SchemaMusicGroup    `json:"byArtist"`
	Publisher     SchemaMusicGroup    `json:"publisher"`
	Track         SchemaTrackList     `json:"track"`
	InAlbum       SchemaAlbumRelation `json:"inAlbum"`
	Duration      string              `json:"duration"`
}

type SchemaAlbumRelation struct {
	ID       string           `json:"@id"`
	Name     string           `json:"name"`
	ByArtist SchemaMusicGroup `json:"byArtist"`
}

type SchemaMusicGroup struct {
	ID   string `json:"@id"`
	Name string `json:"name"`
}

type SchemaTrackList struct {
	NumberOfItems   int               `json:"numberOfItems"`
	ItemListElement []SchemaTrackItem `json:"itemListElement"`
}

type SchemaTrackItem struct {
	Position int                  `json:"position"`
	Item     SchemaMusicRecording `json:"item"`
}

type SchemaMusicRecording struct {
	ID       string `json:"@id"`
	Name     string `json:"name"`
	Duration string `json:"duration"`
}
