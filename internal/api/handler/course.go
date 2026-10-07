package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/catalog"
	"github.com/gin-gonic/gin"
)

type Course struct {
	svc *catalog.Course
}

func NewCourse(svc *catalog.Course) *Course {
	return &Course{
		svc: svc,
	}
}

// Public registers the catalog and the page of a course. No token is needed,
// but one that is sent tells the service who is asking: the owner of a draft
// sees it, others do not.
func (h *Course) Public(group *gin.RouterGroup) {
	group.GET("/courses", h.list)
	group.GET("/courses/:courseId", h.get)
}

// Private registers the routes that change courses.
func (h *Course) Private(group *gin.RouterGroup) {
	group.POST("/courses", h.create)
	group.PUT("/courses/:courseId", h.update)
	group.PATCH("/courses/:courseId/status", h.updateStatus)
	group.DELETE("/courses/:courseId", h.delete)
}

// courseQuery is the query of the catalog (?q=go&sort=newest...). The ids are
// read as text and checked, because Gin cannot read a UUID from the address.
type courseQuery struct {
	Search       *string              `form:"q"`
	CategoryID   *string              `form:"category_id" validate:"omitempty,uuid"`
	InstructorID *string              `form:"instructor_id" validate:"omitempty,uuid"`
	Difficulty   *models.Difficulty   `form:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	Language     *string              `form:"language"`
	MinRating    *float64             `form:"min_rating" validate:"omitempty,gte=0,lte=5"`
	PriceType    *string              `form:"price_type" validate:"omitempty,oneof=free paid"`
	Status       *models.CourseStatus `form:"status" validate:"omitempty,oneof=draft published"`
	Sort         string               `form:"sort" validate:"omitempty,oneof=popular rating newest price"`
	Limit        int                  `form:"limit"`
	Page         int                  `form:"page"`
}

func (q courseQuery) filter() models.CourseFilter {
	return models.CourseFilter{
		Search:       q.Search,
		CategoryID:   optionalID(q.CategoryID),
		InstructorID: optionalID(q.InstructorID),
		Difficulty:   q.Difficulty,
		Language:     q.Language,
		MinRating:    q.MinRating,
		PriceType:    q.PriceType,
		Status:       q.Status,
		Sort:         q.Sort,
		Limit:        q.Limit,
		Page:         q.Page,
	}
}

// @Summary Search the catalog
// @Description Access: anyone; a token, if sent, tells who is asking (visitors see published courses only).
// @Tags Courses
// @Produce json
// @Param query query courseQuery false "Query parameters"
// @Success 200 {object} response.Success{data=handler.CourseListItemList}
// @Failure 400 {object} response.Failure
// @Router /courses [get]
func (h *Course) list(c *gin.Context) {
	var query courseQuery

	if !bindQuery(c, &query) {
		return
	}

	courses, err := h.svc.GetList(c.Request.Context(), middleware.ActorFrom(c), query.filter())
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, courses)
}

// @Summary Get a course with its outcomes, requirements and rating
// @Description Access: anyone; a token, if sent, tells who is asking (a draft is for its owner).
// @Tags Courses
// @Produce json
// @Param courseId path string true "Course id" format(uuid)
// @Success 200 {object} response.Success{data=models.CourseDetail}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId} [get]
func (h *Course) get(c *gin.Context) {
	id, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	course, err := h.svc.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, course)
}

// @Summary Create a course
// @Description Access: Instructor.
// @Tags Courses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateCourseRequest true "Request body"
// @Success 201 {object} response.Success{data=models.CourseDetail}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /courses [post]
func (h *Course) create(c *gin.Context) {
	var req models.CreateCourseRequest

	if !bindJSON(c, &req) {
		return
	}

	course, err := h.svc.Create(c.Request.Context(), middleware.ActorFrom(c), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, course)
}

// @Summary Update a course (outcomes and requirements are replaced)
// @Description Access: Instructor (owner).
// @Tags Courses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param body body models.UpdateCourseRequest true "Request body"
// @Success 200 {object} response.Success{data=models.CourseDetail}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId} [put]
func (h *Course) update(c *gin.Context) {
	id, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var req models.UpdateCourseRequest

	if !bindJSON(c, &req) {
		return
	}

	course, err := h.svc.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, course)
}

// @Summary Publish a course or make it a draft
// @Description Access: Instructor (owner).
// @Tags Courses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param body body models.UpdateCourseStatus true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/status [patch]
func (h *Course) updateStatus(c *gin.Context) {
	id, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var req models.UpdateCourseStatus

	if !bindJSON(c, &req) {
		return
	}

	req.ID = id

	if err := h.svc.UpdateStatus(c.Request.Context(), middleware.ActorFrom(c), req); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the status of the course was changed")
}

// @Summary Delete a course
// @Description Access: Instructor (owner).
// @Tags Courses
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId} [delete]
func (h *Course) delete(c *gin.Context) {
	id, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}
