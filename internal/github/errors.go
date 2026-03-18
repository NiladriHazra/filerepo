package github

// ErrorKind identifies a GitHub API failure category.
type ErrorKind string

const (
	ErrorInvalidToken     ErrorKind = "invalid_token"
	ErrorRateLimitReached ErrorKind = "rate_limit"
	ErrorNotFound         ErrorKind = "not_found"
	ErrorAPI              ErrorKind = "api"
)

// APIError captures GitHub API failure categories that the UI can recover from.
type APIError struct {
	Kind    ErrorKind
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}
