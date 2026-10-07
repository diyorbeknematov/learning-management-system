package handler

import (
	"time"

	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/account"
	"github.com/gin-gonic/gin"
)

type Auth struct {
	svc *account.Auth
}

func NewAuth(svc *account.Auth) *Auth {
	return &Auth{
		svc: svc,
	}
}

// Limit makes the middleware that allows each client address max requests in
// window on a route; name tells the routes apart.
type Limit func(name string, max int, window time.Duration) gin.HandlerFunc

// Public registers the routes that need no access token: they are the way to
// get one. Each of them is limited per client address, the ones that can be
// abused (guessing passwords, filling a mailbox, making accounts) the most.
func (h *Auth) Public(group *gin.RouterGroup, limit Limit) {
	auth := group.Group("/auth")

	auth.POST("/register", limit("register", 5, 10*time.Minute), h.register)
	auth.POST("/login", limit("login", 10, time.Minute), h.login)
	auth.POST("/refresh", limit("refresh", 30, time.Minute), h.refresh)
	auth.POST("/logout", limit("logout", 30, time.Minute), h.logout)
	auth.POST("/forgot-password", limit("forgot-password", 3, 15*time.Minute), h.forgotPassword)
	auth.POST("/reset-password", limit("reset-password", 10, 15*time.Minute), h.resetPassword)
}

// @Summary Register a new student
// @Description Access: anyone.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.RegisterRequest true "Request body"
// @Success 201 {object} response.Success{data=models.TokenPair}
// @Failure 400 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Failure 429 {object} response.Failure
// @Router /auth/register [post]
func (h *Auth) register(c *gin.Context) {
	var req models.RegisterRequest

	if !bindJSON(c, &req) {
		return
	}

	tokens, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, tokens)
}

// @Summary Log in
// @Description Access: anyone.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "Request body"
// @Success 200 {object} response.Success{data=models.LoginResponse}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 429 {object} response.Failure
// @Router /auth/login [post]
func (h *Auth) login(c *gin.Context) {
	var req models.LoginRequest

	if !bindJSON(c, &req) {
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, result)
}

// @Summary Get a new token pair with a refresh token
// @Description Access: anyone.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.RefreshRequest true "Request body"
// @Success 200 {object} response.Success{data=models.TokenPair}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 429 {object} response.Failure
// @Router /auth/refresh [post]
func (h *Auth) refresh(c *gin.Context) {
	var req models.RefreshRequest

	if !bindJSON(c, &req) {
		return
	}

	tokens, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, tokens)
}

// logout ends the session of the refresh token. It needs no access token, which
// may have expired already.
//
// @Summary Log out (delete the refresh token)
// @Description Access: anyone.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.LogoutRequest true "Request body"
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 429 {object} response.Failure
// @Router /auth/logout [post]
func (h *Auth) logout(c *gin.Context) {
	var req models.LogoutRequest

	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}

// @Summary Send a password reset link by email
// @Description Access: anyone.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.ForgotPasswordRequest true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 429 {object} response.Failure
// @Router /auth/forgot-password [post]
func (h *Auth) forgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest

	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "a link to reset the password was sent to the email")
}

// @Summary Set a new password with the reset token
// @Description Access: anyone.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.ResetPasswordRequest true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 429 {object} response.Failure
// @Router /auth/reset-password [post]
func (h *Auth) resetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest

	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the password was changed; log in with the new password")
}
