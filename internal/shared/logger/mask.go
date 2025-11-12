package logger

import "strings"

// Example: john.doe@gmail.com -> j***@gmail.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}

	username := parts[0]
	domain := parts[1]

	if len(username) == 0 {
		return "***@" + domain
	}

	// Keep only first character of username
	return username[:1] + "***@" + domain
}
