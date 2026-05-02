// Package convert defines the container and ownership rules at the public/internal type seam.
// Field mappings stay in the conversion files of the ariadne package; nil, empty, and
// deep-copy semantics live here.
package convert

// CloneSlice returns an independent copy of values.
// nil input produces nil output.
func CloneSlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	cloned := make([]T, len(values))
	copy(cloned, values)
	return cloned
}

// CloneStrings is CloneSlice specialised for []string.
func CloneStrings(values []string) []string {
	return CloneSlice(values)
}

// TranslateSlice converts each element through translate.
// nil input produces nil output; empty input produces empty output.
func TranslateSlice[From any, To any](values []From, translate func(From) To) []To {
	if values == nil {
		return nil
	}
	return TranslateSliceToEmpty(values, translate)
}

// TranslateSliceToEmpty converts each element through translate.
// It always returns a non-nil slice; nil input produces an empty slice.
func TranslateSliceToEmpty[From any, To any](values []From, translate func(From) To) []To {
	translated := make([]To, len(values))
	for i, value := range values {
		translated[i] = translate(value)
	}
	return translated
}

// TranslateNonEmptySlice converts each element through translate.
// nil or empty input produces nil output; non-empty input produces translated output.
func TranslateNonEmptySlice[From any, To any](values []From, translate func(From) To) []To {
	if len(values) == 0 {
		return nil
	}
	return TranslateSliceToEmpty(values, translate)
}
