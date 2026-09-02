package tidal

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type APIDocument struct {
	Data     any            `json:"data"`
	Included []APIResource  `json:"included"`
	Links    map[string]any `json:"links"`
}

type APIResource struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    ResourceAttributes    `json:"attributes"`
	Relationships ResourceRelationships `json:"relationships"`
}

type ResourceAttributes struct {
	Title         string            `json:"title"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	BarcodeID     string            `json:"barcodeId"`
	UPC           string            `json:"upc"`
	ReleaseDate   string            `json:"releaseDate"`
	Duration      string            `json:"duration"`
	Explicit      bool              `json:"explicit"`
	NumberOfItems int               `json:"numberOfItems"`
	Copyright     ResourceCopyright `json:"copyright"`
	Files         []ResourceFile    `json:"files"`
	ISRC          string            `json:"isrc"`
}

type ResourceCopyright struct {
	Text string `json:"text"`
}

type ResourceFile struct {
	Href string   `json:"href"`
	Meta FileMeta `json:"meta"`
}

type FileMeta struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ResourceRelationships struct {
	Artists  Relationship `json:"artists"`
	Items    Relationship `json:"items"`
	CoverArt Relationship `json:"coverArt"`
	Albums   Relationship `json:"albums"`
	Tracks   Relationship `json:"tracks"`
}

type Relationship struct {
	Data []RelationshipData `json:"data"`
}

type RelationshipData struct {
	ID   string           `json:"id"`
	Type string           `json:"type"`
	Meta RelationshipMeta `json:"meta"`
}

type RelationshipMeta struct {
	TrackNumber  int `json:"trackNumber"`
	VolumeNumber int `json:"volumeNumber"`
}
