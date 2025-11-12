package member

import (
	sharedContext "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/context"
	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/handler"
	"github.com/gin-gonic/gin"
)

type MemberHandler struct {
	memberService *MemberService
}

func NewMemberHandler(memberService *MemberService) *MemberHandler {
	return &MemberHandler{
		memberService: memberService,
	}
}

func (h *MemberHandler) GetProfile(c *gin.Context) {
	MemberID, ok := sharedContext.RequireMemberID(c)
	if !ok {
		return
	}

	response, err := h.memberService.GetProfile(c.Request.Context(), MemberID)
	if err != nil {
		if resp, ok := sharedError.ResolveDomainError(err); ok {
			handler.RespondError(c, err, resp)
			return
		}

		handler.RespondError(c, err, sharedError.InternalServerError)
		return
	}

	c.JSON(200, response)
}
