package tidal

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type apiDocument struct {
	Data     any            `json:"data"`
	Included []apiResource  `json:"included"`
	Links    map[string]any `json:"links"`
}

type apiResource struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    resourceAttributes    `json:"attributes"`
	Relationships resourceRelationships `json:"relationships"`
}

type resourceAttributes struct {
	Title         string            `json:"title"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	BarcodeID     string            `json:"barcodeId"`
	UPC           string            `json:"upc"`
	ReleaseDate   string            `json:"releaseDate"`
	Duration      string            `json:"duration"`
	Explicit      bool              `json:"explicit"`
	NumberOfItems int               `json:"numberOfItems"`
	Copyright     resourceCopyright `json:"copyright"`
	Files         []resourceFile    `json:"files"`
	ISRC          string            `json:"isrc"`
}

type resourceCopyright struct {
	Text string `json:"text"`
}

type resourceFile struct {
	Href string   `json:"href"`
	Meta fileMeta `json:"meta"`
}

type fileMeta struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type resourceRelationships struct {
	Artists  relationship `json:"artists"`
	Items    relationship `json:"items"`
	CoverArt relationship `json:"coverArt"`
	Albums   relationship `json:"albums"`
}

type relationship struct {
	Data []relationshipData `json:"data"`
}

type relationshipData struct {
	ID   string           `json:"id"`
	Type string           `json:"type"`
	Meta relationshipMeta `json:"meta"`
}

type relationshipMeta struct {
	TrackNumber  int `json:"trackNumber"`
	VolumeNumber int `json:"volumeNumber"`
}
