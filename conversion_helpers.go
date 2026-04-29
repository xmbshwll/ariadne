package ariadne

import (
	"github.com/xmbshwll/ariadne/internal/model"
)

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func translateSlice[From any, To any](values []From, translate func(From) To) []To {
	if values == nil {
		return nil
	}
	return translateSliceToEmpty(values, translate)
}

func translateSliceToEmpty[From any, To any](values []From, translate func(From) To) []To {
	translated := make([]To, 0, len(values))
	for _, value := range values {
		translated = append(translated, translate(value))
	}
	return translated
}

func translateNonEmptySlice[From any, To any](values []From, translate func(From) To) []To {
	if len(values) == 0 {
		return nil
	}
	return translateSliceToEmpty(values, translate)
}

func translateServiceMap[From any, To any](values map[model.ServiceName]From, translate func(From) To) map[ServiceName]To {
	translated := make(map[ServiceName]To, len(values))
	for service, value := range values {
		translated[fromInternalServiceName(service)] = translate(value)
	}
	return translated
}
