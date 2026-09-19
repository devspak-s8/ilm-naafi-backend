package provider

import (
	"fmt"
	"net/http"
)

type ProviderError struct {
	Status int
	Code   string
	Err    error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *ProviderError) Unwrap() error { return e.Err }

func unavailable(err error) *ProviderError {
	return &ProviderError{Status: http.StatusServiceUnavailable, Code: "QURAN_PROVIDER_UNAVAILABLE", Err: err}
}

func invalidResponse(err error) *ProviderError {
	return &ProviderError{Status: http.StatusBadGateway, Code: "QURAN_PROVIDER_INVALID_RESPONSE", Err: err}
}

func statusError(status int) *ProviderError {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return &ProviderError{Status: http.StatusBadRequest, Code: "QURAN_INVALID_REQUEST"}
	case http.StatusUnauthorized:
		return &ProviderError{Status: http.StatusBadGateway, Code: "QURAN_PROVIDER_UNAUTHORIZED"}
	case http.StatusForbidden:
		return &ProviderError{Status: http.StatusBadGateway, Code: "QURAN_PROVIDER_FORBIDDEN"}
	case http.StatusNotFound:
		return &ProviderError{Status: http.StatusNotFound, Code: "QURAN_RESOURCE_NOT_FOUND"}
	case http.StatusTooManyRequests:
		return &ProviderError{Status: http.StatusTooManyRequests, Code: "QURAN_PROVIDER_RATE_LIMITED"}
	default:
		return &ProviderError{Status: http.StatusServiceUnavailable, Code: "QURAN_PROVIDER_UNAVAILABLE"}
	}
}

func unsupportedProvider(name string) *ProviderError {
	return unavailable(fmt.Errorf("unsupported Quran provider %q", name))
}
