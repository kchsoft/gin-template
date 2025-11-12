package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	sharedContext "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/context"
	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/token"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerScheme        = "Bearer"
)

// JWT error constants (errInfo)
const (
	missingToken  = "MISSING_TOKEN"
	invalidToken  = "INVALID_TOKEN"
	expiredToken  = "EXPIRED_TOKEN"
	invalidClaims = "INVALID_CLAIMS"
)

// Domain errors
var (
	ErrMissingToken  = sharedError.NewDomainError(missingToken)
	ErrInvalidToken  = sharedError.NewDomainError(invalidToken)
	ErrExpiredToken  = sharedError.NewDomainError(expiredToken)
	ErrInvalidClaims = sharedError.NewDomainError(invalidClaims)
)

// Register JWT error responses
func init() {
	sharedError.RegisterDomainErrorResponse(missingToken, sharedError.ErrorResponse{
		Status:  http.StatusUnauthorized,
		Code:    "AUTH-000",
		Message: "로그인을 해주세요.",
	})

	sharedError.RegisterDomainErrorResponse(invalidToken, sharedError.ErrorResponse{
		Status:  http.StatusUnauthorized,
		Code:    "AUTH-000",
		Message: "로그인을 해주세요.",
	})

	sharedError.RegisterDomainErrorResponse(expiredToken, sharedError.ErrorResponse{
		Status:  http.StatusUnauthorized,
		Code:    "AUTH-000",
		Message: "로그인을 해주세요.",
	})

	sharedError.RegisterDomainErrorResponse(invalidClaims, sharedError.ErrorResponse{
		Status:  http.StatusUnauthorized,
		Code:    "AUTH-000",
		Message: "로그인을 해주세요.",
	})
}

func JWT(cfg *config.Config) gin.HandlerFunc {
	tokenManager := token.NewJWTManager(cfg)

	return func(c *gin.Context) {
		// 요청 정보 (로깅용)
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()

		// Step 1: 토큰 추출
		token, err := extractToken(c)
		if err != nil {
			// 에러 발생 지점에서 바로 로깅
			slog.Warn("JWT 토큰 추출 실패",
				"step", "extract_token",
				"error", err.Error(),
				"client_ip", clientIP,
				"method", method,
				"path", path,
				"user_agent", userAgent,
			)
			handleJWTError(c, err)
			return
		}

		// Step 2: 토큰 검증
		claims, err := tokenManager.ValidateToken(token)
		if err != nil {
			// 에러 발생 지점에서 바로 로깅
			slog.Warn("JWT 토큰 검증 실패",
				"step", "validate_token",
				"error", err.Error(),
				"client_ip", clientIP,
				"method", method,
				"path", path,
				"user_agent", userAgent,
			)
			handleJWTError(c, mapTokenError(err))
			return
		}

		// 인증 성공 - Context에 사용자 정보 저장
		c.Set(sharedContext.MemberIDKey, claims.MemberID)
		c.Set(sharedContext.MemberEmailKey, claims.Email)
		c.Next()
	}
}

// handleJWTError handles JWT errors using the standardized error response format
// Note: Logging is done at the point of error detection in JWT() function
func handleJWTError(c *gin.Context, err error) {
	if resp, ok := sharedError.ResolveDomainError(err); ok {
		c.JSON(resp.Status, resp)
	} else {
		// 예상치 못한 에러 → Fallback 응답
		c.JSON(http.StatusUnauthorized, sharedError.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Code:    "AUTH-999",
			Message: "인증에 실패했습니다.",
		})
	}
	c.Abort()
}

func extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return "", ErrMissingToken
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], BearerScheme) {
		return "", ErrInvalidToken
	}

	return parts[1], nil
}

func mapTokenError(err error) error {
	switch {
	case errors.Is(err, token.ErrExpiredToken):
		return ErrExpiredToken
	case errors.Is(err, token.ErrInvalidClaims):
		return ErrInvalidClaims
	default:
		return ErrInvalidToken
	}
}
