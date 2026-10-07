package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestReview must be called after the student and the course are
// created, so the review is removed before them.
func createTestReview(t *testing.T, tc *TestContext, studentID, courseID uuid.UUID, rating int) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	comment := "review_" + uuid.NewString()

	id, err := tc.Repo.Review.Create(ctx, models.CreateReview{
		StudentID: studentID,
		CourseID:  courseID,
		Rating:    rating,
		Comment:   &comment,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM reviews WHERE id = $1`, id)
	})

	return id
}

func TestReviewRepo_Create(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	id := createTestReview(t, tc, studentID, courseID, 5)
	require.NotEqual(t, uuid.Nil, id)
}

func TestReviewRepo_Create_AlreadyReviewed(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestReview(t, tc, studentID, courseID, 5)

	_, err := tc.Repo.Review.Create(context.Background(), models.CreateReview{
		StudentID: studentID,
		CourseID:  courseID,
		Rating:    4,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestReviewRepo_Create_InvalidRating(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	_, err := tc.Repo.Review.Create(context.Background(), models.CreateReview{
		StudentID: studentID,
		CourseID:  courseID,
		Rating:    6,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestReviewRepo_Create_CourseNotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)

	_, err := tc.Repo.Review.Create(context.Background(), models.CreateReview{
		StudentID: studentID,
		CourseID:  uuid.New(),
		Rating:    5,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestReviewRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestReview(t, tc, studentID, courseID, 4)

	review, err := tc.Repo.Review.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, review.ID)
	require.Equal(t, studentID, review.StudentID)
	require.Equal(t, courseID, review.CourseID)
	require.Equal(t, 4, review.Rating)
	require.NotEmpty(t, review.StudentName)
}

func TestReviewRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Review.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestReviewRepo_GetListByCourseID(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestReview(t, tc, first, courseID, 5)
	createTestReview(t, tc, second, courseID, 3)

	reviews, total, err := tc.Repo.Review.GetListByCourseID(ctx, courseID, models.ReviewFilter{})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, reviews, 2)

	reviews, total, err = tc.Repo.Review.GetListByCourseID(ctx, courseID, models.ReviewFilter{Limit: 1, Page: 2})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, reviews, 1)
}

func TestReviewRepo_GetAverageRating(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	average, err := tc.Repo.Review.GetAverageRating(ctx, courseID)
	require.NoError(t, err)
	require.Zero(t, average)

	createTestReview(t, tc, first, courseID, 4)
	createTestReview(t, tc, second, courseID, 5)

	average, err = tc.Repo.Review.GetAverageRating(ctx, courseID)
	require.NoError(t, err)
	require.Equal(t, 4.5, average)
}

func TestReviewRepo_Update(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestReview(t, tc, studentID, courseID, 3)

	rating := 5
	comment := "changed my mind"

	review, err := tc.Repo.Review.Update(context.Background(), models.UpdateReview{
		ID:      id,
		Rating:  &rating,
		Comment: &comment,
	})

	require.NoError(t, err)
	require.Equal(t, 5, review.Rating)
	require.Equal(t, comment, *review.Comment)
	require.NotEmpty(t, review.StudentName)
}

func TestReviewRepo_Update_InvalidRating(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestReview(t, tc, studentID, courseID, 3)

	rating := 0

	_, err := tc.Repo.Review.Update(context.Background(), models.UpdateReview{ID: id, Rating: &rating})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestReviewRepo_Update_NotFound(t *testing.T) {
	tc := setupTest(t)

	rating := 4

	_, err := tc.Repo.Review.Update(context.Background(), models.UpdateReview{ID: uuid.New(), Rating: &rating})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestReviewRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestReview(t, tc, studentID, courseID, 3)

	require.NoError(t, tc.Repo.Review.Delete(ctx, id))

	_, err := tc.Repo.Review.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)

	err = tc.Repo.Review.Delete(ctx, id)
	appErr, ok = apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}
