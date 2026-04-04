package amazonmusic

import (
	"context"
	"errors"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAdapter(t *testing.T) {
	adapter := New(nil)

	parsed, err := adapter.ParseAlbumURL("https://music.amazon.com/albums/B0064UPU4G")
	if err != nil {
		t.Fatalf("ParseAlbumURL error: %v", err)
	}
	if parsed.ID != "B0064UPU4G" {
		t.Fatalf("id = %q, want B0064UPU4G", parsed.ID)
	}

	_, err = adapter.FetchAlbum(context.Background(), model.ParsedAlbumURL{
		Service:      model.ServiceAmazonMusic,
		EntityType:   "album",
		ID:           "B0064UPU4G",
		CanonicalURL: "https://music.amazon.com/albums/B0064UPU4G",
	})
	if !errors.Is(err, ErrDeferredRuntimeAdapter) {
		t.Fatalf("error = %v, want deferred adapter error", err)
	}

	upcResults, err := adapter.SearchByUPC(context.Background(), "123")
	if err != nil {
		t.Fatalf("SearchByUPC error: %v", err)
	}
	if len(upcResults) != 0 {
		t.Fatalf("upc results = %d, want 0", len(upcResults))
	}
}
