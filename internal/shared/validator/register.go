package validator

import (
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"log/slog"
)

// GetValidator returns the validator instance from Gin binding
func GetValidator() (*validator.Validate, error) {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil, fmt.Errorf("validator 엔진을 가져올 수 없습니다")
	}
	return v, nil
}

// RegisterAll registers all common validators defined in this package
// Domain-specific validators should be registered separately by each domain
func RegisterAll() error {
	v, err := GetValidator()
	if err != nil {
		return fmt.Errorf("validator 엔진 가져오기 실패: %w", err)
	}

	// Register common validators
	if err := v.RegisterValidation("phone", ValidatePhone); err != nil {
		return fmt.Errorf("phone validator 등록 실패: %w", err)
	}

	slog.Info("공통 Validator 등록 완료", "validators", "phone")
	return nil
}
