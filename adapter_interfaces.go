// Package ariadne defines adapter interfaces.
package ariadne

import "github.com/xmbshwll/ariadne/internal/resolve"

// SourceAdapter fetches canonical album metadata from a parsed source URL.
//
// Implementations must either return a parsed value or a non-nil error from ParseAlbumURL,
// and either a canonical album or a non-nil error from FetchAlbum. Returning nil with a nil
// error violates the adapter contract and is normalized to an exported ErrSourceAdapterReturnedNil* sentinel.
type SourceAdapter = resolve.SourceAdapter

// SongSourceAdapter fetches canonical song metadata from a parsed source URL.
//
// Implementations must either return a parsed value or a non-nil error from ParseSongURL,
// and either a canonical song or a non-nil error from FetchSong. Returning nil with a nil
// error violates the adapter contract and is normalized to an exported ErrSourceAdapterReturnedNil* sentinel.
type SongSourceAdapter = resolve.SongSourceAdapter

// TargetAdapter identifies one album target Music Service.
//
// Adapters participate in album Target Search by additionally implementing any
// of UPCSearcher, ISRCSearcher, or MetadataSearcher.
type TargetAdapter = resolve.TargetAdapter

// UPCSearcher searches album targets by UPC.
type UPCSearcher = resolve.UPCSearcher

// ISRCSearcher searches album targets by track ISRCs.
type ISRCSearcher = resolve.ISRCSearcher

// MetadataSearcher searches album targets by canonical metadata.
type MetadataSearcher = resolve.MetadataSearcher

// SongTargetAdapter identifies one song target Music Service.
//
// Adapters participate in song Target Search by additionally implementing
// SongISRCSearcher or SongMetadataSearcher.
type SongTargetAdapter = resolve.SongTargetAdapter

// SongISRCSearcher searches song targets by ISRC.
type SongISRCSearcher = resolve.SongISRCSearcher

// SongMetadataSearcher searches song targets by canonical metadata.
type SongMetadataSearcher = resolve.SongMetadataSearcher
