package resolve

import (
	"context"
	"errors"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
)

var (
	errNilSourceAlbum = errors.New("fetch source album returned nil")
	errNilSourceSong  = errors.New("fetch source song returned nil")
)

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
			continue
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
