package resolve

import (
	"context"
	"errors"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
)

var (
	// ErrSourceAdapterReturnedNilParsedURL indicates that a source adapter returned nil parsed URL instead of a value or error.
	ErrSourceAdapterReturnedNilParsedURL = errors.New("source adapter returned nil parsed url")
	// ErrSourceAdapterReturnedNilAlbum indicates that an album source adapter returned nil album without an error.
	ErrSourceAdapterReturnedNilAlbum = errors.New("source adapter returned nil album")
	// ErrSourceAdapterReturnedNilSong indicates that a song source adapter returned nil song without an error.
	ErrSourceAdapterReturnedNilSong = errors.New("source adapter returned nil song")

	errNilSourceAlbum = sourceAdapterContractError{
		message: "fetch source album returned nil",
		target:  ErrSourceAdapterReturnedNilAlbum,
	}
	errNilSourceSong = sourceAdapterContractError{
		message: "fetch source song returned nil",
		target:  ErrSourceAdapterReturnedNilSong,
	}
)

type sourceAdapterContractError struct {
	message string
	target  error
}

func (e sourceAdapterContractError) Error() string {
	return e.message
}

func (e sourceAdapterContractError) Is(target error) bool {
	return target == e.target
}

type fatalParseFailure interface {
	FatalParseFailure() bool
}

// recognizeSourceInput returns the first Source Adapter that parses inputURL.
// Fatal parse failures stop recognition; nil parses violate the adapter contract.
func recognizeSourceInput[S any, P any](sources []S, inputURL string, parse func(S) (*P, error)) (*P, S, error) {
	var zero S
	if len(sources) == 0 {
		return nil, zero, ErrNoSourceAdapters
	}
	for _, source := range sources {
		parsed, err := parse(source)
		if err != nil {
			var fatal fatalParseFailure
			if errors.As(err, &fatal) && fatal.FatalParseFailure() {
				return nil, zero, err
			}
			continue
		}
		if parsed == nil {
			return nil, zero, ErrSourceAdapterReturnedNilParsedURL
		}
		return parsed, source, nil
	}
	return nil, zero, fmt.Errorf("%w: %s", ErrUnsupportedURL, inputURL)
}

// hydrateSourceInput runs Runtime Hydration for a recognized Source Input and
// normalizes the nil-entity adapter contract violation to a contract error.
func hydrateSourceInput[S interface{ Service() model.ServiceName }, Entity any](
	ctx context.Context,
	adapter S,
	entityLabel string,
	nilEntityErr error,
	hydrate func(context.Context) (*Entity, error),
) (*Entity, error) {
	entity, err := hydrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch source %s with %s: %w", entityLabel, adapter.Service(), err)
	}
	if entity == nil {
		return nil, fmt.Errorf("%w from %s", nilEntityErr, adapter.Service())
	}
	return entity, nil
}
