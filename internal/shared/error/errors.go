package error

import (
	"errors"
	"net/http"
)

type DomainError interface {
	error // Embed standard error interface
	Info() string
}

type domainSentinel struct {
	errInfo string
}

func (e *domainSentinel) Error() string {
	return e.errInfo
}

func (e *domainSentinel) Info() string {
	return e.errInfo
}

// ErrorResponse is the JSON response structure for errors
type ErrorResponse struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"` // client message
}

// Common errors
var (
	domainErrorResponses = map[string]ErrorResponse{}

	// ValidationFailed indicates the request payload failed validation
	ValidationFailed = ErrorResponse{
		Status:  http.StatusBadRequest,
		Code:    "ERROR-001", // METHOD_ARGUMENT_NOT_VALID
		Message: "잘못된 요청입니다.",
	}

	// InvalidRequest indicates the request format is invalid (e.g., JSON parsing error)
	InvalidRequest = ErrorResponse{
		Status:  http.StatusBadRequest,
		Code:    "ERROR-002", // INVALID_REQUEST
		Message: "잘못된 요청 형식입니다.",
	}

	// InternalServerError indicates an unexpected server error
	InternalServerError = ErrorResponse{
		Status:  http.StatusInternalServerError,
		Code:    "ERROR-003", // INTERNAL_SERVER_ERROR
		Message: "서버 내부 오류가 발생했습니다.",
	}
)

// NewDomainError creates a sentinel error that can participate in error chains.
func NewDomainError(errInfo string) DomainError {
	return &domainSentinel{errInfo: errInfo}
}

// RegisterDomainErrorResponse registers a mapping between a domain error errInfo and a shared error response.
func RegisterDomainErrorResponse(errInfo string, resp ErrorResponse) {
	domainErrorResponses[errInfo] = resp
}

// ResolveDomainError converts a domain error into a shared error response if a mapping exists.
func ResolveDomainError(err error) (ErrorResponse, bool) {
	if err == nil {
		return ErrorResponse{}, false
	}

	var domainErr DomainError
	if errors.As(err, &domainErr) {
		if resp, ok := domainErrorResponses[domainErr.Info()]; ok {
			return resp, true
		}
	}
	return ErrorResponse{}, false
}
