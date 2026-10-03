package har

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorCode identifies an error type.
type ErrorCode int

const (
	// ErrCodeUnknown is an unknown error.
	ErrCodeUnknown ErrorCode = iota
	// ErrCodeFileSystem is a file system error.
	ErrCodeFileSystem
	// ErrCodeJSONParse is a JSON parsing error.
	ErrCodeJSONParse
	// ErrCodeInvalidFormat indicates an invalid format.
	ErrCodeInvalidFormat
	// ErrCodeValidation is a validation error.
	ErrCodeValidation
	// ErrCodeMissingField indicates a required field is missing.
	ErrCodeMissingField
	// ErrCodeInvalidValue indicates an invalid field value.
	ErrCodeInvalidValue
	// ErrCodeUnsupported indicates an unsupported operation.
	ErrCodeUnsupported
)

// HarError is a custom HAR error type.
type HarError struct {
	// Error code.
	Code ErrorCode
	// Error message.
	Message string
	// Original error, if any.
	Err error
	// Dot-separated field path, such as "log.entries[0].request.url".
	Field string
	// Metadata containing additional context.
	Metadata map[string]interface{}
	// Other errors encountered during partial parsing.
	PartialErrors []*HarError
}

// Error implements the error interface.
func (e *HarError) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := e.Message
	if e.Field != "" {
		msg = fmt.Sprintf("field '%s': %s", e.Field, msg)
	}

	if e.Err != nil {
		msg = fmt.Sprintf("%s - %v", msg, e.Err)
	}

	if len(e.PartialErrors) > 0 {
		partialMsgs := make([]string, 0, len(e.PartialErrors))
		for _, pe := range e.PartialErrors {
			if pe == nil {
				continue
			}
			partialMsgs = append(partialMsgs, pe.Error())
		}
		if len(partialMsgs) > 0 {
			msg = fmt.Sprintf("%s (partial errors: %s)", msg, strings.Join(partialMsgs, "; "))
		}
	}

	return msg
}

// Unwrap returns the underlying error for errors.Is/errors.As support.
func (e *HarError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// WithField adds a field path to the error.
func (e *HarError) WithField(field string) *HarError {
	if e == nil {
		return nil
	}
	if e.Field == "" {
		e.Field = field
	} else {
		e.Field = field + "." + e.Field
	}
	return e
}

// WithMetadata adds metadata to the error.
func (e *HarError) WithMetadata(key string, value interface{}) *HarError {
	if e == nil {
		return nil
	}
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// AddPartialError adds a partial parsing error.
func (e *HarError) AddPartialError(err *HarError) *HarError {
	if e == nil {
		return nil
	}
	if err == nil {
		return e
	}
	e.PartialErrors = append(e.PartialErrors, err)
	return e
}

// HasPartialErrors reports whether the error contains partial errors.
func (e *HarError) HasPartialErrors() bool {
	if e == nil {
		return false
	}
	for _, pe := range e.PartialErrors {
		if pe != nil {
			return true
		}
	}
	return false
}

// GetPartialErrors returns all partial errors.
func (e *HarError) GetPartialErrors() []*HarError {
	if e == nil {
		return nil
	}
	return e.PartialErrors
}

// GetCode returns the error code.
func (e *HarError) GetCode() ErrorCode {
	if e == nil {
		return ErrCodeUnknown
	}
	return e.Code
}

// IsFileSystemError reports whether this is a file system error.
func (e *HarError) IsFileSystemError() bool {
	if e == nil {
		return false
	}
	return e.Code == ErrCodeFileSystem
}

// IsJSONParseError reports whether this is a JSON parsing error.
func (e *HarError) IsJSONParseError() bool {
	if e == nil {
		return false
	}
	return e.Code == ErrCodeJSONParse
}

// IsFormatError reports whether this is a format error.
func (e *HarError) IsFormatError() bool {
	if e == nil {
		return false
	}
	return e.Code == ErrCodeInvalidFormat
}

// IsValidationError reports whether this is a validation error.
func (e *HarError) IsValidationError() bool {
	if e == nil {
		return false
	}
	return e.Code == ErrCodeValidation
}

// NewHarError creates a HAR error.
func NewHarError(code ErrorCode, message string, err error) *HarError {
	return &HarError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewFileSystemError creates a file system error.
func NewFileSystemError(message string, err error) *HarError {
	return NewHarError(ErrCodeFileSystem, message, err)
}

// NewJSONParseError creates a JSON parsing error.
func NewJSONParseError(message string, err error) *HarError {
	return NewHarError(ErrCodeJSONParse, message, err)
}

// WrapJSONUnmarshalError wraps a JSON parsing error with additional details.
func WrapJSONUnmarshalError(err error) *HarError {
	if err == nil {
		return nil
	}

	// Try to extract details from the JSON error.
	var jsonErr *json.UnmarshalTypeError
	var syntaxErr *json.SyntaxError

	if e, ok := err.(*json.UnmarshalTypeError); ok {
		jsonErr = e
		return NewJSONParseError(
			fmt.Sprintf("type mismatch: expected %s, got %s",
				jsonErr.Type.String(), jsonErr.Value),
			err,
		).WithField(jsonErr.Field).WithMetadata("offset", jsonErr.Offset)
	} else if e, ok := err.(*json.SyntaxError); ok {
		syntaxErr = e
		return NewJSONParseError(
			fmt.Sprintf("JSON syntax error: %s", syntaxErr.Error()),
			err,
		).WithMetadata("offset", syntaxErr.Offset)
	} else if strings.Contains(err.Error(), "cannot unmarshal") {
		// Handle other JSON parsing errors whose type cannot be identified precisely.
		parts := strings.Split(err.Error(), ":")
		if len(parts) >= 2 {
			return NewJSONParseError(
				fmt.Sprintf("JSON parsing error: %s", strings.TrimSpace(parts[1])),
				err,
			)
		}
	}

	// Handle other JSON errors.
	return NewJSONParseError("JSON parsing error", err)
}

// NewValidationError creates a validation error.
func NewValidationError(message string, field string) *HarError {
	return NewHarError(ErrCodeValidation, message, nil).WithField(field)
}

// NewInvalidFormatError creates an invalid format error.
func NewInvalidFormatError(message string) *HarError {
	return NewHarError(ErrCodeInvalidFormat, message, nil)
}

// NewMissingFieldError creates an error for a missing field.
func NewMissingFieldError(field string) *HarError {
	return NewHarError(ErrCodeMissingField, "required field is missing", nil).WithField(field)
}

// NewInvalidValueError creates an error for an invalid field value.
func NewInvalidValueError(field string, value interface{}, reason string) *HarError {
	msg := "invalid field value"
	if reason != "" {
		msg = msg + ": " + reason
	}
	return NewHarError(ErrCodeInvalidValue, msg, nil).
		WithField(field).
		WithMetadata("value", value)
}

// NewUnsupportedError creates an unsupported operation error.
func NewUnsupportedError(message string) *HarError {
	return NewHarError(ErrCodeUnsupported, message, nil)
}

// ParseOptions configures parsing.
type ParseOptions struct {
	// Enable lenient parsing, which attempts to parse valid portions of the input.
	Lenient bool
	// Skip validation.
	SkipValidation bool
	// Collect all parsing warnings without failing the parse.
	CollectWarnings bool
	// Maximum number of warnings before parsing stops.
	MaxWarnings int
}

// DefaultParseOptions returns the default parsing options.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		Lenient:         false,
		SkipValidation:  false,
		CollectWarnings: false,
		MaxWarnings:     100,
	}
}
