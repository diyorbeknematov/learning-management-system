package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/catalog"
	"github.com/gin-gonic/gin"
)

type Category struct {
	svc *catalog.Category
}

func NewCategory(svc *catalog.Category) *Category {
	return &Category{
		svc: svc,
	}
}

// Public registers the categories that everybody can read.
func (h *Category) Public(group *gin.RouterGroup) {
	group.GET("/categories", h.list)
	group.GET("/categories/:id", h.get)
}

// Private registers the changes of categories, for the SuperAdmin.
func (h *Category) Private(group *gin.RouterGroup) {
	group.POST("/categories", h.create)
	group.PUT("/categories/:id", h.update)
	group.DELETE("/categories/:id", h.delete)
}

// @Summary List categories
// @Description Access: anyone.
// @Tags Categories
// @Produce json
// @Param query query models.CategoryFilter false "Query parameters"
// @Success 200 {object} response.Success{data=handler.CategoryList}
// @Failure 400 {object} response.Failure
// @Router /categories [get]
func (h *Category) list(c *gin.Context) {
	var filter models.CategoryFilter

	if !bindQuery(c, &filter) {
		return
	}

	categories, err := h.svc.GetList(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, categories)
}

// @Summary Get a category
// @Description Access: anyone.
// @Tags Categories
// @Produce json
// @Param id path string true "Id" format(uuid)
// @Success 200 {object} response.Success{data=models.Category}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /categories/{id} [get]
func (h *Category) get(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	category, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, category)
}

// @Summary Create a category
// @Description Access: SuperAdmin.
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateCategory true "Request body"
// @Success 201 {object} response.Success{data=models.Category}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /categories [post]
func (h *Category) create(c *gin.Context) {
	var req models.CreateCategory

	if !bindJSON(c, &req) {
		return
	}

	category, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, category)
}

// @Summary Update a category
// @Description Access: SuperAdmin.
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id" format(uuid)
// @Param body body models.UpdateCategory true "Request body"
// @Success 200 {object} response.Success{data=models.Category}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /categories/{id} [put]
func (h *Category) update(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req models.UpdateCategory

	if !bindJSON(c, &req) {
		return
	}

	req.ID = id

	category, err := h.svc.Update(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, category)
}

// @Summary Delete a category
// @Description Access: SuperAdmin.
// @Tags Categories
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /categories/{id} [delete]
func (h *Category) delete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}
