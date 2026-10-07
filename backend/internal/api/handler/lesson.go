package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/catalog"
	"github.com/gin-gonic/gin"
)

// Lesson has the routes of lessons and of their materials.
type Lesson struct {
	svc *catalog.Lesson
}

func NewLesson(svc *catalog.Lesson) *Lesson {
	return &Lesson{
		svc: svc,
	}
}

// Public registers the lessons of the syllabus and the materials. The
// materials are here too, because a preview lesson is open to visitors; the
// service decides who may open the others.
func (h *Lesson) Public(group *gin.RouterGroup) {
	group.GET("/modules/:moduleId/lessons", h.list)
	group.GET("/lessons/:lessonId", h.get)

	group.GET("/lessons/:lessonId/materials", h.materials)
	group.GET("/materials/:materialId", h.material)
}

func (h *Lesson) Private(group *gin.RouterGroup) {
	group.POST("/modules/:moduleId/lessons", h.create)
	group.PUT("/lessons/:lessonId", h.update)
	group.PATCH("/lessons/:lessonId/order", h.updateOrder)
	group.DELETE("/lessons/:lessonId", h.delete)

	group.POST("/lessons/:lessonId/materials", h.createMaterial)
	group.PUT("/materials/:materialId", h.updateMaterial)
	group.DELETE("/materials/:materialId", h.deleteMaterial)
}

// @Summary List the lessons of a module
// @Description Access: anyone; a token, if sent, tells who is asking.
// @Tags Lessons
// @Produce json
// @Param moduleId path string true "Module id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.Lesson}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /modules/{moduleId}/lessons [get]
func (h *Lesson) list(c *gin.Context) {
	moduleID, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	lessons, err := h.svc.GetListByModule(c.Request.Context(), middleware.ActorFrom(c), moduleID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, lessons)
}

// @Summary Get a lesson
// @Description Access: anyone; a token, if sent, tells who is asking.
// @Tags Lessons
// @Produce json
// @Param lessonId path string true "Lesson id" format(uuid)
// @Success 200 {object} response.Success{data=models.Lesson}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId} [get]
func (h *Lesson) get(c *gin.Context) {
	id, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	lesson, err := h.svc.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, lesson)
}

// @Summary Create a lesson
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module id" format(uuid)
// @Param body body models.CreateLessonRequest true "Request body"
// @Success 201 {object} response.Success{data=models.Lesson}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /modules/{moduleId}/lessons [post]
func (h *Lesson) create(c *gin.Context) {
	moduleID, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	var req models.CreateLessonRequest

	if !bindJSON(c, &req) {
		return
	}

	lesson, err := h.svc.Create(c.Request.Context(), middleware.ActorFrom(c), moduleID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, lesson)
}

// @Summary Update a lesson
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lessonId path string true "Lesson id" format(uuid)
// @Param body body models.UpdateLesson true "Request body"
// @Success 200 {object} response.Success{data=models.Lesson}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId} [put]
func (h *Lesson) update(c *gin.Context) {
	id, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	var req models.UpdateLesson

	if !bindJSON(c, &req) {
		return
	}

	lesson, err := h.svc.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, lesson)
}

// @Summary Move a lesson
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lessonId path string true "Lesson id" format(uuid)
// @Param body body models.UpdateLessonOrder true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId}/order [patch]
func (h *Lesson) updateOrder(c *gin.Context) {
	id, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	var req models.UpdateLessonOrder

	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.UpdateOrder(c.Request.Context(), middleware.ActorFrom(c), id, req.OrderNumber); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the order was changed")
}

// @Summary Delete a lesson
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Produce json
// @Security BearerAuth
// @Param lessonId path string true "Lesson id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId} [delete]
func (h *Lesson) delete(c *gin.Context) {
	id, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}

// @Summary List the materials of a lesson (a lesson that is not a preview needs an enrolled student, the owner or SuperAdmin)
// @Description Access: anyone; a token, if sent, tells who is asking.
// @Tags Lessons
// @Produce json
// @Param lessonId path string true "Lesson id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.LessonMaterial}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId}/materials [get]
func (h *Lesson) materials(c *gin.Context) {
	lessonID, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	materials, err := h.svc.GetMaterials(c.Request.Context(), middleware.ActorFrom(c), lessonID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, materials)
}

// @Summary Get a material (same rule as the list)
// @Description Access: anyone; a token, if sent, tells who is asking.
// @Tags Lessons
// @Produce json
// @Param materialId path string true "Material id" format(uuid)
// @Success 200 {object} response.Success{data=models.LessonMaterial}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /materials/{materialId} [get]
func (h *Lesson) material(c *gin.Context) {
	id, ok := pathID(c, "materialId")
	if !ok {
		return
	}

	material, err := h.svc.GetMaterial(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, material)
}

// @Summary Add a material to a lesson
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lessonId path string true "Lesson id" format(uuid)
// @Param body body models.CreateLessonMaterial true "Request body"
// @Success 201 {object} response.Success{data=models.LessonMaterial}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /lessons/{lessonId}/materials [post]
func (h *Lesson) createMaterial(c *gin.Context) {
	lessonID, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	var req models.CreateLessonMaterial

	if !bindJSON(c, &req) {
		return
	}

	material, err := h.svc.CreateMaterial(c.Request.Context(), middleware.ActorFrom(c), lessonID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, material)
}

// @Summary Update a material
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param materialId path string true "Material id" format(uuid)
// @Param body body models.UpdateLessonMaterial true "Request body"
// @Success 200 {object} response.Success{data=models.LessonMaterial}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /materials/{materialId} [put]
func (h *Lesson) updateMaterial(c *gin.Context) {
	id, ok := pathID(c, "materialId")
	if !ok {
		return
	}

	var req models.UpdateLessonMaterial

	if !bindJSON(c, &req) {
		return
	}

	material, err := h.svc.UpdateMaterial(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, material)
}

// @Summary Delete a material
// @Description Access: Instructor (owner).
// @Tags Lessons
// @Produce json
// @Security BearerAuth
// @Param materialId path string true "Material id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /materials/{materialId} [delete]
func (h *Lesson) deleteMaterial(c *gin.Context) {
	id, ok := pathID(c, "materialId")
	if !ok {
		return
	}

	if err := h.svc.DeleteMaterial(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}
