package auth

import (
	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"net/http"
)

const (
	incorrectEmailPassword = "INCORRECT_EMAIL_PASSWORD" // errInfo
)

var (
	ErrInCorrectEmailPassword = sharedError.NewDomainError(incorrectEmailPassword)
)

func init() {
	sharedError.RegisterDomainErrorResponse(incorrectEmailPassword, sharedError.ErrorResponse{
		Status:  http.StatusBadRequest,
		Code:    "AUTH-003",
		Message: "이메일 또는 비밀번호가 일치하지 않습니다.",
	})
}
