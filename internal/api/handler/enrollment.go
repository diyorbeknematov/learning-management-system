package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/learning"
	"github.com/gin-gonic/gin"
)

// Enrollment has the routes of enrollments, of the progress of a student, and
// of the instructor's view of the students of a course.
type Enrollment struct {
	enrollments *learning.Enrollment
	progress    *learning.Progress
}

func NewEnrollment(enrollments *learning.Enrollment, progress *learning.Progress) *Enrollment {
	return &Enrollment{
		enrollments: enrollments,
		progress:    progress,
	}
}

func (h *Enrollment) Private(group *gin.RouterGroup) {
	group.POST("/courses/:courseId/enrollments", h.enroll)
	group.GET("/courses/:courseId/enrollments", h.roster)

	group.GET("/enrollments/me", h.mine)
	group.GET("/enrollments/:enrollmentId", h.get)
	group.PATCH("/enrollments/:enrollmentId/status", h.updateStatus)

	group.GET("/courses/:courseId/progress", h.courseProgress)
	group.GET("/lessons/:lessonId/progress", h.lessonProgress)
	group.POST("/lessons/:lessonId/progress", h.setLessonProgress)

	group.GET("/courses/:courseId/students", h.students)
	group.GET("/courses/:courseId/students/:studentId/progress", h.studentProgress)
}

// @Summary Enroll in a course (a paid course also creates the payment)
// @Description Access: Student.
// @Tags Enrollments
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Success 201 {object} response.Success{data=models.Enrollment}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /courses/{courseId}/enrollments [post]
func (h *Enrollment) enroll(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	enrollment, err := h.enrollments.Enroll(c.Request.Context(), middleware.ActorFrom(c), courseID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, enrollment)
}

// @Summary List the enrollments of a course
// @Description Access: Instructor (owner).
// @Tags Enrollments
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param query query models.EnrollmentFilter false "Query parameters"
// @Success 200 {object} response.Success{data=handler.EnrollmentRosterItemList}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/enrollments [get]
func (h *Enrollment) roster(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var filter models.EnrollmentFilter

	if !bindQuery(c, &filter) {
		return
	}

	list, err := h.enrollments.GetListByCourse(c.Request.Context(), middleware.ActorFrom(c), courseID, filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, list)
}

// mine is the dashboard of the student: the courses with their progress.
//
// @Summary List my enrollments
// @Description Access: Student.
// @Tags Enrollments
// @Produce json
// @Security BearerAuth
// @Param query query models.EnrollmentFilter false "Query parameters"
// @Success 200 {object} response.Success{data=handler.MyEnrollmentList}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /enrollments/me [get]
func (h *Enrollment) mine(c *gin.Context) {
	var filter models.EnrollmentFilter

	if !bindQuery(c, &filter) {
		return
	}

	list, err := h.enrollments.GetMyList(c.Request.Context(), middleware.ActorFrom(c), filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, list)
}

// @Summary Get an enrollment
// @Description Access: owner student, course instructor or SuperAdmin.
// @Tags Enrollments
// @Produce json
// @Security BearerAuth
// @Param enrollmentId path string true "Enrollment id" format(uuid)
// @Success 200 {object} response.Success{data=models.Enrollment}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /enrollments/{enrollmentId} [get]
func (h *Enrollment) get(c *gin.Context) {
	id, ok := pathID(c, "enrollmentId")
	if !ok {
		return
	}

	enrollment, err := h.enrollments.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, enrollment)
}

// @Summary Change the status of an enrollment
// @Description Access: owner student, course instructor or SuperAdmin.
// @Tags Enrollments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param enrollmentId path string true "Enrollment id" format(uuid)
// @Param body body models.UpdateEnrollmentStatus true "Request body"
// @Success 200 {object} response.Success{data=response.MessageData}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /enrollments/{enrollmentId}/status [patch]
func (h *Enrollment) updateStatus(c *gin.Context) {
	id, ok := pathID(c, "enrollmentId")
	if !ok {
		return
	}

	var req models.UpdateEnrollmentStatus

	if !bindJSON(c, &req) {
		return
	}

	req.ID = id

	if err := h.enrollments.UpdateStatus(c.Request.Context(), middleware.ActorFrom(c), req); err != nil {
		response.Fail(c, err)

		return
	}

	response.Message(c, "the status of the enrollment was changed")
}

// @Summary My progress in a course
// @Description Access: Student.
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Success 200 {object} response.Success{data=models.CourseProgress}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/progress [get]
func (h *Enrollment) courseProgress(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	progress, err := h.progress.GetCourseProgress(c.Request.Context(), middleware.ActorFrom(c), courseID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, progress)
}

// @Summary My progress in a lesson
// @Description Access: Student.
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param lessonId path string true "Lesson id" format(uuid)
// @Success 200 {object} response.Success{data=models.LessonProgress}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId}/progress [get]
func (h *Enrollment) lessonProgress(c *gin.Context) {
	lessonID, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	progress, err := h.progress.GetLessonProgress(c.Request.Context(), middleware.ActorFrom(c), lessonID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, progress)
}

// lessonProgressRequest has a pointer, so that a body without "completed" is an
// error instead of silently meaning "not done".
type lessonProgressRequest struct {
	Completed *bool `json:"completed" validate:"required"`
}

// @Summary Mark a lesson done or not done (finishing the course issues the certificate)
// @Description Access: Student.
// @Tags Progress
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lessonId path string true "Lesson id" format(uuid)
// @Param body body lessonProgressRequest true "Request body"
// @Success 200 {object} response.Success{data=models.LessonProgress}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /lessons/{lessonId}/progress [post]
func (h *Enrollment) setLessonProgress(c *gin.Context) {
	lessonID, ok := pathID(c, "lessonId")
	if !ok {
		return
	}

	var req lessonProgressRequest

	if !bindJSON(c, &req) {
		return
	}

	progress, err := h.progress.SetLessonProgress(c.Request.Context(), middleware.ActorFrom(c), lessonID, *req.Completed)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, progress)
}

// @Summary List the students of a course with their progress
// @Description Access: Instructor (owner).
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param query query models.EnrollmentFilter false "Query parameters"
// @Success 200 {object} response.Success{data=handler.StudentProgressList}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/students [get]
func (h *Enrollment) students(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var filter models.EnrollmentFilter

	if !bindQuery(c, &filter) {
		return
	}

	list, err := h.progress.GetStudents(c.Request.Context(), middleware.ActorFrom(c), courseID, filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, list)
}

// @Summary Progress of one student in a course
// @Description Access: Instructor (owner).
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param studentId path string true "Student id" format(uuid)
// @Success 200 {object} response.Success{data=models.CourseProgress}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/students/{studentId}/progress [get]
func (h *Enrollment) studentProgress(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	studentID, ok := pathID(c, "studentId")
	if !ok {
		return
	}

	progress, err := h.progress.GetStudentProgress(c.Request.Context(), middleware.ActorFrom(c), courseID, studentID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, progress)
}
