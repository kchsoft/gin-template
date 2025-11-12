package validator

import (
	"errors"
	"fmt"

	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/go-playground/validator/v10"
)

// ToErrorResponse converts gin binding/validator errors into a standardized response.
func ToErrorResponse(err error) (*sharedError.ErrorResponse, bool) {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return nil, false
	}

	if len(validationErrors) == 0 {
		return nil, false
	}

	// 첫 번째 validation error만 반환 (사용자 친화적)
	fieldErr := validationErrors[0]
	message := getErrorMessage(fieldErr)

	resp := sharedError.ValidationFailed
	resp.Message = message
	return &resp, true
}

// getErrorMessage returns user-friendly error message for validation error
func getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "필수 항목을 입력해 주세요."
	case "email":
		return "이메일 형식이 올바르지 않습니다."
	case "min":
		return fmt.Sprintf("최소 %s자 이상이어야 합니다.", fe.Param())
	case "max":
		return fmt.Sprintf("최대 %s자까지 입력 가능합니다.", fe.Param())
	case "phone":
		return "휴대폰 번호 형식이 올바르지 않습니다. (010-XXXX-XXXX)"
	default:
		return fmt.Sprintf("'%s' 필드가 올바르지 않습니다.", fe.Field())
	}
}
