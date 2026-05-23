package providers

import (
	"fmt"
	"strings"
)

type ErrorCode string

const (
	ErrorCodeMissingConfig   ErrorCode = "missing_config"
	ErrorCodeMissingAPIKey   ErrorCode = "missing_api_key"
	ErrorCodeProviderFailure ErrorCode = "provider_failure"
)

type Error struct {
	Code     ErrorCode
	Provider string
	Message  string
	Cause    error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func MissingAPIKey(provider string) error {
	return &Error{Code: ErrorCodeMissingAPIKey, Provider: provider, Message: fmt.Sprintf("Missing API key for %s.", provider)}
}

func MissingConfig(provider, field string) error {
	return &Error{Code: ErrorCodeMissingConfig, Provider: provider, Message: fmt.Sprintf("Missing %s for %s.", field, provider)}
}

func ProviderFailure(provider string, cause error) error {
	details := summarizeErrorCause(cause)
	message := fmt.Sprintf("%s provider request failed.", provider)
	if details != "" {
		message = fmt.Sprintf("%s provider request failed: %s.", provider, details)
	}
	return &Error{Code: ErrorCodeProviderFailure, Provider: provider, Message: message, Cause: cause}
}

func summarizeErrorCause(cause error) string {
	if cause == nil {
		return ""
	}
	details := []string{strings.TrimSpace(cause.Error())}
	unique := make([]string, 0, len(details))
	for _, detail := range details {
		if detail == "" || containsString(unique, detail) {
			continue
		}
		unique = append(unique, detail)
	}
	return truncateDetail(strings.Join(unique, " "))
}

func truncateDetail(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 300 {
		return string(runes[:300])
	}
	return value
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
