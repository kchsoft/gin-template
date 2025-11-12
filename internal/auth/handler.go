package auth

import (
	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/handler"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *AuthService
}

func NewAuthHandler(authService *AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (a *AuthHandler) Login(c *gin.Context) {
	var request LoginRequest

	// Parse and validate JSON request
	if !handler.BindJSON(c, &request) {
		return
	}

	response, err := a.authService.Login(c.Request.Context(), &request)
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

func (a *AuthHandler) Signup(c *gin.Context) {
	var request SignupRequest

	// Parse and validate JSON request
	if !handler.BindJSON(c, &request) {
		return
	}

	err := a.authService.Signup(c.Request.Context(), &request)
	if err != nil {
		if resp, ok := sharedError.ResolveDomainError(err); ok {
			handler.RespondError(c, err, resp)
			return
		}

		handler.RespondError(c, err, sharedError.InternalServerError)
		return
	}
	c.JSON(201, gin.H{})
}
