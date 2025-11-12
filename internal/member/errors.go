package member

import (
	"net/http"

	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
)

const (
	memberAlreadyExists = "MEMBER_ALREADY_EXISTS" // errInfo
	memberNotFound      = "MEMBER_NOT_FOUND"      // errInfo
)

var (
	ErrMemberAlreadyExists = sharedError.NewDomainError(memberAlreadyExists)
	ErrMemberNotFound      = sharedError.NewDomainError(memberNotFound)
)

func init() {
	sharedError.RegisterDomainErrorResponse(memberNotFound, sharedError.ErrorResponse{
		Status:  http.StatusNotFound,
		Code:    "MEMBER-001",
		Message: "회원 정보를 찾을 수 없습니다.",
	})

	sharedError.RegisterDomainErrorResponse(memberAlreadyExists, sharedError.ErrorResponse{
		Status:  http.StatusConflict,
		Code:    "MEMBER-002",
		Message: "이미 가입된 사용자입니다.",
	})
}
