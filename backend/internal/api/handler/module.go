package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/catalog"
	"github.com/gin-gonic/gin"
)

type Module struct {
	svc *catalog.Module
}

func NewModule(svc *catalog.Module) *Module {
	return &Module{
		svc: svc,
	}
}

// Public registers the syllabus of a course: it is open for a published
// course.
func (h *Module) Public(group *gin.RouterGroup) {
	group.GET("/courses/:courseId/modules", h.list)
	group.GET("/modules/:moduleId", h.get)
}

func (h *Module) Private(group *gin.RouterGroup) {
	group.POST("/courses/:courseId/modules", h.create)
	group.PUT("/modules/:moduleId", h.update)
	group.PATCH("/modules/:moduleId/order", h.updateOrder)
	group.DELETE("/modules/:moduleId", h.delete)
}

// @Summary List the modules of a course
// @Description Access: anyone; a token, if sent, tells who is asking.
// @Tags Modules
// @Produce json
// @Param courseId path string true "Course id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.ModuleDetail}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/modules [get]
func (h *Module) list(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	modules, err := h.svc.GetListByCourse(c.Request.Context(), middleware.ActorFrom(c), courseID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, modules)
}

// @Summary Get a module
// @Description Access: anyone; a token, if sent, tells who is asking.
// @Tags Modules
// @Produce json
// @Param moduleId path string true "Module id" format(uuid)
// @Success 200 {object} response.Success{data=models.ModuleDetail}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /modules/{moduleId} [get]
func (h *Module) get(c *gin.Context) {
	id, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	module, err := h.svc.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, module)
}

// @Summary Create a module
// @Description Access: Instructor (owner).
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param body body models.CreateModuleRequest true "Request body"
// @Success 201 {object} response.Success{data=models.Module}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /courses/{courseId}/modules [post]
func (h *Module) create(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var req models.CreateModuleRequest

	if !bindJSON(c, &req) {
		return
	}

	module, err := h.svc.Create(c.Request.Context(), middleware.ActorFrom(c), courseID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, module)
}

// @Summary Update a module
// @Description Access: Instructor (owner).
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module id" format(uuid)
// @Param body body models.UpdateModule true "Request body"
// @Success 200 {object} response.Success{data=models.Module}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /modules/{moduleId} [put]
func (h *Module) update(c *gin.Context) {
	id, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	var req models.UpdateModule

	if !bindJSON(c, &req) {
		return
	}

	module, err := h.svc.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, module)
}

// @Summary Move a module
// @Description Access: Instructor (owner).
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module id" format(uuid)
// @Param body body models.UpdateModuleOrder true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /modules/{moduleId}/order [patch]
func (h *Module) updateOrder(c *gin.Context) {
	id, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	var req models.UpdateModuleOrder

	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.UpdateOrder(c.Request.Context(), middleware.ActorFrom(c), id, req.OrderNumber); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the order was changed")
}

// @Summary Delete a module
// @Description Access: Instructor (owner).
// @Tags Modules
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /modules/{moduleId} [delete]
func (h *Module) delete(c *gin.Context) {
	id, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}
