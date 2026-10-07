package tests

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestCertificate must be called after the student and the course are
// created, so the certificate is removed before them.
func createTestCertificate(t *testing.T, tc *TestContext, studentID, courseID uuid.UUID) (uuid.UUID, string) {
	t.Helper()

	ctx := context.Background()

	uniqueID := "CERT-" + uuid.NewString()

	id, err := tc.Repo.Certificate.Create(ctx, models.CreateCertificate{
		StudentID:      studentID,
		CourseID:       courseID,
		CompletionDate: time.Now(),
		UniqueID:       uniqueID,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM certificates WHERE id = $1`, id)
	})

	return id, uniqueID
}

func TestCertificateRepo_Create(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	id, _ := createTestCertificate(t, tc, studentID, courseID)
	require.NotEqual(t, uuid.Nil, id)
}

func TestCertificateRepo_Create_AlreadyIssued(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestCertificate(t, tc, studentID, courseID)

	_, err := tc.Repo.Certificate.Create(context.Background(), models.CreateCertificate{
		StudentID:      studentID,
		CourseID:       courseID,
		CompletionDate: time.Now(),
		UniqueID:       "CERT-" + uuid.NewString(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestCertificateRepo_Create_CourseNotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)

	_, err := tc.Repo.Certificate.Create(context.Background(), models.CreateCertificate{
		StudentID:      studentID,
		CourseID:       uuid.New(),
		CompletionDate: time.Now(),
		UniqueID:       "CERT-" + uuid.NewString(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCertificateRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id, uniqueID := createTestCertificate(t, tc, studentID, courseID)

	certificate, err := tc.Repo.Certificate.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, certificate.ID)
	require.Equal(t, uniqueID, certificate.UniqueID)
	require.NotEmpty(t, certificate.StudentName)
	require.NotEmpty(t, certificate.CourseTitle)
	require.NotEmpty(t, certificate.InstructorName)
}

func TestCertificateRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Certificate.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCertificateRepo_GetByUniqueID(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id, uniqueID := createTestCertificate(t, tc, studentID, courseID)

	certificate, err := tc.Repo.Certificate.GetByUniqueID(context.Background(), uniqueID)

	require.NoError(t, err)
	require.Equal(t, id, certificate.ID)
	require.NotEmpty(t, certificate.InstructorName)
}

func TestCertificateRepo_GetByUniqueID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Certificate.GetByUniqueID(context.Background(), "CERT-missing")

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCertificateRepo_GetByStudentAndCourse(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id, _ := createTestCertificate(t, tc, studentID, courseID)

	certificate, err := tc.Repo.Certificate.GetByStudentAndCourse(context.Background(), studentID, courseID)

	require.NoError(t, err)
	require.Equal(t, id, certificate.ID)
}

func TestCertificateRepo_GetByStudentAndCourse_NotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	_, err := tc.Repo.Certificate.GetByStudentAndCourse(context.Background(), studentID, courseID)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCertificateRepo_GetListByStudentID(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	first := createTestCourse(t, tc, "course_"+uuid.NewString())
	second := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestCertificate(t, tc, studentID, first)
	createTestCertificate(t, tc, studentID, second)

	certificates, err := tc.Repo.Certificate.GetListByStudentID(context.Background(), studentID)

	require.NoError(t, err)
	require.Len(t, certificates, 2)
	require.NotEmpty(t, certificates[0].CourseTitle)
}
