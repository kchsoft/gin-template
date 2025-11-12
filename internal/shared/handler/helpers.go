package handler

import (
	"net/http"

	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/validator"
	"github.com/gin-gonic/gin"
)

// BindJSON parses and validates JSON request body
// Returns true if binding succeeded, false if failed (response already sent)
//
// Usage:
//
//	var req SignupRequest
//	if !handler.BindJSON(c, &req) {
//	    return
//	}
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		// Add error to context for middleware logging
		c.Error(err)

		// Check if it's a validation error
		if resp, ok := validator.ToErrorResponse(err); ok {
			c.JSON(http.StatusBadRequest, resp)
		} else {
			// JSON parsing error or other binding errors
			c.JSON(sharedError.InvalidRequest.Status, sharedError.InvalidRequest)
		}
		return false
	}
	return true
}

// RespondError sends an error response with logging
//
// Usage:
//
//	if err := service.DoSomething(); err != nil {
//	    handler.RespondError(c, err, sharedError.InternalServerError)
//	    return
//	}
func RespondError(c *gin.Context, err error, errResp sharedError.ErrorResponse) {
	// Add error to context for middleware logging
	c.Error(err)

	// Send error response
	c.JSON(errResp.Status, errResp)
}
