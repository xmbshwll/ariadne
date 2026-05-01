package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
)

// Conversion helpers define the container and ownership rules at the
// public/internal type seam. Field mappings stay in the conversion files; nil,
// empty, and deep-copy semantics live here.
func cloneSlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	cloned := make([]T, len(values))
	copy(cloned, values)
	return cloned
}

func cloneStrings(values []string) []string {
	return cloneSlice(values)
}

// translateSlice preserves nil inputs and returns non-nil empty output for
// non-nil empty input.
func translateSlice[From any, To any](values []From, translate func(From) To) []To {
	if values == nil {
		return nil
	}
	return translateSliceToEmpty(values, translate)
}

// translateSliceToEmpty always returns a non-nil slice, including for nil input.
func translateSliceToEmpty[From any, To any](values []From, translate func(From) To) []To {
	translated := make([]To, len(values))
	for i, value := range values {
		translated[i] = translate(value)
	}
	return translated
}

// translateNonEmptySlice returns nil for nil or empty input.
func translateNonEmptySlice[From any, To any](values []From, translate func(From) To) []To {
	if len(values) == 0 {
		return nil
	}
	return translateSliceToEmpty(values, translate)
}

// translateServiceMap always returns a non-nil map so public result containers
// are stable even when no target services were searched.
func translateServiceMap[From any, To any](values map[model.ServiceName]From, translate func(From) To) map[ServiceName]To {
	translated := make(map[ServiceName]To, len(values))
	for service, value := range values {
		translated[fromInternalServiceName(service)] = translate(value)
	}
	return translated
}
