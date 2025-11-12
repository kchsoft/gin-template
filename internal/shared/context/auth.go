package context

import (
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/logger"
	"net/http"
	"strconv"

	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/gin-gonic/gin"
)

// Context keys for storing user authentication information
const (
	MemberIDKey    = "member_id"
	MemberEmailKey = "member_email"
)

func GetMemberID(c *gin.Context) (uint32, bool) {
	memberID, exists := c.Get(MemberIDKey)
	if !exists {
		return 0, false
	}

	idStr, ok := memberID.(string)
	if !ok {
		return 0, false
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}

	return uint32(id), true
}

// RequireMemberID retrieves the authenticated user's ID from the Gin context.
// If the user ID is not found, automatically sends an authentication error response.
// Returns the user ID and true if found, empty string and false if not found (error already sent).
// Use this in most handlers to reduce boilerplate.
func RequireMemberID(c *gin.Context) (uint32, bool) {
	memberID, ok := GetMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, sharedError.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Code:    "AUTH-000",
			Message: "로그인을 해주세요.",
		})
		c.Abort()
		logger.FromContext(c.Request.Context()).Error("[API] context에 회원 ID가 존재하지 않습니다.")
		return 0, false
	}
	return memberID, true
}
