package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/learning"
	"github.com/gin-gonic/gin"
)

// Quiz has the routes of quizzes, their questions and the attempts at them.
type Quiz struct {
	quizzes   *learning.Quiz
	questions *learning.Question
	attempts  *learning.Attempt
}

func NewQuiz(quizzes *learning.Quiz, questions *learning.Question, attempts *learning.Attempt) *Quiz {
	return &Quiz{
		quizzes:   quizzes,
		questions: questions,
		attempts:  attempts,
	}
}

func (h *Quiz) Private(group *gin.RouterGroup) {
	group.GET("/courses/:courseId/quizzes", h.listByCourse)
	group.POST("/courses/:courseId/quizzes", h.createForCourse)
	group.GET("/modules/:moduleId/quizzes", h.listByModule)
	group.POST("/modules/:moduleId/quizzes", h.createForModule)
	group.GET("/quizzes/:quizId", h.get)
	group.PUT("/quizzes/:quizId", h.update)
	group.DELETE("/quizzes/:quizId", h.delete)

	group.GET("/quizzes/:quizId/questions", h.listQuestions)
	group.POST("/quizzes/:quizId/questions", h.createQuestion)
	group.GET("/questions/:questionId", h.getQuestion)
	group.PUT("/questions/:questionId", h.updateQuestion)
	group.DELETE("/questions/:questionId", h.deleteQuestion)

	group.POST("/quizzes/:quizId/attempts", h.startAttempt)
	group.GET("/quizzes/:quizId/attempts", h.listAttempts)
	group.GET("/attempts/:attemptId", h.getAttempt)
	group.POST("/attempts/:attemptId/submit", h.submitAttempt)
}

// @Summary List the quizzes of a course
// @Description Access: any logged in user.
// @Tags Quizzes
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.Quiz}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/quizzes [get]
func (h *Quiz) listByCourse(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	quizzes, err := h.quizzes.GetListByCourse(c.Request.Context(), middleware.ActorFrom(c), courseID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, quizzes)
}

// @Summary List the quizzes of a module
// @Description Access: any logged in user.
// @Tags Quizzes
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.Quiz}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /modules/{moduleId}/quizzes [get]
func (h *Quiz) listByModule(c *gin.Context) {
	moduleID, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	quizzes, err := h.quizzes.GetListByModule(c.Request.Context(), middleware.ActorFrom(c), moduleID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, quizzes)
}

// @Summary Create the final quiz of a course
// @Description Access: Instructor (owner).
// @Tags Quizzes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param body body models.CreateQuiz true "Request body"
// @Success 201 {object} response.Success{data=models.Quiz}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /courses/{courseId}/quizzes [post]
func (h *Quiz) createForCourse(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var req models.CreateQuiz

	if !bindJSON(c, &req) {
		return
	}

	quiz, err := h.quizzes.CreateForCourse(c.Request.Context(), middleware.ActorFrom(c), courseID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, quiz)
}

// @Summary Create a quiz of a module
// @Description Access: Instructor (owner).
// @Tags Quizzes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module id" format(uuid)
// @Param body body models.CreateQuiz true "Request body"
// @Success 201 {object} response.Success{data=models.Quiz}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /modules/{moduleId}/quizzes [post]
func (h *Quiz) createForModule(c *gin.Context) {
	moduleID, ok := pathID(c, "moduleId")
	if !ok {
		return
	}

	var req models.CreateQuiz

	if !bindJSON(c, &req) {
		return
	}

	quiz, err := h.quizzes.CreateForModule(c.Request.Context(), middleware.ActorFrom(c), moduleID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, quiz)
}

// @Summary Get a quiz
// @Description Access: any logged in user.
// @Tags Quizzes
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Success 200 {object} response.Success{data=models.Quiz}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /quizzes/{quizId} [get]
func (h *Quiz) get(c *gin.Context) {
	id, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	quiz, err := h.quizzes.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, quiz)
}

// @Summary Update a quiz
// @Description Access: Instructor (owner).
// @Tags Quizzes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Param body body models.UpdateQuiz true "Request body"
// @Success 200 {object} response.Success{data=models.Quiz}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /quizzes/{quizId} [put]
func (h *Quiz) update(c *gin.Context) {
	id, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	var req models.UpdateQuiz

	if !bindJSON(c, &req) {
		return
	}

	quiz, err := h.quizzes.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, quiz)
}

// @Summary Delete a quiz
// @Description Access: Instructor (owner).
// @Tags Quizzes
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /quizzes/{quizId} [delete]
func (h *Quiz) delete(c *gin.Context) {
	id, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	if err := h.quizzes.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}

