package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
)

type financeRepo struct {
	db DBTX
}

func NewFinanceRepository(db DBTX) *financeRepo {
	return &financeRepo{
		db: db,
	}
}

// GetSummary returns the revenue and the expenses of the period together with
// the profit of every day, week or month (filter.GroupBy). The profit status
// (profit or loss) is decided by the service from NetProfit.
func (r *financeRepo) GetSummary(
	ctx context.Context,
	filter models.FinanceFilter,
) (*models.FinanceSummary, error) {
	query := `
		WITH revenue AS (
			SELECT
				date_trunc($1, paid_at) AS period,
				SUM(amount)::float8 AS revenue
			FROM payments
			WHERE status = 'paid'
				AND ($2::timestamptz IS NULL OR paid_at >= $2)
				AND ($3::timestamptz IS NULL OR paid_at <= $3)
			GROUP BY 1
		),
		expenses AS (
			SELECT
				date_trunc($1, created_at) AS period,
				SUM(amount)::float8 AS expenses
			FROM instructor_payouts
			WHERE ($2::timestamptz IS NULL OR created_at >= $2)
				AND ($3::timestamptz IS NULL OR created_at <= $3)
			GROUP BY 1
		)
		SELECT
			COALESCE(r.period, e.period) AS period,
			COALESCE(r.revenue, 0) AS revenue,
			COALESCE(e.expenses, 0) AS expenses,
			COALESCE(r.revenue, 0) - COALESCE(e.expenses, 0) AS profit
		FROM revenue r
		FULL OUTER JOIN expenses e ON e.period = r.period
		ORDER BY 1;
	`

	rows, err := r.db.Query(
		ctx,
		query,
		helpers.PeriodUnit(filter.GroupBy),
		filter.From,
		filter.To,
	)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetFinanceSummary",
			"failed to get finance summary",
			err,
		)
	}
	defer rows.Close()

	summary := models.FinanceSummary{
		Points: make([]models.FinancePoint, 0),
	}

	for rows.Next() {
		var point models.FinancePoint

		err := rows.Scan(
			&point.Period,
			&point.Revenue,
			&point.Expenses,
			&point.Profit,
		)
		if err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetFinanceSummary",
				"failed to scan finance summary",
				err,
			)
		}

		summary.Revenue += point.Revenue
		summary.Expenses += point.Expenses
		summary.Points = append(summary.Points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetFinanceSummary",
			"failed to read finance summary",
			err,
		)
	}

	summary.NetProfit = summary.Revenue - summary.Expenses

	return &summary, nil
}

// GetRevenue returns the paid amounts of the period. Only Revenue is filled
// in the points.
func (r *financeRepo) GetRevenue(
	ctx context.Context,
	filter models.FinanceFilter,
) (*models.RevenueSummary, error) {
	query := `
		SELECT
			date_trunc($1, paid_at) AS period,
			SUM(amount)::float8 AS revenue
		FROM payments
		WHERE status = 'paid'
			AND ($2::timestamptz IS NULL OR paid_at >= $2)
			AND ($3::timestamptz IS NULL OR paid_at <= $3)
		GROUP BY 1
		ORDER BY 1;
	`

	rows, err := r.db.Query(
		ctx,
		query,
		helpers.PeriodUnit(filter.GroupBy),
		filter.From,
		filter.To,
	)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetRevenue",
			"failed to get revenue",
			err,
		)
	}
	defer rows.Close()

	revenue := models.RevenueSummary{
		Points: make([]models.FinancePoint, 0),
	}

	for rows.Next() {
		var point models.FinancePoint

		if err := rows.Scan(&point.Period, &point.Revenue); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetRevenue",
				"failed to scan revenue",
				err,
			)
		}

		revenue.Total += point.Revenue
		revenue.Points = append(revenue.Points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetRevenue",
			"failed to read revenue",
			err,
		)
	}

	return &revenue, nil
}

// GetExpenses returns the instructor payouts of the period. Only Expenses is
// filled in the points; the payouts themselves come from GetPayouts.
func (r *financeRepo) GetExpenses(
	ctx context.Context,
	filter models.FinanceFilter,
) (*models.ExpenseSummary, error) {
	query := `
		SELECT
			date_trunc($1, created_at) AS period,
			SUM(amount)::float8 AS expenses
		FROM instructor_payouts
		WHERE ($2::timestamptz IS NULL OR created_at >= $2)
			AND ($3::timestamptz IS NULL OR created_at <= $3)
		GROUP BY 1
		ORDER BY 1;
	`

	rows, err := r.db.Query(
		ctx,
		query,
		helpers.PeriodUnit(filter.GroupBy),
		filter.From,
		filter.To,
	)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetExpenses",
			"failed to get expenses",
			err,
		)
	}
	defer rows.Close()

	expenses := models.ExpenseSummary{
		Points: make([]models.FinancePoint, 0),
	}

	for rows.Next() {
		var point models.FinancePoint

		if err := rows.Scan(&point.Period, &point.Expenses); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetExpenses",
				"failed to scan expenses",
				err,
			)
		}

		expenses.Total += point.Expenses
		expenses.Points = append(expenses.Points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetExpenses",
			"failed to read expenses",
			err,
		)
	}

	return &expenses, nil
}

func (r *financeRepo) GetPayouts(
	ctx context.Context,
	filter models.PaymentFilter,
) ([]models.InstructorPayout, int, error) {
	baseQuery := `
		SELECT
			id,
			instructor_id,
			course_id,
			enrollment_id,
			type,
			value,
			amount,
			created_at
		FROM instructor_payouts
		WHERE TRUE
	`

	countQuery := `
		SELECT COUNT(*)
		FROM instructor_payouts
		WHERE TRUE
	`

	conditions := []string{}
	args := []any{}
	argIndex := 1

	if filter.From != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("created_at >= $%d", argIndex),
		)

		args = append(args, *filter.From)
		argIndex++
	}

	if filter.To != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("created_at <= $%d", argIndex),
		)

		args = append(args, *filter.To)
		argIndex++
	}

	if filter.CourseID != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("course_id = $%d", argIndex),
		)

		args = append(args, *filter.CourseID)
		argIndex++
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")

		baseQuery += whereClause
		countQuery += whereClause
	}

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	baseQuery += fmt.Sprintf(
		" ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetPayouts",
			"failed to get payouts",
			err,
		)
	}
	defer rows.Close()

	payouts := make([]models.InstructorPayout, 0)

	for rows.Next() {
		var payout models.InstructorPayout

		err := rows.Scan(
			&payout.ID,
			&payout.InstructorID,
			&payout.CourseID,
			&payout.EnrollmentID,
			&payout.Type,
			&payout.Value,
			&payout.Amount,
			&payout.CreatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetPayouts",
				"failed to scan payouts",
				err,
			)
		}

		payouts = append(payouts, payout)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetPayouts",
			"failed to read payouts",
			err,
		)
	}

	var total int

	err = r.db.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetPayouts",
			"failed to get total count",
			err,
		)
	}

	return payouts, total, nil
}
