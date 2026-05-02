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

type sourceInput[P any, Entity any] struct {
	Parsed P
	Entity Entity
}

func resolveSourceInput[S interface{ Service() model.ServiceName }, P any, Entity any](
	ctx context.Context,
	sources []S,
	inputURL string,
	parse func(S, string) (*P, error),
	hydrate func(context.Context, S, P) (*Entity, error),
	entityLabel string,
	nilEntityErr error,
) (sourceInput[P, Entity], error) {
	var zero sourceInput[P, Entity]
	if len(sources) == 0 {
		return zero, ErrNoSourceAdapters
	}

	adapter, parsed, err := recognizeSourceInput(sources, inputURL, parse)
	if err != nil {
		return zero, err
	}

	entity, err := hydrateSourceInput(ctx, adapter, *parsed, hydrate, entityLabel, nilEntityErr)
	if err != nil {
		return zero, err
	}

	return sourceInput[P, Entity]{
		Parsed: *parsed,
		Entity: *entity,
	}, nil
}

type fatalParseFailure interface {
	FatalParseFailure() bool
}

func recognizeSourceInput[S any, P any](sources []S, inputURL string, parse func(S, string) (*P, error)) (S, *P, error) {
	var zero S
	for _, source := range sources {
		parsed, err := parse(source, inputURL)
		if err != nil {
			var fatal fatalParseFailure
			if errors.As(err, &fatal) && fatal.FatalParseFailure() {
				return zero, nil, err
			}
			continue
		}
		if parsed == nil {
			return zero, nil, ErrSourceAdapterReturnedNilParsedURL
		}
		return source, parsed, nil
	}
	return zero, nil, fmt.Errorf("%w: %s", ErrUnsupportedURL, inputURL)
}

func hydrateSourceInput[S interface{ Service() model.ServiceName }, P any, Entity any](
	ctx context.Context,
	adapter S,
	parsed P,
	hydrate func(context.Context, S, P) (*Entity, error),
	entityLabel string,
	nilEntityErr error,
) (*Entity, error) {
	entity, err := hydrate(ctx, adapter, parsed)
	if err != nil {
		return nil, fmt.Errorf("fetch source %s with %s: %w", entityLabel, adapter.Service(), err)
	}
	if entity == nil {
		return nil, fmt.Errorf("%w from %s", nilEntityErr, adapter.Service())
	}
	return entity, nil
}