// @Summary List the questions of a quiz with the right answers
// @Description Access: Instructor (owner).
// @Tags Questions
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.Question}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /quizzes/{quizId}/questions [get]
func (h *Quiz) listQuestions(c *gin.Context) {
	quizID, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	questions, err := h.questions.GetListByQuiz(c.Request.Context(), middleware.ActorFrom(c), quizID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, questions)
}

// @Summary Add a question with its options
// @Description Access: Instructor (owner).
// @Tags Questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Param body body models.CreateQuestionRequest true "Request body"
// @Success 201 {object} response.Success{data=models.Question}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /quizzes/{quizId}/questions [post]
func (h *Quiz) createQuestion(c *gin.Context) {
	quizID, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	var req models.CreateQuestionRequest

	if !bindJSON(c, &req) {
		return
	}

	question, err := h.questions.Create(c.Request.Context(), middleware.ActorFrom(c), quizID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, question)
}

// @Summary Get a question
// @Description Access: Instructor (owner).
// @Tags Questions
// @Produce json
// @Security BearerAuth
// @Param questionId path string true "Question id" format(uuid)
// @Success 200 {object} response.Success{data=models.Question}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /questions/{questionId} [get]
func (h *Quiz) getQuestion(c *gin.Context) {
	id, ok := pathID(c, "questionId")
	if !ok {
		return
	}

	question, err := h.questions.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, question)
}

// @Summary Update a question
// @Description Access: Instructor (owner).
// @Tags Questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param questionId path string true "Question id" format(uuid)
// @Param body body models.UpdateQuestion true "Request body"
// @Success 200 {object} response.Success{data=models.Question}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /questions/{questionId} [put]
func (h *Quiz) updateQuestion(c *gin.Context) {
	id, ok := pathID(c, "questionId")
	if !ok {
		return
	}

	var req models.UpdateQuestion

	if !bindJSON(c, &req) {
		return
	}

	question, err := h.questions.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, question)
}

// @Summary Delete a question
// @Description Access: Instructor (owner).
// @Tags Questions
// @Produce json
// @Security BearerAuth
// @Param questionId path string true "Question id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /questions/{questionId} [delete]
func (h *Quiz) deleteQuestion(c *gin.Context) {
	id, ok := pathID(c, "questionId")
	if !ok {
		return
	}

	if err := h.questions.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}

// startAttempt begins an attempt, or continues the one that is running; both
// answer 200 with the questions in the random order of this attempt.
//
// @Summary Start an attempt (questions and options come shuffled, without the right answers)
// @Description Access: Student.
// @Tags Attempts
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Success 200 {object} response.Success{data=models.AttemptView}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /quizzes/{quizId}/attempts [post]
func (h *Quiz) startAttempt(c *gin.Context) {
	quizID, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	attempt, err := h.attempts.Start(c.Request.Context(), middleware.ActorFrom(c), quizID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, attempt)
}

// @Summary List attempts (own for a student, all for the owner and SuperAdmin)
// @Description Access: any logged in user.
// @Tags Attempts
// @Produce json
// @Security BearerAuth
// @Param quizId path string true "Quiz id" format(uuid)
// @Success 200 {object} response.Success{data=[]models.QuizAttempt}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /quizzes/{quizId}/attempts [get]
func (h *Quiz) listAttempts(c *gin.Context) {
	quizID, ok := pathID(c, "quizId")
	if !ok {
		return
	}

	attempts, err := h.attempts.GetList(c.Request.Context(), middleware.ActorFrom(c), quizID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, attempts)
}

// @Summary Get an attempt
// @Description Access: any logged in user.
// @Tags Attempts
// @Produce json
// @Security BearerAuth
// @Param attemptId path string true "Attempt id" format(uuid)
// @Success 200 {object} response.Success{data=models.AttemptDetail}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /attempts/{attemptId} [get]
func (h *Quiz) getAttempt(c *gin.Context) {
	id, ok := pathID(c, "attemptId")
	if !ok {
		return
	}

	attempt, err := h.attempts.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, attempt)
}

// @Summary Submit the answers and get the score
// @Description Access: Student.
// @Tags Attempts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param attemptId path string true "Attempt id" format(uuid)
// @Param body body models.SubmitAttempt true "Request body"
// @Success 200 {object} response.Success{data=models.AttemptResult}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /attempts/{attemptId}/submit [post]
func (h *Quiz) submitAttempt(c *gin.Context) {
	id, ok := pathID(c, "attemptId")
	if !ok {
		return
	}

	var req models.SubmitAttempt

	if !bindJSON(c, &req) {
		return
	}

	req.AttemptID = id

	result, err := h.attempts.Submit(c.Request.Context(), middleware.ActorFrom(c), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, result)
}
