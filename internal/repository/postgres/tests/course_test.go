package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestCourse(t *testing.T, tc *TestContext, title string) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	instructorID := createTestUser(t, tc)
	categoryID := createTestCategory(t, tc)

	difficulty := models.DifficultyBeginner
	language := "uz"

	id, err := tc.Repo.Course.Create(ctx, models.CreateCourse{
		InstructorID: instructorID,
		CategoryID:   categoryID,
		Title:        title,
		Difficulty:   &difficulty,
		Language:     &language,
		Price:        49.5,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM course_learning_outcomes WHERE course_id = $1`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM course_requirements WHERE course_id = $1`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, id)
		deleteTestUser(t, tc, instructorID)
	})

	return id
}

func TestCourseRepo_Create(t *testing.T) {
	tc := setupTest(t)

	id := createTestCourse(t, tc, "course_"+uuid.NewString())
	require.NotEqual(t, uuid.Nil, id)
}

func TestCourseRepo_Create_InvalidCategory(t *testing.T) {
	tc := setupTest(t)

	instructorID := createTestUser(t, tc)
	t.Cleanup(func() { deleteTestUser(t, tc, instructorID) })

	_, err := tc.Repo.Course.Create(context.Background(), models.CreateCourse{
		InstructorID: instructorID,
		CategoryID:   uuid.New(),
		Title:        "bad",
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestCourseRepo_CreateLearningOutcome(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	require.NoError(t, tc.Repo.Course.CreateLearningOutcome(ctx, models.CourseLearningOutcome{
		CourseID: id, Content: "Learn Go", Position: 1,
	}))
	require.NoError(t, tc.Repo.Course.CreateLearningOutcome(ctx, models.CourseLearningOutcome{
		CourseID: id, Content: "Write tests", Position: 2,
	}))

	outcomes, err := tc.Repo.Course.GetLearningOutcomes(ctx, id)

	require.NoError(t, err)
	require.Len(t, outcomes, 2)
	require.Equal(t, "Learn Go", outcomes[0].Content)
	require.Equal(t, "Write tests", outcomes[1].Content)
}

func TestCourseRepo_CreateRequirement(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	require.NoError(t, tc.Repo.Course.CreateRequirement(ctx, models.CourseRequirement{
		CourseID: id, Content: "Basic programming", Position: 1,
	}))

	requirements, err := tc.Repo.Course.GetRequirements(ctx, id)

	require.NoError(t, err)
	require.Len(t, requirements, 1)
	require.Equal(t, "Basic programming", requirements[0].Content)
}

func TestCourseRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	course, err := tc.Repo.Course.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, course.ID)
	require.Equal(t, models.CourseStatusDraft, course.Status)
	require.Equal(t, models.DifficultyBeginner, *course.Difficulty)
	require.Equal(t, 49.5, course.Price)
}

func TestCourseRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Course.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCourseRepo_GetList(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	marker := uuid.NewString()
	id := createTestCourse(t, tc, "listed_"+marker)

	courses, total, err := tc.Repo.Course.GetList(ctx, models.CourseFilter{
		Search: &marker,
		Sort:   "popular",
	})

	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, id, courses[0].ID)
	require.NotEmpty(t, courses[0].CategoryName)
	require.NotEmpty(t, courses[0].InstructorName)
	require.Zero(t, courses[0].ReviewCount)
}

func TestCourseRepo_GetList_Filters(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	marker := uuid.NewString()
	id := createTestCourse(t, tc, "filtered_"+marker)

	published := models.CourseStatusPublished
	_, total, err := tc.Repo.Course.GetList(ctx, models.CourseFilter{Search: &marker, Status: &published})
	require.NoError(t, err)
	require.Zero(t, total)

	require.NoError(t, tc.Repo.Course.UpdateStatus(ctx, models.UpdateCourseStatus{
		ID: id, Status: models.CourseStatusPublished,
	}))

	paid, free := "paid", "free"
	minRating := 0.0

	_, total, err = tc.Repo.Course.GetList(ctx, models.CourseFilter{
		Search: &marker, Status: &published, PriceType: &paid, MinRating: &minRating, Sort: "rating",
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)

	_, total, err = tc.Repo.Course.GetList(ctx, models.CourseFilter{Search: &marker, PriceType: &free})
	require.NoError(t, err)
	require.Zero(t, total)
}

func TestCourseRepo_Update(t *testing.T) {
	tc := setupTest(t)

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	title := "updated_" + uuid.NewString()
	price := 10.0

	course, err := tc.Repo.Course.Update(context.Background(), models.UpdateCourse{
		ID: id, Title: &title, Price: &price,
	})

	require.NoError(t, err)
	require.Equal(t, title, course.Title)
	require.Equal(t, price, course.Price)
	require.Equal(t, models.DifficultyBeginner, *course.Difficulty)
}

func TestCourseRepo_Update_NotFound(t *testing.T) {
	tc := setupTest(t)

	title := "x"
	_, err := tc.Repo.Course.Update(context.Background(), models.UpdateCourse{ID: uuid.New(), Title: &title})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCourseRepo_UpdateStatus(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	require.NoError(t, tc.Repo.Course.UpdateStatus(ctx, models.UpdateCourseStatus{
		ID: id, Status: models.CourseStatusPublished,
	}))

	course, err := tc.Repo.Course.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, models.CourseStatusPublished, course.Status)
}

func TestCourseRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	require.NoError(t, tc.Repo.Course.Delete(ctx, id))

	_, err := tc.Repo.Course.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCourseRepo_DeleteLearningOutcomes(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	require.NoError(t, tc.Repo.Course.CreateLearningOutcome(ctx, models.CourseLearningOutcome{
		CourseID: id, Content: "Learn Go", Position: 1,
	}))
	require.NoError(t, tc.Repo.Course.DeleteLearningOutcomes(ctx, id))

	outcomes, err := tc.Repo.Course.GetLearningOutcomes(ctx, id)
	require.NoError(t, err)
	require.Empty(t, outcomes)
}

func TestCourseRepo_DeleteRequirements(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestCourse(t, tc, "course_"+uuid.NewString())

	require.NoError(t, tc.Repo.Course.CreateRequirement(ctx, models.CourseRequirement{
		CourseID: id, Content: "Basic programming", Position: 1,
	}))
	require.NoError(t, tc.Repo.Course.DeleteRequirements(ctx, id))

	requirements, err := tc.Repo.Course.GetRequirements(ctx, id)
	require.NoError(t, err)
	require.Empty(t, requirements)
}
