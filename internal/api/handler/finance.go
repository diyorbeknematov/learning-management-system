package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/finance"
	"github.com/gin-gonic/gin"
)

// Finance has the routes of the money: payments, and the revenue, expenses and
// profit of the platform.
type Finance struct {
	finance  *finance.Finance
	payments *finance.Payment
}

func NewFinance(finance *finance.Finance, payments *finance.Payment) *Finance {
	return &Finance{
		finance:  finance,
		payments: payments,
	}
}

func (h *Finance) Private(group *gin.RouterGroup) {
	group.GET("/payments", h.listPayments)
	group.GET("/payments/:paymentId", h.getPayment)
	group.GET("/enrollments/:enrollmentId/payment", h.enrollmentPayment)

	group.GET("/finance", h.summary)
	group.GET("/finance/revenue", h.revenue)
	group.GET("/finance/expenses", h.expenses)
}

// paymentQuery is the query of the list of payments; the course id is read as
// text, like in courseQuery.
type paymentQuery struct {
	periodQuery
	CourseID *string `form:"course_id" validate:"omitempty,uuid"`
	Limit    int     `form:"limit"`
	Page     int     `form:"page"`
}

// @Summary List payments
// @Description Access: SuperAdmin.
// @Tags Payments
// @Produce json
// @Security BearerAuth
// @Param query query paymentQuery false "Query parameters"
// @Success 200 {object} response.Success{data=handler.PaymentList}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /payments [get]
func (h *Finance) listPayments(c *gin.Context) {
	var query paymentQuery

	if !bindQuery(c, &query) {
		return
	}

	dates, ok := period(c, query.periodQuery)
	if !ok {
		return
	}

	list, err := h.payments.GetList(c.Request.Context(), middleware.ActorFrom(c), models.PaymentFilter{
		DateRange: dates,
		CourseID:  optionalID(query.CourseID),
		Limit:     query.Limit,
		Page:      query.Page,
	})
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, list)
}

// @Summary Get a payment
// @Description Access: owner student, course instructor or SuperAdmin.
// @Tags Payments
// @Produce json
// @Security BearerAuth
// @Param paymentId path string true "Payment id" format(uuid)
// @Success 200 {object} response.Success{data=models.Payment}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /payments/{paymentId} [get]
func (h *Finance) getPayment(c *gin.Context) {
	id, ok := pathID(c, "paymentId")
	if !ok {
		return
	}

	payment, err := h.payments.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, payment)
}

// @Summary Get the payment of an enrollment
// @Description Access: owner student, course instructor or SuperAdmin.
// @Tags Payments
// @Produce json
// @Security BearerAuth
// @Param enrollmentId path string true "Enrollment id" format(uuid)
// @Success 200 {object} response.Success{data=models.Payment}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /enrollments/{enrollmentId}/payment [get]
func (h *Finance) enrollmentPayment(c *gin.Context) {
	enrollmentID, ok := pathID(c, "enrollmentId")
	if !ok {
		return
	}

	payment, err := h.payments.GetByEnrollment(c.Request.Context(), middleware.ActorFrom(c), enrollmentID)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, payment)
}

// financeQuery is the query of the finance routes: a period and how to group
// the numbers.
type financeQuery struct {
	periodQuery
	GroupBy string `form:"group_by" validate:"omitempty,oneof=day week month"`
}

func (q financeQuery) filter(c *gin.Context) (models.FinanceFilter, bool) {
	dates, ok := period(c, q.periodQuery)
	if !ok {
		return models.FinanceFilter{}, false
	}

	return models.FinanceFilter{DateRange: dates, GroupBy: q.GroupBy}, true
}

// @Summary Revenue, instructor payouts and the net
// @Description Access: SuperAdmin.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param query query financeQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.FinanceSummary}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /finance [get]
func (h *Finance) summary(c *gin.Context) {
	var query financeQuery

	if !bindQuery(c, &query) {
		return
	}

	filter, ok := query.filter(c)
	if !ok {
		return
	}

	summary, err := h.finance.GetSummary(c.Request.Context(), middleware.ActorFrom(c), filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, summary)
}

// @Summary Revenue by period and by course
// @Description Access: SuperAdmin.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param query query financeQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.RevenueSummary}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /finance/revenue [get]
func (h *Finance) revenue(c *gin.Context) {
	var query financeQuery

	if !bindQuery(c, &query) {
		return
	}

	filter, ok := query.filter(c)
	if !ok {
		return
	}

	revenue, err := h.finance.GetRevenue(c.Request.Context(), middleware.ActorFrom(c), filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, revenue)
}

// @Summary Payouts to instructors by period and by course
// @Description Access: SuperAdmin.
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param query query financeQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.ExpenseSummary}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /finance/expenses [get]
func (h *Finance) expenses(c *gin.Context) {
	var query financeQuery

	if !bindQuery(c, &query) {
		return
	}

	filter, ok := query.filter(c)
	if !ok {
		return
	}

	expenses, err := h.finance.GetExpenses(c.Request.Context(), middleware.ActorFrom(c), filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, expenses)
}
