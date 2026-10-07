package finance

import (
	"context"
	"math"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
)

const (
	// defaultPeriod is used when the request has no dates; without it "by day"
	// would return the whole history.
	defaultPeriod = 30 * 24 * time.Hour

	// payoutsShown is how many of the newest payouts the expenses show.
	payoutsShown = 100

	statusProfit = "profit"
	statusLoss   = "loss"
)

// Finance is the money view of the SuperAdmin: what the courses earned, what
// is paid to the instructors, and what is left.
type Finance struct {
	repo *repo.Repository
}

func NewFinance(deps core.Dependencies) *Finance {
	return &Finance{
		repo: deps.Repo,
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// period checks the dates of a request. Without any date the last 30 days are
// taken.
func period(dates models.DateRange, op string) (models.DateRange, error) {
	if dates.From == nil && dates.To == nil {
		now := time.Now()
		from := now.Add(-defaultPeriod)

		return models.DateRange{From: &from, To: &now}, nil
	}

	if dates.From != nil && dates.To != nil && dates.From.After(*dates.To) {
		return dates, apperror.InvalidInput("service", op, "from must not be after to", apperror.ErrInvalidInput)
	}

	return dates, nil
}

func roundPoints(points []models.FinancePoint) {
	for i := range points {
		points[i].Revenue = round2(points[i].Revenue)
		points[i].Expenses = round2(points[i].Expenses)
		points[i].Profit = round2(points[i].Profit)
	}
}

// GetSummary returns the revenue, the expenses and the net profit of the
// period, with the numbers of every day, week or month. The status is "profit"
// or "loss".
func (s *Finance) GetSummary(ctx context.Context, actor models.Actor, filter models.FinanceFilter) (*models.FinanceSummary, error) {
	if err := core.RequireSuperAdmin(actor, "GetFinanceSummary"); err != nil {
		return nil, err
	}

	var err error

	filter.DateRange, err = period(filter.DateRange, "GetFinanceSummary")
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.Finance.GetSummary(ctx, filter)
	if err != nil {
		return nil, err
	}

	summary.Revenue = round2(summary.Revenue)
	summary.Expenses = round2(summary.Expenses)
	summary.NetProfit = round2(summary.Revenue - summary.Expenses)
	roundPoints(summary.Points)

	summary.Status = statusProfit
	if summary.NetProfit < 0 {
		summary.Status = statusLoss
	}

	return summary, nil
}

// GetRevenue returns what was paid in the period.
func (s *Finance) GetRevenue(ctx context.Context, actor models.Actor, filter models.FinanceFilter) (*models.RevenueSummary, error) {
	if err := core.RequireSuperAdmin(actor, "GetRevenue"); err != nil {
		return nil, err
	}

	var err error

	filter.DateRange, err = period(filter.DateRange, "GetRevenue")
	if err != nil {
		return nil, err
	}

	revenue, err := s.repo.Finance.GetRevenue(ctx, filter)
	if err != nil {
		return nil, err
	}

	revenue.Total = round2(revenue.Total)
	roundPoints(revenue.Points)

	return revenue, nil
}

// GetExpenses returns what is paid to the instructors in the period, with the
// newest payouts.
func (s *Finance) GetExpenses(ctx context.Context, actor models.Actor, filter models.FinanceFilter) (*models.ExpenseSummary, error) {
	if err := core.RequireSuperAdmin(actor, "GetExpenses"); err != nil {
		return nil, err
	}

	var err error

	filter.DateRange, err = period(filter.DateRange, "GetExpenses")
	if err != nil {
		return nil, err
	}

	expenses, err := s.repo.Finance.GetExpenses(ctx, filter)
	if err != nil {
		return nil, err
	}

	payouts, _, err := s.repo.Finance.GetPayouts(ctx, models.PaymentFilter{
		DateRange: filter.DateRange,
		Limit:     payoutsShown,
		Page:      1,
	})
	if err != nil {
		return nil, err
	}

	expenses.Total = round2(expenses.Total)
	expenses.Payouts = payouts
	roundPoints(expenses.Points)

	return expenses, nil
}
