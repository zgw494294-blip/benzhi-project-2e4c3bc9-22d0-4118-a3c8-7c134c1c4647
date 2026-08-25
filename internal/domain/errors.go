package domain

import "fmt"

type ErrorCode string

const (
	ErrInvalid      ErrorCode = "invalid"
	ErrNotFound     ErrorCode = "not_found"
	ErrConflict     ErrorCode = "conflict"
	ErrVersion      ErrorCode = "version_conflict"
	ErrState        ErrorCode = "invalid_state"
	ErrQuality      ErrorCode = "quality_gate_failed"
	ErrUnauthorized ErrorCode = "unauthorized"
)

type DomainError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Field   string    `json:"field,omitempty"`
}

func (e *DomainError) Error() string { return string(e.Code) + ": " + e.Message }
func Invalid(message, field string) error {
	return &DomainError{Code: ErrInvalid, Message: message, Field: field}
}
func NotFound(message string) error        { return &DomainError{Code: ErrNotFound, Message: message} }
func Conflict(message string) error        { return &DomainError{Code: ErrConflict, Message: message} }
func VersionConflict(message string) error { return &DomainError{Code: ErrVersion, Message: message} }
func StateError(message string) error      { return &DomainError{Code: ErrState, Message: message} }
func QualityError(message string) error    { return &DomainError{Code: ErrQuality, Message: message} }
func Wrap(code ErrorCode, format string, args ...any) error {
	return &DomainError{Code: code, Message: fmt.Sprintf(format, args...)}
}
