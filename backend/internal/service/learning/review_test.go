package learning_test

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (s shop) completedStudent(t *testing.T) *models.User {
	t.Helper()

	student := s.enrolledStudent(t)
	s.finish(t, student)

	return student
}

func (s shop) review(t *testing.T, student *models.User, rating int, comment string) *models.Review {
	t.Helper()

	review, err := s.e.Svc.Review.Create(context.Background(), testutil.ActorOf(student), s.course.ID, models.CreateReview{
		Rating:  rating,
		Comment: &comment,
	})
	require.NoError(t, err)

	return review
}

func TestReview_Create(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.completedStudent(t)
	review := s.review(t, student, 5, "Great course")

	require.Equal(t, student.ID, review.StudentID)
	require.Equal(t, s.course.ID, review.CourseID)
	require.Equal(t, 5, review.Rating)
	require.Equal(t, "Great course", *review.Comment)
	require.Equal(t, student.FirstName+" "+student.LastName, review.StudentName)
}

func TestReview_Create_NeedsACompletedCourse(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	ctx := context.Background()
	req := models.CreateReview{Rating: 5}

	_, err := s.e.Svc.Review.Create(ctx, testutil.ActorOf(s.student(t)), s.course.ID, req)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	started := s.enrolledStudent(t)
	s.mark(t, started, s.lessons[0], true)

	_, err = s.e.Svc.Review.Create(ctx, testutil.ActorOf(started), s.course.ID, req)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestReview_Create_FinalQuizMustBePassedFirst(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	x.finish(t, student)

	_, err := x.e.Svc.Review.Create(context.Background(), testutil.ActorOf(student), x.course.ID, models.CreateReview{Rating: 4})
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	x.pass(t, student)

	_, err = x.e.Svc.Review.Create(context.Background(), testutil.ActorOf(student), x.course.ID, models.CreateReview{Rating: 4})
	require.NoError(t, err)
}

func TestReview_Create_Twice(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.completedStudent(t)
	s.review(t, student, 5, "first")

	_, err := s.e.Svc.Review.Create(context.Background(), testutil.ActorOf(student), s.course.ID, models.CreateReview{Rating: 1})
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestReview_Create_InvalidRating(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.completedStudent(t)

	for _, rating := range []int{0, 6, -1} {
		_, err := s.e.Svc.Review.Create(context.Background(), testutil.ActorOf(student), s.course.ID, models.CreateReview{Rating: rating})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestReview_Create_UnknownCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Review.Create(context.Background(), testutil.ActorOf(s.completedStudent(t)), uuid.New(), models.CreateReview{Rating: 5})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestReview_GetListByCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()

	s.review(t, s.completedStudent(t), 4, "good")
	s.review(t, s.completedStudent(t), 5, "great")

	list, err := s.e.Svc.Review.GetListByCourse(ctx, models.Actor{}, s.course.ID, models.ReviewFilter{})
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.Len(t, list.Items, 2)
	require.Equal(t, 4.5, list.AvgRating)
	require.Equal(t, 1, list.Page)
	require.Equal(t, 10, list.Limit)

	page, err := s.e.Svc.Review.GetListByCourse(ctx, models.Actor{}, s.course.ID, models.ReviewFilter{Limit: 1, Page: 2})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 1)
}

func TestReview_GetListByCourse_NoReviewsYet(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	list, err := s.e.Svc.Review.GetListByCourse(context.Background(), models.Actor{}, s.course.ID, models.ReviewFilter{})
	require.NoError(t, err)
	require.Zero(t, list.Total)
	require.Empty(t, list.Items)
	require.Zero(t, list.AvgRating)
}

func TestReview_GetListByCourse_DraftIsHidden(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()

	err := s.e.Svc.Course.UpdateStatus(ctx, testutil.ActorOf(s.owner), models.UpdateCourseStatus{ID: s.course.ID, Status: models.CourseStatusDraft})
	require.NoError(t, err)

	_, err = s.e.Svc.Review.GetListByCourse(ctx, models.Actor{}, s.course.ID, models.ReviewFilter{})
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = s.e.Svc.Review.GetListByCourse(ctx, testutil.ActorOf(s.owner), s.course.ID, models.ReviewFilter{})
	require.NoError(t, err)
}

func TestReview_Update(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.completedStudent(t)
	review := s.review(t, student, 3, "so-so")

	rating := 5
	comment := "changed my mind"

	updated, err := s.e.Svc.Review.Update(context.Background(), testutil.ActorOf(student), review.ID, models.UpdateReview{Rating: &rating, Comment: &comment})
	require.NoError(t, err)
	require.Equal(t, 5, updated.Rating)
	require.Equal(t, comment, *updated.Comment)
}

func TestReview_Update_InvalidRating(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.completedStudent(t)
	review := s.review(t, student, 3, "so-so")
	rating := 9

	_, err := s.e.Svc.Review.Update(context.Background(), testutil.ActorOf(student), review.ID, models.UpdateReview{Rating: &rating})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestReview_Update_OnlyTheAuthor(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	review := s.review(t, s.completedStudent(t), 3, "so-so")
	rating := 1

	for _, other := range []models.Actor{testutil.ActorOf(s.completedStudent(t)), testutil.ActorOf(s.owner), testutil.AdminActor()} {
		_, err := s.e.Svc.Review.Update(context.Background(), other, review.ID, models.UpdateReview{Rating: &rating})
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestReview_Update_NotFound(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	rating := 4

	_, err := s.e.Svc.Review.Update(context.Background(), testutil.AdminActor(), uuid.New(), models.UpdateReview{Rating: &rating})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestReview_Delete_ByTheAuthor(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.completedStudent(t)
	review := s.review(t, student, 3, "so-so")

	require.NoError(t, s.e.Svc.Review.Delete(ctx, testutil.ActorOf(student), review.ID))

	list, err := s.e.Svc.Review.GetListByCourse(ctx, models.Actor{}, s.course.ID, models.ReviewFilter{})
	require.NoError(t, err)
	require.Zero(t, list.Total)

	// the student may write a new review afterwards
	s.review(t, student, 4, "again")
}

func TestReview_Delete_BySuperAdmin(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	review := s.review(t, s.completedStudent(t), 1, "rude words")

	require.NoError(t, s.e.Svc.Review.Delete(context.Background(), testutil.AdminActor(), review.ID))
}

func TestReview_Delete_OthersCannot(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	review := s.review(t, s.completedStudent(t), 3, "so-so")

	for _, other := range []models.Actor{testutil.ActorOf(s.completedStudent(t)), testutil.ActorOf(s.owner)} {
		err := s.e.Svc.Review.Delete(context.Background(), other, review.ID)
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestReview_Delete_NotFound(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	err := s.e.Svc.Review.Delete(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}
