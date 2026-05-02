package adapterutil

import (
	"errors"
	"fmt"

	"github.com/xmbshwll/ariadne/internal/model"
)

// ErrRuntimeDeferred identifies service URLs Ariadne can parse but cannot hydrate at runtime yet.
var ErrRuntimeDeferred = errors.New("runtime adapter is deferred")

// RuntimeDeferredError carries the service-specific deferred runtime failure.
type RuntimeDeferredError struct {
	Service model.ServiceName
	Reason  string
}

func (e RuntimeDeferredError) Error() string {
	if e.Service == "" {
		return ErrRuntimeDeferred.Error()
	}
	if e.Reason == "" {
		return deferredServiceLabel(e.Service) + " runtime adapter is deferred"
	}
	return fmt.Sprintf("%s runtime adapter is deferred: %s", deferredServiceLabel(e.Service), e.Reason)
}

func (e RuntimeDeferredError) Is(target error) bool {
	if target == ErrRuntimeDeferred {
		return true
	}

	targetErr, ok := target.(RuntimeDeferredError)
	if !ok {
		targetPtr, ok := target.(*RuntimeDeferredError)
		if !ok {
			return false
		}
		targetErr = *targetPtr
	}
	return targetErr.Service != "" && e.Service == targetErr.Service
}

func RuntimeDeferredService(service model.ServiceName) error {
	return RuntimeDeferredError{Service: service}
}

func NewRuntimeDeferredError(service model.ServiceName, reason string) error {
	return RuntimeDeferredError{Service: service, Reason: reason}
}

func deferredServiceLabel(service model.ServiceName) string {
	switch service {
	case model.ServiceAmazonMusic:
		return "amazon music"
	case model.ServiceYouTubeMusic:
		return "youtube music"
	default:
		return string(service)
	}
}
