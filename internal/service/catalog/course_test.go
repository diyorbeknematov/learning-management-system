package catalog_test

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCourse_Create(t *testing.T) {
	e := testutil.Setup(t)

	instructor := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)
	difficulty := models.DifficultyBeginner

	course, err := e.Svc.Course.Create(context.Background(), testutil.ActorOf(instructor), models.CreateCourseRequest{
		CategoryID:       category.ID,
		Title:            "Go basics",
		Difficulty:       &difficulty,
		Price:            49.5,
		LearningOutcomes: []string{"Write Go", "Test Go"},
		Requirements:     []string{"A laptop"},
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	require.Equal(t, instructor.ID, course.InstructorID)
	require.Equal(t, models.CourseStatusDraft, course.Status)
	require.Equal(t, 49.5, course.Price)
	require.Equal(t, category.Name, course.CategoryName)

	require.Len(t, course.LearningOutcomes, 2)
	require.Equal(t, "Write Go", course.LearningOutcomes[0].Content)
	require.Equal(t, 1, course.LearningOutcomes[0].Position)
	require.Equal(t, 2, course.LearningOutcomes[1].Position)
	require.Len(t, course.Requirements, 1)

	require.Equal(t, instructor.ID, course.Instructor.ID)
	require.Equal(t, instructor.FirstName, course.Instructor.FirstName)
}

func TestCourse_Create_InstructorIDComesFromActor(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	other := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	course, err := e.Svc.Course.Create(context.Background(), testutil.ActorOf(owner), models.CreateCourseRequest{
		InstructorID: &other.ID,
		CategoryID:   category.ID,
		Title:        "Mine",
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	require.Equal(t, owner.ID, course.InstructorID)
}

func TestCourse_Create_SuperAdminForAnotherInstructor(t *testing.T) {
	e := testutil.Setup(t)

	instructor := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	course, err := e.Svc.Course.Create(context.Background(), testutil.AdminActor(), models.CreateCourseRequest{
		InstructorID: &instructor.ID,
		CategoryID:   category.ID,
		Title:        "For an instructor",
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	require.Equal(t, instructor.ID, course.InstructorID)
}

func TestCourse_Create_PayoutOnlyBySuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	instructor := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	payoutType := models.PayoutTypePercentage
	payoutValue := 30.0

	req := models.CreateCourseRequest{
		CategoryID:  category.ID,
		Title:       "Payout",
		PayoutType:  &payoutType,
		PayoutValue: &payoutValue,
	}

	byInstructor, err := e.Svc.Course.Create(ctx, testutil.ActorOf(instructor), req)
	require.NoError(t, err)
	e.CleanupCourse(t, byInstructor.ID)

	asAdmin, err := e.Svc.Course.GetByID(ctx, testutil.AdminActor(), byInstructor.ID)
	require.NoError(t, err)
	require.Nil(t, asAdmin.PayoutType, "the instructor's payout was ignored")

	req.InstructorID = &instructor.ID

	byAdmin, err := e.Svc.Course.Create(ctx, testutil.AdminActor(), req)
	require.NoError(t, err)
	e.CleanupCourse(t, byAdmin.ID)

	require.NotNil(t, byAdmin.PayoutType)
	require.Equal(t, 30.0, *byAdmin.PayoutValue)

	asOwner, err := e.Svc.Course.GetByID(ctx, testutil.ActorOf(instructor), byAdmin.ID)
	require.NoError(t, err)
	require.Nil(t, asOwner.PayoutType, "the payout is hidden from the instructor")
	require.Nil(t, asOwner.PayoutValue)
}

func TestCourse_Create_UnknownCategory(t *testing.T) {
	e := testutil.Setup(t)

	instructor := e.NewUser(t, models.RoleInstructor)

	_, err := e.Svc.Course.Create(context.Background(), testutil.ActorOf(instructor), models.CreateCourseRequest{
		CategoryID: uuid.New(),
		Title:      "No category",
	})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCourse_GetByID_Syllabus(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))

	e.AddLesson(t, course.ID)

	detail, err := e.Svc.Course.GetByID(ctx, testutil.ActorOf(instructor), course.ID)
	require.NoError(t, err)
	require.Equal(t, 1, detail.LessonCount)
	require.Len(t, detail.Modules, 1)
	require.Len(t, detail.Modules[0].Lessons, 1)
	require.Equal(t, "Lesson", detail.Modules[0].Lessons[0].Title)
}

func TestCourse_GetByID_RatingsAndCounts(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))
	e.Publish(t, instructor, course.ID)

	first := e.NewUser(t, models.RoleStudent)
	second := e.NewUser(t, models.RoleStudent)

	for _, student := range []*models.User{first, second} {
		_, err := e.Repo.Enrollment.Create(ctx, models.CreateEnrollment{CourseID: course.ID, StudentID: student.ID})
		require.NoError(t, err)
	}

	for student, rating := range map[*models.User]int{first: 4, second: 5} {
		_, err := e.Repo.Review.Create(ctx, models.CreateReview{StudentID: student.ID, CourseID: course.ID, Rating: rating})
		require.NoError(t, err)
	}

	detail, err := e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	require.NoError(t, err)
	require.Equal(t, 4.5, detail.AvgRating)
	require.Equal(t, 2, detail.ReviewCount)
	require.Equal(t, 2, detail.EnrollmentCount)
	require.Equal(t, 4.5, detail.Instructor.AvgRating)
}

func TestCourse_GetByID_DraftIsHidden(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	_, err := e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = e.Svc.Course.GetByID(ctx, testutil.ActorOf(stranger), course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = e.Svc.Course.GetByID(ctx, testutil.ActorOf(owner), course.ID)
	require.NoError(t, err)

	_, err = e.Svc.Course.GetByID(ctx, testutil.AdminActor(), course.ID)
	require.NoError(t, err)
}

func TestCourse_GetByID_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Course.GetByID(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCourse_GetList_OnlyPublishedForThePublic(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	published := e.NewCourse(t, owner, category)
	e.Publish(t, owner, published.ID)
	draft := e.NewCourse(t, owner, category)

	filter := models.CourseFilter{CategoryID: &category.ID}

	list, err := e.Svc.Course.GetList(ctx, models.Actor{}, filter)
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Equal(t, published.ID, list.Items[0].ID)

	list, err = e.Svc.Course.GetList(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)

	ownFilter := models.CourseFilter{CategoryID: &category.ID, InstructorID: &owner.ID}

	list, err = e.Svc.Course.GetList(ctx, testutil.ActorOf(owner), ownFilter)
	require.NoError(t, err)
	require.Equal(t, 2, list.Total, "the owner sees their drafts, e.g. %s", draft.ID)

	stranger := e.NewUser(t, models.RoleInstructor)

	list, err = e.Svc.Course.GetList(ctx, testutil.ActorOf(stranger), ownFilter)
	require.NoError(t, err)
	require.Equal(t, 1, list.Total, "another instructor sees only the published course")
}

func TestCourse_GetList_StatusFilterCannotShowDrafts(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)
	e.NewCourse(t, owner, category)

	draft := models.CourseStatusDraft

	list, err := e.Svc.Course.GetList(context.Background(), models.Actor{}, models.CourseFilter{
		CategoryID: &category.ID,
		Status:     &draft,
	})
	require.NoError(t, err)
	require.Zero(t, list.Total)
}

func TestCourse_GetList_HidesPayout(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	payoutType := models.PayoutTypeFixed
	payoutValue := 5.0

	course, err := e.Svc.Course.Create(ctx, testutil.AdminActor(), models.CreateCourseRequest{
		InstructorID: &owner.ID,
		CategoryID:   category.ID,
		Title:        "With payout",
		PayoutType:   &payoutType,
		PayoutValue:  &payoutValue,
	})
	require.NoError(t, err)
	e.CleanupCourse(t, course.ID)
	e.Publish(t, owner, course.ID)

	filter := models.CourseFilter{CategoryID: &category.ID}

	list, err := e.Svc.Course.GetList(ctx, models.Actor{}, filter)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Nil(t, list.Items[0].PayoutType)

	list, err = e.Svc.Course.GetList(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.NotNil(t, list.Items[0].PayoutType)
}

func TestCourse_Update(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	title := "Renamed"
	price := 99.0

	updated, err := e.Svc.Course.Update(context.Background(), testutil.ActorOf(owner), course.ID, models.UpdateCourseRequest{
		Title: &title,
		Price: &price,
	})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Title)
	require.Equal(t, 99.0, updated.Price)
}

func TestCourse_Update_ReplacesLists(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	course, err := e.Svc.Course.Create(ctx, testutil.ActorOf(owner), models.CreateCourseRequest{
		CategoryID:       category.ID,
		Title:            "Lists",
		LearningOutcomes: []string{"one", "two", "three"},
		Requirements:     []string{"first"},
	})
	require.NoError(t, err)
	e.CleanupCourse(t, course.ID)

	outcomes := []string{"only"}

	updated, err := e.Svc.Course.Update(ctx, testutil.ActorOf(owner), course.ID, models.UpdateCourseRequest{LearningOutcomes: &outcomes})
	require.NoError(t, err)
	require.Len(t, updated.LearningOutcomes, 1)
	require.Equal(t, "only", updated.LearningOutcomes[0].Content)
	require.Len(t, updated.Requirements, 1, "an unsent list stays as it was")

	none := []string{}

	updated, err = e.Svc.Course.Update(ctx, testutil.ActorOf(owner), course.ID, models.UpdateCourseRequest{Requirements: &none})
	require.NoError(t, err)
	require.Empty(t, updated.Requirements)
	require.Len(t, updated.LearningOutcomes, 1)
}

func TestCourse_Update_PayoutOnlyBySuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	payoutType := models.PayoutTypePercentage
	payoutValue := 40.0
	req := models.UpdateCourseRequest{PayoutType: &payoutType, PayoutValue: &payoutValue}

	_, err := e.Svc.Course.Update(ctx, testutil.ActorOf(owner), course.ID, req)
	require.NoError(t, err)

	asAdmin, err := e.Svc.Course.GetByID(ctx, testutil.AdminActor(), course.ID)
	require.NoError(t, err)
	require.Nil(t, asAdmin.PayoutType, "the instructor cannot set the payout")

	updated, err := e.Svc.Course.Update(ctx, testutil.AdminActor(), course.ID, req)
	require.NoError(t, err)
	require.NotNil(t, updated.PayoutType)
	require.Equal(t, 40.0, *updated.PayoutValue)
}

func TestCourse_Update_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	title := "Hijacked"

	_, err := e.Svc.Course.Update(context.Background(), testutil.ActorOf(stranger), course.ID, models.UpdateCourseRequest{Title: &title})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestCourse_Update_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	title := "x"

	_, err := e.Svc.Course.Update(context.Background(), testutil.AdminActor(), uuid.New(), models.UpdateCourseRequest{Title: &title})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCourse_UpdateStatus_Publish(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	e.Publish(t, owner, course.ID)

	detail, err := e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	require.NoError(t, err)
	require.Equal(t, models.CourseStatusPublished, detail.Status)

	err = e.Svc.Course.UpdateStatus(ctx, testutil.ActorOf(owner), models.UpdateCourseStatus{ID: course.ID, Status: models.CourseStatusDraft})
	require.NoError(t, err)

	_, err = e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCourse_UpdateStatus_EmptyCourseCannotBePublished(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	err := e.Svc.Course.UpdateStatus(context.Background(), testutil.ActorOf(owner), models.UpdateCourseStatus{
		ID:     course.ID,
		Status: models.CourseStatusPublished,
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestCourse_UpdateStatus_ModuleWithoutLessonsIsNotEnough(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	_, err := e.Repo.Module.Create(context.Background(), models.CreateModule{CourseID: course.ID, Title: "Empty", OrderNumber: 1})
	require.NoError(t, err)

	err = e.Svc.Course.UpdateStatus(context.Background(), testutil.ActorOf(owner), models.UpdateCourseStatus{
		ID:     course.ID,
		Status: models.CourseStatusPublished,
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestCourse_UpdateStatus_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	e.AddLesson(t, course.ID)

	err := e.Svc.Course.UpdateStatus(context.Background(), testutil.ActorOf(stranger), models.UpdateCourseStatus{
		ID:     course.ID,
		Status: models.CourseStatusPublished,
	})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestCourse_Delete(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	require.NoError(t, e.Svc.Course.Delete(ctx, testutil.ActorOf(owner), course.ID))

	_, err := e.Svc.Course.GetByID(ctx, testutil.AdminActor(), course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCourse_Delete_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	err := e.Svc.Course.Delete(context.Background(), testutil.ActorOf(stranger), course.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = e.Svc.Course.GetByID(context.Background(), testutil.ActorOf(owner), course.ID)
	require.NoError(t, err)
}

func TestCourse_Delete_SuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	require.NoError(t, e.Svc.Course.Delete(context.Background(), testutil.AdminActor(), course.ID))
}

func uploadCover(e *testutil.Env) string {
	key := "covers/" + testutil.Uniq() + ".png"
	e.Storage.Upload(key, 1024, "image/png")

	return key
}

func TestCourse_Cover_HasALink(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)
	cover := uploadCover(e)

	course, err := e.Svc.Course.Create(ctx, testutil.ActorOf(owner), models.CreateCourseRequest{
		CategoryID: category.ID,
		Title:      "With a cover",
		Cover:      &cover,
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	require.Equal(t, cover, *course.Cover)
	require.Equal(t, "http://files.test/"+cover, *course.CoverURL)

	e.Publish(t, owner, course.ID)

	list, err := e.Svc.Course.GetList(ctx, models.Actor{}, models.CourseFilter{CategoryID: &category.ID})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, "http://files.test/"+cover, *list.Items[0].CoverURL)
}

func TestCourse_Cover_WithoutOneThereIsNoLink(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	require.Nil(t, course.Cover)
	require.Nil(t, course.CoverURL)
}

func TestCourse_Cover_MustBeUploadedAndAnImage(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	missing := "covers/never-uploaded.png"
	notImage := "covers/" + testutil.Uniq() + ".pdf"
	e.Storage.Upload(notImage, 1024, "application/pdf")
	wrongFolder := "avatars/" + testutil.Uniq() + ".png"
	e.Storage.Upload(wrongFolder, 1024, "image/png")

	for _, cover := range []string{missing, notImage, wrongFolder} {
		cover := cover

		_, err := e.Svc.Course.Create(ctx, testutil.ActorOf(owner), models.CreateCourseRequest{CategoryID: category.ID, Title: "x", Cover: &cover})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestCourse_Cover_TooLarge(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	cover := "covers/" + testutil.Uniq() + ".png"
	e.Storage.Upload(cover, 11<<20, "image/png")

	_, err := e.Svc.Course.Create(context.Background(), testutil.ActorOf(owner), models.CreateCourseRequest{CategoryID: category.ID, Title: "x", Cover: &cover})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	require.Contains(t, e.Storage.Deleted, cover)
}

func TestCourse_Cover_ReplacingRemovesTheOldFile(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)
	first, second := uploadCover(e), uploadCover(e)

	course, err := e.Svc.Course.Create(ctx, testutil.ActorOf(owner), models.CreateCourseRequest{CategoryID: category.ID, Title: "x", Cover: &first})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	updated, err := e.Svc.Course.Update(ctx, testutil.ActorOf(owner), course.ID, models.UpdateCourseRequest{Cover: &second})
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+second, *updated.CoverURL)
	require.Equal(t, []string{first}, e.Storage.Deleted)

	// a failed update keeps the cover
	missing := "covers/missing.png"

	_, err = e.Svc.Course.Update(ctx, testutil.ActorOf(owner), course.ID, models.UpdateCourseRequest{Cover: &missing})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	require.Equal(t, []string{first}, e.Storage.Deleted)
}

func TestCourse_Instructor_HasAnAvatarLink(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)

	avatar := "avatars/" + testutil.Uniq() + ".png"
	e.Storage.Upload(avatar, 1024, "image/png")

	_, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(owner), models.UpdateMeRequest{Avatar: &avatar})
	require.NoError(t, err)

	course := e.NewCourse(t, owner, e.NewCategory(t))

	detail, err := e.Svc.Course.GetByID(ctx, testutil.ActorOf(owner), course.ID)
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+avatar, *detail.Instructor.AvatarURL)
}

func TestCourse_Cover_MovesOutOfTheTemporaryFolder(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	name := testutil.Uniq()
	temporary := "tmp/covers/" + name + ".png"
	e.Storage.Upload(temporary, 1024, "image/png")

	course, err := e.Svc.Course.Create(context.Background(), testutil.ActorOf(owner), models.CreateCourseRequest{
		CategoryID: category.ID,
		Title:      "Temporary cover",
		Cover:      &temporary,
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	permanent := "covers/" + name + ".png"
	require.Equal(t, permanent, *course.Cover)
	require.Contains(t, e.Storage.Objects, permanent)
}

func TestCourse_Cover_TemporaryFileMustBeUploadedAndOfTheRightFolder(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	missing := "tmp/covers/never-uploaded.png"
	wrongFolder := "tmp/avatars/" + testutil.Uniq() + ".png"
	e.Storage.Upload(wrongFolder, 1024, "image/png")

	for _, cover := range []string{missing, wrongFolder} {
		cover := cover

		_, err := e.Svc.Course.Create(context.Background(), testutil.ActorOf(owner), models.CreateCourseRequest{CategoryID: category.ID, Title: "x", Cover: &cover})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}
