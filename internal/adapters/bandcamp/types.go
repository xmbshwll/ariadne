package bandcamp

type schemaAlbum struct {
	ID            string              `json:"@id"`
	Type          string              `json:"@type"`
	Name          string              `json:"name"`
	DatePublished string              `json:"datePublished"`
	Image         any                 `json:"image"`
	ByArtist      schemaMusicGroup    `json:"byArtist"`
	Publisher     schemaMusicGroup    `json:"publisher"`
	Track         schemaTrackList     `json:"track"`
	InAlbum       schemaAlbumRelation `json:"inAlbum"`
	Duration      string              `json:"duration"`
}

type schemaAlbumRelation struct {
	ID   string `json:"@id"`
	Name string `json:"name"`
}

type schemaMusicGroup struct {
	ID   string `json:"@id"`
	Name string `json:"name"`
}

type schemaTrackList struct {
	NumberOfItems   int               `json:"numberOfItems"`
	ItemListElement []schemaTrackItem `json:"itemListElement"`
}

type schemaTrackItem struct {
	Position int                  `json:"position"`
	Item     schemaMusicRecording `json:"item"`
}

type schemaMusicRecording struct {
	ID       string `json:"@id"`
	Name     string `json:"name"`
	Duration string `json:"duration"`
}
