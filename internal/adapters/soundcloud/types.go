package soundcloud

import "encoding/json"

type hydrationEnvelope struct {
	Hydratable string          `json:"hydratable"`
	Data       json.RawMessage `json:"data"`
}

type searchResponse struct {
	Collection []soundPlaylist `json:"collection"`
}

type trackSearchResponse struct {
	Collection []soundTrack `json:"collection"`
}

type soundPlaylist struct {
	ID           int64        `json:"id"`
	Kind         string       `json:"kind"`
	SetType      string       `json:"set_type"`
	Title        string       `json:"title"`
	Permalink    string       `json:"permalink"`
	PermalinkURL string       `json:"permalink_url"`
	ArtworkURL   string       `json:"artwork_url"`
	ReleaseDate  string       `json:"release_date"`
	PublishedAt  string       `json:"published_at"`
	DisplayDate  string       `json:"display_date"`
	Duration     int          `json:"duration"`
	LabelName    string       `json:"label_name"`
	Genre        string       `json:"genre"`
	User         soundUser    `json:"user"`
	Tracks       []soundTrack `json:"tracks"`
}

type soundUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Permalink    string `json:"permalink"`
	PermalinkURL string `json:"permalink_url"`
}

type soundTrack struct {
	ID                int64             `json:"id"`
	Title             string            `json:"title"`
	PermalinkURL      string            `json:"permalink_url"`
	ArtworkURL        string            `json:"artwork_url"`
	Duration          int               `json:"duration"`
	FullDuration      int               `json:"full_duration"`
	ReleaseDate       string            `json:"release_date"`
	DisplayDate       string            `json:"display_date"`
	LabelName         string            `json:"label_name"`
	User              soundUser         `json:"user"`
	PublisherMetadata publisherMetadata `json:"publisher_metadata"`
}

type publisherMetadata struct {
	Artist          string `json:"artist"`
	AlbumTitle      string `json:"album_title"`
	UPCOrEAN        string `json:"upc_or_ean"`
	ISRC            string `json:"isrc"`
	Explicit        bool   `json:"explicit"`
	PLineForDisplay string `json:"p_line_for_display"`
	CLineForDisplay string `json:"c_line_for_display"`
}
