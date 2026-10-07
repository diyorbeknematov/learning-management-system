package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/account"
	"github.com/gin-gonic/gin"
)

type User struct {
	svc *account.User
}

func NewUser(svc *account.User) *User {
	return &User{
		svc: svc,
	}
}

// Private registers the routes of users. The routes of the logged in user
// ("me") and the administration of all users are in one group; Casbin tells
// who may call which.
func (h *User) Private(group *gin.RouterGroup) {
	users := group.Group("/users")

	users.GET("", h.list)
	users.POST("", h.create)

	users.GET("/me", h.me)
	users.PUT("/me", h.updateMe)
	users.PUT("/me/password", h.changePassword)

	users.GET("/:id", h.get)
	users.PUT("/:id", h.update)
	users.PATCH("/:id/status", h.updateStatus)
	users.DELETE("/:id", h.delete)
}

// @Summary List users
// @Description Access: SuperAdmin.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param query query models.UserFilter false "Query parameters"
// @Success 200 {object} response.Success{data=handler.UserList}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /users [get]
func (h *User) list(c *gin.Context) {
	var filter models.UserFilter

	if !bindQuery(c, &filter) {
		return
	}

	users, err := h.svc.GetList(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, users)
}

// @Summary Create a user with any role
// @Description Access: SuperAdmin.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateUserRequest true "Request body"
// @Success 201 {object} response.Success{data=models.User}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /users [post]
func (h *User) create(c *gin.Context) {
	var req models.CreateUserRequest

	if !bindJSON(c, &req) {
		return
	}

	user, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, user)
}

// @Summary Get a user
// @Description Access: SuperAdmin.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id" format(uuid)
// @Success 200 {object} response.Success{data=models.User}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /users/{id} [get]
func (h *User) get(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, user)
}

// @Summary Update a user (a role change ends the sessions of the user)
// @Description Access: SuperAdmin.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id" format(uuid)
// @Param body body models.UpdateUserRequest true "Request body"
// @Success 200 {object} response.Success{data=models.User}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /users/{id} [put]
func (h *User) update(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req models.UpdateUserRequest

	if !bindJSON(c, &req) {
		return
	}

	user, err := h.svc.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, user)
}

// @Summary Block or unblock a user (blocking ends the sessions at once)
// @Description Access: SuperAdmin.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id" format(uuid)
// @Param body body models.UpdateUserStatus true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /users/{id}/status [patch]
func (h *User) updateStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req models.UpdateUserStatus

	if !bindJSON(c, &req) {
		return
	}

	req.ID = id

	if err := h.svc.UpdateStatus(c.Request.Context(), middleware.ActorFrom(c), req); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the status was changed")
}

// @Summary Delete a user
// @Description Access: SuperAdmin.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /users/{id} [delete]
func (h *User) delete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}

// @Summary Get my profile
// @Description Access: any logged in user.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Success{data=models.User}
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /users/me [get]
func (h *User) me(c *gin.Context) {
	user, err := h.svc.GetMe(c.Request.Context(), middleware.ActorFrom(c))
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, user)
}

// @Summary Update my profile
// @Description Access: any logged in user.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.UpdateMeRequest true "Request body"
// @Success 200 {object} response.Success{data=models.User}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /users/me [put]
func (h *User) updateMe(c *gin.Context) {
	var req models.UpdateMeRequest

	if !bindJSON(c, &req) {
		return
	}

	user, err := h.svc.UpdateMe(c.Request.Context(), middleware.ActorFrom(c), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, user)
}

// @Summary Change my password (ends the other sessions)
// @Description Access: any logged in user.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.ChangePasswordRequest true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /users/me/password [put]
func (h *User) changePassword(c *gin.Context) {
	var req models.ChangePasswordRequest

	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), middleware.ActorFrom(c), req); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the password was changed; log in again with the new password")
}
