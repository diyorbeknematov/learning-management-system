package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type paymentRepo struct {
	db DBTX
}

func NewPaymentRepository(db DBTX) *paymentRepo {
	return &paymentRepo{
		db: db,
	}
}

// scanPayment scans a row selected with the payment columns in the order used
// below. It works for both a single row and rows of a list.
func scanPayment(row pgx.Row, payment *models.Payment) error {
	return row.Scan(
		&payment.ID,
		&payment.EnrollmentID,
		&payment.Amount,
		&payment.Status,
		&payment.PaidAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
}

func (r *paymentRepo) Create(
	ctx context.Context,
	payment models.CreatePayment,
) (uuid.UUID, error) {
	query := `
		INSERT INTO payments (
			id,
			enrollment_id,
			amount,
			paid_at
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		payment.EnrollmentID,
		payment.Amount,
		payment.PaidAt,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreatePayment",
				"enrollment is already paid",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsCheckViolation(err) {
			return uuid.Nil, apperror.InvalidInput(
				"repository",
				"CreatePayment",
				"payment amount cannot be negative",
				apperror.ErrInvalidInput,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreatePayment",
				"enrollment not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreatePayment",
			"failed to create payment",
			err,
		)
	}

	return id, nil
}

func (r *paymentRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Payment, error) {
	query := `
		SELECT
			id,
			enrollment_id,
			amount,
			status,
			paid_at,
			created_at,
			updated_at
		FROM payments
		WHERE id = $1;
	`

	var payment models.Payment

	err := scanPayment(r.db.QueryRow(ctx, query, id), &payment)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetPaymentByID",
				"payment not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetPaymentByID",
			"failed to get payment",
			err,
		)
	}

	return &payment, nil
}

func (r *paymentRepo) GetByEnrollmentID(
	ctx context.Context,
	enrollmentID uuid.UUID,
) (*models.Payment, error) {
	query := `
		SELECT
			id,
			enrollment_id,
			amount,
			status,
			paid_at,
			created_at,
			updated_at
		FROM payments
		WHERE enrollment_id = $1;
	`

	var payment models.Payment

	err := scanPayment(r.db.QueryRow(ctx, query, enrollmentID), &payment)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetPaymentByEnrollmentID",
				"payment not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetPaymentByEnrollmentID",
			"failed to get payment",
			err,
		)
	}

	return &payment, nil
}

func (r *paymentRepo) GetList(
	ctx context.Context,
	filter models.PaymentFilter,
) ([]models.Payment, int, error) {
	baseQuery := `
		SELECT
			p.id,
			p.enrollment_id,
			p.amount,
			p.status,
			p.paid_at,
			p.created_at,
			p.updated_at
		FROM payments p
		JOIN enrollments e ON e.id = p.enrollment_id
		WHERE TRUE
	`

	countQuery := `
		SELECT COUNT(*)
		FROM payments p
		JOIN enrollments e ON e.id = p.enrollment_id
		WHERE TRUE
	`

	conditions := []string{}
	args := []any{}
	argIndex := 1

	if filter.From != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("p.paid_at >= $%d", argIndex),
		)

		args = append(args, *filter.From)
		argIndex++
	}

	if filter.To != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("p.paid_at <= $%d", argIndex),
		)

		args = append(args, *filter.To)
		argIndex++
	}

	if filter.CourseID != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("e.course_id = $%d", argIndex),
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
		" ORDER BY p.paid_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetPaymentList",
			"failed to get payments",
			err,
		)
	}
	defer rows.Close()

	payments := make([]models.Payment, 0)

	for rows.Next() {
		var payment models.Payment

		if err := scanPayment(rows, &payment); err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetPaymentList",
				"failed to scan payments",
				err,
			)
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetPaymentList",
			"failed to read payments",
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
			"GetPaymentList",
			"failed to get total count",
			err,
		)
	}

	return payments, total, nil
}

// CreatePayout stores the instructor's share of an enrollment. It is created
// together with the payment (one payout per enrollment).
func (r *paymentRepo) CreatePayout(
	ctx context.Context,
	payout models.CreateInstructorPayout,
) (uuid.UUID, error) {
	query := `
		INSERT INTO instructor_payouts (
			id,
			instructor_id,
			course_id,
			enrollment_id,
			type,
			value,
			amount
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		payout.InstructorID,
		payout.CourseID,
		payout.EnrollmentID,
		payout.Type,
		payout.Value,
		payout.Amount,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreatePayout",
				"payout for this enrollment already exists",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsCheckViolation(err) {
			return uuid.Nil, apperror.InvalidInput(
				"repository",
				"CreatePayout",
				"invalid payout value or amount",
				apperror.ErrInvalidInput,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreatePayout",
				"instructor, course or enrollment not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreatePayout",
			"failed to create payout",
			err,
		)
	}

	return id, nil
}
