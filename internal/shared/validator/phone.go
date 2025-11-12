package validator

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

var (
	// phoneRegex matches Korean mobile numbers
	// Formats: 010-1234-5678 or 01012345678
	phoneRegex = regexp.MustCompile(`^01[0-9]-?[0-9]{4}-?[0-9]{4}$`)
)

// ValidatePhone validates a Korean mobile phone number
// This is a common validator used across multiple domains
func ValidatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	return phoneRegex.MatchString(phone)
}
