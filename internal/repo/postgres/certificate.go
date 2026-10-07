package postgres

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type certificateRepo struct {
	db DBTX
}

func NewCertificateRepository(db DBTX) *certificateRepo {
	return &certificateRepo{
		db: db,
	}
}

// scanCertificateDetail scans a row selected with the certificate columns and
// the student, course and instructor names in the order used below. It works
// for both a single row and rows of a list.
func scanCertificateDetail(row pgx.Row, certificate *models.CertificateDetail) error {
	return row.Scan(
		&certificate.ID,
		&certificate.StudentID,
		&certificate.CourseID,
		&certificate.CompletionDate,
		&certificate.UniqueID,
		&certificate.QRCode,
		&certificate.ObjectKey,
		&certificate.CreatedAt,
		&certificate.StudentName,
		&certificate.CourseTitle,
		&certificate.InstructorName,
	)
}

func (r *certificateRepo) Create(
	ctx context.Context,
	certificate models.CreateCertificate,
) (uuid.UUID, error) {
	query := `
		INSERT INTO certificates (
			id,
			student_id,
			course_id,
			completion_date,
			unique_id,
			qr_code,
			object_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		certificate.StudentID,
		certificate.CourseID,
		certificate.CompletionDate,
		certificate.UniqueID,
		certificate.QRCode,
		certificate.ObjectKey,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, handleError(err, "CreateCertificate", "failed to create certificate")
	}

	return id, nil
}

func (r *certificateRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.CertificateDetail, error) {
	query := `
		SELECT
			ct.id,
			ct.student_id,
			ct.course_id,
			ct.completion_date,
			ct.unique_id,
			ct.qr_code,
			ct.object_key,
			ct.created_at,
			s.first_name || ' ' || s.last_name AS student_name,
			c.title AS course_title,
			i.first_name || ' ' || i.last_name AS instructor_name
		FROM certificates ct
		JOIN users s ON s.id = ct.student_id
		JOIN courses c ON c.id = ct.course_id
		JOIN users i ON i.id = c.instructor_id
		WHERE ct.id = $1;
	`

	var certificate models.CertificateDetail

	err := scanCertificateDetail(r.db.QueryRow(ctx, query, id), &certificate)

	if err != nil {
		return nil, handleError(err, "GetCertificateByID", "failed to get certificate")
	}

	return &certificate, nil
}

// GetByUniqueID is used by the public verification endpoint.
func (r *certificateRepo) GetByUniqueID(
	ctx context.Context,
	uniqueID string,
) (*models.CertificateDetail, error) {
	query := `
		SELECT
			ct.id,
			ct.student_id,
			ct.course_id,
			ct.completion_date,
			ct.unique_id,
			ct.qr_code,
			ct.object_key,
			ct.created_at,
			s.first_name || ' ' || s.last_name AS student_name,
			c.title AS course_title,
			i.first_name || ' ' || i.last_name AS instructor_name
		FROM certificates ct
		JOIN users s ON s.id = ct.student_id
		JOIN courses c ON c.id = ct.course_id
		JOIN users i ON i.id = c.instructor_id
		WHERE ct.unique_id = $1;
	`

	var certificate models.CertificateDetail

	err := scanCertificateDetail(r.db.QueryRow(ctx, query, uniqueID), &certificate)

	if err != nil {
		return nil, handleError(err, "GetCertificateByUniqueID", "failed to get certificate")
	}

	return &certificate, nil
}

// GetByStudentAndCourse is used to check whether the certificate was already
// issued.
func (r *certificateRepo) GetByStudentAndCourse(
	ctx context.Context,
	studentID uuid.UUID,
	courseID uuid.UUID,
) (*models.Certificate, error) {
	query := `
		SELECT
			id,
			student_id,
			course_id,
			completion_date,
			unique_id,
			qr_code,
			object_key,
			created_at
		FROM certificates
		WHERE student_id = $1
			AND course_id = $2;
	`

	var certificate models.Certificate

	err := r.db.QueryRow(
		ctx,
		query,
		studentID,
		courseID,
	).Scan(
		&certificate.ID,
		&certificate.StudentID,
		&certificate.CourseID,
		&certificate.CompletionDate,
		&certificate.UniqueID,
		&certificate.QRCode,
		&certificate.ObjectKey,
		&certificate.CreatedAt,
	)

	if err != nil {
		return nil, handleError(err, "GetCertificateByStudentAndCourse", "failed to get certificate")
	}

	return &certificate, nil
}

func (r *certificateRepo) GetListByStudentID(
	ctx context.Context,
	studentID uuid.UUID,
) ([]models.CertificateDetail, error) {
	query := `
		SELECT
			ct.id,
			ct.student_id,
			ct.course_id,
			ct.completion_date,
			ct.unique_id,
			ct.qr_code,
			ct.object_key,
			ct.created_at,
			s.first_name || ' ' || s.last_name AS student_name,
			c.title AS course_title,
			i.first_name || ' ' || i.last_name AS instructor_name
		FROM certificates ct
		JOIN users s ON s.id = ct.student_id
		JOIN courses c ON c.id = ct.course_id
		JOIN users i ON i.id = c.instructor_id
		WHERE ct.student_id = $1
		ORDER BY ct.completion_date DESC;
	`

	rows, err := r.db.Query(ctx, query, studentID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetCertificateList",
			"failed to get certificates",
			err,
		)
	}
	defer rows.Close()

	certificates := make([]models.CertificateDetail, 0)

	for rows.Next() {
		var certificate models.CertificateDetail

		if err := scanCertificateDetail(rows, &certificate); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetCertificateList",
				"failed to scan certificates",
				err,
			)
		}

		certificates = append(certificates, certificate)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetCertificateList",
			"failed to read certificates",
			err,
		)
	}

	return certificates, nil
}
