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

// lessonSetup is an instructor with a course that has one module.
type lessonSetup struct {
	e      *testutil.Env
	owner  *models.User
	course *models.CourseDetail
	module *models.Module
}

func newLessonSetup(t *testing.T) lessonSetup {
	t.Helper()

	e := testutil.Setup(t)
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	return lessonSetup{e: e, owner: owner, course: course, module: newModule(t, e, owner, course.ID, "Module", nil)}
}

func (s lessonSetup) lesson(t *testing.T, title string, order *int, preview bool) *models.Lesson {
	t.Helper()

	lesson, err := s.e.Svc.Lesson.Create(context.Background(), testutil.ActorOf(s.owner), s.module.ID, models.CreateLessonRequest{
		Title:       title,
		OrderNumber: order,
		IsPreview:   preview,
	})
	require.NoError(t, err)

	return lesson
}

func (s lessonSetup) titles(t *testing.T) []string {
	t.Helper()

	lessons, err := s.e.Svc.Lesson.GetListByModule(context.Background(), testutil.ActorOf(s.owner), s.module.ID)
	require.NoError(t, err)

	titles := make([]string, len(lessons))

	for i, lesson := range lessons {
		require.Equal(t, i+1, lesson.OrderNumber, "the numbers must be 1..n without gaps")
		titles[i] = lesson.Title
	}

	return titles
}

func (s lessonSetup) publish(t *testing.T) {
	t.Helper()

	err := s.e.Svc.Course.UpdateStatus(context.Background(), testutil.ActorOf(s.owner), models.UpdateCourseStatus{
		ID:     s.course.ID,
		Status: models.CourseStatusPublished,
	})
	require.NoError(t, err)
}

func TestLesson_Create_AppendsAndInserts(t *testing.T) {
	s := newLessonSetup(t)

	first := s.lesson(t, "A", nil, false)
	s.lesson(t, "B", nil, false)
	inserted := s.lesson(t, "X", intPtr(2), false)

	require.Equal(t, 1, first.OrderNumber)
	require.Equal(t, 2, inserted.OrderNumber)
	require.Equal(t, []string{"A", "X", "B"}, s.titles(t))
}

func TestLesson_Create_Fields(t *testing.T) {
	s := newLessonSetup(t)

	duration := 25

	lesson, err := s.e.Svc.Lesson.Create(context.Background(), testutil.ActorOf(s.owner), s.module.ID, models.CreateLessonRequest{
		Title:     "Intro",
		Duration:  &duration,
		IsPreview: true,
	})
	require.NoError(t, err)
	require.Equal(t, s.module.ID, lesson.ModuleID)
	require.Equal(t, 25, *lesson.Duration)
	require.True(t, lesson.IsPreview)
}

func TestLesson_Create_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	stranger := s.e.NewUser(t, models.RoleInstructor)

	_, err := s.e.Svc.Lesson.Create(context.Background(), testutil.ActorOf(stranger), s.module.ID, models.CreateLessonRequest{Title: "x"})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestLesson_Create_UnknownModule(t *testing.T) {
	s := newLessonSetup(t)

	_, err := s.e.Svc.Lesson.Create(context.Background(), testutil.AdminActor(), uuid.New(), models.CreateLessonRequest{Title: "x"})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestLesson_GetByID_And_List_DraftIsHidden(t *testing.T) {
	s := newLessonSetup(t)

	ctx := context.Background()
	lesson := s.lesson(t, "A", nil, false)

	_, err := s.e.Svc.Lesson.GetByID(ctx, models.Actor{}, lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = s.e.Svc.Lesson.GetListByModule(ctx, models.Actor{}, s.module.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	found, err := s.e.Svc.Lesson.GetByID(ctx, testutil.ActorOf(s.owner), lesson.ID)
	require.NoError(t, err)
	require.Equal(t, lesson.ID, found.ID)

	s.publish(t)

	found, err = s.e.Svc.Lesson.GetByID(ctx, models.Actor{}, lesson.ID)
	require.NoError(t, err)
	require.Equal(t, lesson.ID, found.ID)

	list, err := s.e.Svc.Lesson.GetListByModule(ctx, models.Actor{}, s.module.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestLesson_GetByID_NotFound(t *testing.T) {
	s := newLessonSetup(t)

	_, err := s.e.Svc.Lesson.GetByID(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestLesson_Update(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	title := "Renamed"
	preview := true

	updated, err := s.e.Svc.Lesson.Update(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.UpdateLesson{
		Title:     &title,
		IsPreview: &preview,
	})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Title)
	require.True(t, updated.IsPreview)
	require.Equal(t, 1, updated.OrderNumber)
}

func TestLesson_Update_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	stranger := s.e.NewUser(t, models.RoleInstructor)
	title := "Hijacked"

	_, err := s.e.Svc.Lesson.Update(context.Background(), testutil.ActorOf(stranger), lesson.ID, models.UpdateLesson{Title: &title})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestLesson_UpdateOrder(t *testing.T) {
	s := newLessonSetup(t)

	s.lesson(t, "A", nil, false)
	s.lesson(t, "B", nil, false)
	third := s.lesson(t, "C", nil, false)

	require.NoError(t, s.e.Svc.Lesson.UpdateOrder(context.Background(), testutil.ActorOf(s.owner), third.ID, 1))
	require.Equal(t, []string{"C", "A", "B"}, s.titles(t))
}

func TestLesson_UpdateOrder_OnlyInItsOwnModule(t *testing.T) {
	s := newLessonSetup(t)

	other := newModule(t, s.e, s.owner, s.course.ID, "Other", nil)

	_, err := s.e.Svc.Lesson.Create(context.Background(), testutil.ActorOf(s.owner), other.ID, models.CreateLessonRequest{Title: "O1"})
	require.NoError(t, err)

	s.lesson(t, "A", nil, false)
	second := s.lesson(t, "B", nil, false)

	require.NoError(t, s.e.Svc.Lesson.UpdateOrder(context.Background(), testutil.ActorOf(s.owner), second.ID, 1))
	require.Equal(t, []string{"B", "A"}, s.titles(t))

	moduleLessons, err := s.e.Svc.Lesson.GetListByModule(context.Background(), testutil.ActorOf(s.owner), other.ID)
	require.NoError(t, err)
	require.Len(t, moduleLessons, 1)
	require.Equal(t, 1, moduleLessons[0].OrderNumber)
}

func TestLesson_UpdateOrder_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	stranger := s.e.NewUser(t, models.RoleInstructor)

	err := s.e.Svc.Lesson.UpdateOrder(context.Background(), testutil.ActorOf(stranger), lesson.ID, 1)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestLesson_Delete_ClosesTheGap(t *testing.T) {
	s := newLessonSetup(t)

	ctx := context.Background()

	s.lesson(t, "A", nil, false)
	middle := s.lesson(t, "B", nil, false)
	s.lesson(t, "C", nil, false)

	require.NoError(t, s.e.Svc.Lesson.Delete(ctx, testutil.ActorOf(s.owner), middle.ID))
	require.Equal(t, []string{"A", "C"}, s.titles(t))

	_, err := s.e.Svc.Lesson.GetByID(ctx, testutil.ActorOf(s.owner), middle.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestLesson_Delete_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	stranger := s.e.NewUser(t, models.RoleInstructor)

	err := s.e.Svc.Lesson.Delete(context.Background(), testutil.ActorOf(stranger), lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func strPtr(s string) *string {
	return &s
}

func (s lessonSetup) text(t *testing.T, lessonID uuid.UUID, content string) *models.LessonMaterial {
	t.Helper()

	material, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lessonID, models.CreateLessonMaterial{
		Type:    models.MaterialTypeText,
		Content: strPtr(content),
	})
	require.NoError(t, err)

	return material
}

func (s lessonSetup) file(t *testing.T, lessonID uuid.UUID, key string) *models.LessonMaterial {
	t.Helper()

	s.e.Storage.Upload(key, 2048, "application/pdf")

	material, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lessonID, models.CreateLessonMaterial{
		Type:      models.MaterialTypeFile,
		ObjectKey: strPtr(key),
		FileName:  strPtr("slides.pdf"),
	})
	require.NoError(t, err)

	return material
}

func TestMaterial_Create_Text(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "Read this")

	require.Equal(t, models.MaterialTypeText, material.Type)
	require.Equal(t, "Read this", *material.Content)
	require.Equal(t, lesson.ID, material.LessonID)
}

func TestMaterial_Create_TextNeedsContent(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)

	for _, content := range []*string{nil, strPtr(""), strPtr("   ")} {
		_, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
			Type:    models.MaterialTypeText,
			Content: content,
		})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestMaterial_Create_Video(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)

	material, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
		Type:    models.MaterialTypeVideo,
		Content: strPtr("https://videos.example.com/intro.mp4"),
	})
	require.NoError(t, err)
	require.Equal(t, models.MaterialTypeVideo, material.Type)
}

func TestMaterial_Create_VideoMustBeALink(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)

	for _, content := range []string{"not a link", "ftp://example.com/video", "javascript:alert(1)", "/relative/path"} {
		_, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
			Type:    models.MaterialTypeVideo,
			Content: strPtr(content),
		})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestMaterial_Create_File(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	key := "materials/" + uuid.NewString() + ".pdf"

	material := s.file(t, lesson.ID, key)

	require.Equal(t, models.MaterialTypeFile, material.Type)
	require.Equal(t, key, *material.ObjectKey)
	require.Equal(t, int64(2048), *material.FileSize, "the size comes from the storage")
	require.Equal(t, "application/pdf", *material.MimeType)
	require.Equal(t, "http://files.test/"+key, *material.FileURL)
}

func TestMaterial_Create_FileMustBeUploaded(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)

	_, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
		Type:      models.MaterialTypeFile,
		ObjectKey: strPtr("materials/never-uploaded.pdf"),
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestMaterial_Create_FileKeyMustBeInMaterialsFolder(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	s.e.Storage.Upload("avatars/someone.png", 10, "image/png")

	for _, key := range []*string{nil, strPtr("avatars/someone.png"), strPtr("../materials/x.pdf")} {
		_, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
			Type:      models.MaterialTypeFile,
			ObjectKey: key,
		})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestMaterial_Create_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	stranger := s.e.NewUser(t, models.RoleInstructor)

	_, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(stranger), lesson.ID, models.CreateLessonMaterial{
		Type:    models.MaterialTypeText,
		Content: strPtr("x"),
	})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestMaterial_Access_OwnerAndSuperAdminAlwaysOpen(t *testing.T) {
	s := newLessonSetup(t)

	ctx := context.Background()
	lesson := s.lesson(t, "A", nil, false)
	s.text(t, lesson.ID, "secret")

	for _, actor := range []models.Actor{testutil.ActorOf(s.owner), testutil.AdminActor()} {
		materials, err := s.e.Svc.Lesson.GetMaterials(ctx, actor, lesson.ID)
		require.NoError(t, err)
		require.Len(t, materials, 1)
	}
}

func TestMaterial_Access_PreviewIsOpenToEverybody(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, true)
	s.text(t, lesson.ID, "free sample")
	s.publish(t)

	materials, err := s.e.Svc.Lesson.GetMaterials(context.Background(), models.Actor{}, lesson.ID)
	require.NoError(t, err)
	require.Len(t, materials, 1)
}

func TestMaterial_Access_PaidLessonNeedsEnrollment(t *testing.T) {
	s := newLessonSetup(t)

	ctx := context.Background()
	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "paid content")
	s.publish(t)

	student := s.e.NewUser(t, models.RoleStudent)

	_, err := s.e.Svc.Lesson.GetMaterials(ctx, models.Actor{}, lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = s.e.Svc.Lesson.GetMaterials(ctx, testutil.ActorOf(student), lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = s.e.Svc.Lesson.GetMaterial(ctx, testutil.ActorOf(student), material.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	enrollmentID, err := s.e.Repo.Enrollment.Create(ctx, models.CreateEnrollment{CourseID: s.course.ID, StudentID: student.ID})
	require.NoError(t, err)

	materials, err := s.e.Svc.Lesson.GetMaterials(ctx, testutil.ActorOf(student), lesson.ID)
	require.NoError(t, err)
	require.Len(t, materials, 1)

	opened, err := s.e.Svc.Lesson.GetMaterial(ctx, testutil.ActorOf(student), material.ID)
	require.NoError(t, err)
	require.Equal(t, "paid content", *opened.Content)

	err = s.e.Repo.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{ID: enrollmentID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	_, err = s.e.Svc.Lesson.GetMaterials(ctx, testutil.ActorOf(student), lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestMaterial_Access_DraftCourseIsHidden(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, true)
	s.text(t, lesson.ID, "not published yet")

	_, err := s.e.Svc.Lesson.GetMaterials(context.Background(), models.Actor{}, lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestMaterial_GetMaterials_FileHasLink(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	key := "materials/" + uuid.NewString() + ".pdf"
	s.file(t, lesson.ID, key)

	materials, err := s.e.Svc.Lesson.GetMaterials(context.Background(), testutil.ActorOf(s.owner), lesson.ID)
	require.NoError(t, err)
	require.Len(t, materials, 1)
	require.Equal(t, "http://files.test/"+key, *materials[0].FileURL)
}

func TestMaterial_Update_Text(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "old")

	updated, err := s.e.Svc.Lesson.UpdateMaterial(context.Background(), testutil.ActorOf(s.owner), material.ID, models.UpdateLessonMaterial{
		Content: strPtr("new"),
	})
	require.NoError(t, err)
	require.Equal(t, "new", *updated.Content)
	require.Equal(t, models.MaterialTypeText, updated.Type)
}

func TestMaterial_Update_TextCannotBecomeEmpty(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "old")

	_, err := s.e.Svc.Lesson.UpdateMaterial(context.Background(), testutil.ActorOf(s.owner), material.ID, models.UpdateLessonMaterial{
		Content: strPtr(""),
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestMaterial_Update_TextIgnoresFileFields(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "old")

	updated, err := s.e.Svc.Lesson.UpdateMaterial(context.Background(), testutil.ActorOf(s.owner), material.ID, models.UpdateLessonMaterial{
		ObjectKey: strPtr("materials/sneaky.pdf"),
		FileName:  strPtr("sneaky.pdf"),
	})
	require.NoError(t, err)
	require.Nil(t, updated.ObjectKey)
	require.Nil(t, updated.FileName)
}

func TestMaterial_Update_ReplacingAFileRemovesTheOldOne(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	oldKey := "materials/" + uuid.NewString() + ".pdf"
	newKey := "materials/" + uuid.NewString() + ".pdf"
	material := s.file(t, lesson.ID, oldKey)

	s.e.Storage.Upload(newKey, 4096, "application/pdf")

	updated, err := s.e.Svc.Lesson.UpdateMaterial(context.Background(), testutil.ActorOf(s.owner), material.ID, models.UpdateLessonMaterial{
		ObjectKey: &newKey,
	})
	require.NoError(t, err)
	require.Equal(t, newKey, *updated.ObjectKey)
	require.Equal(t, int64(4096), *updated.FileSize)
	require.Equal(t, "http://files.test/"+newKey, *updated.FileURL)
	require.Contains(t, s.e.Storage.Deleted, oldKey)
	require.NotContains(t, s.e.Storage.Deleted, newKey)
}

func TestMaterial_Update_NewFileMustBeUploaded(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	oldKey := "materials/" + uuid.NewString() + ".pdf"
	material := s.file(t, lesson.ID, oldKey)

	_, err := s.e.Svc.Lesson.UpdateMaterial(context.Background(), testutil.ActorOf(s.owner), material.ID, models.UpdateLessonMaterial{
		ObjectKey: strPtr("materials/never-uploaded.pdf"),
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	require.Empty(t, s.e.Storage.Deleted, "the old file stays when the update fails")
}

func TestMaterial_Update_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "old")
	stranger := s.e.NewUser(t, models.RoleInstructor)

	_, err := s.e.Svc.Lesson.UpdateMaterial(context.Background(), testutil.ActorOf(stranger), material.ID, models.UpdateLessonMaterial{
		Content: strPtr("hijacked"),
	})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestMaterial_Delete_RemovesTheFile(t *testing.T) {
	s := newLessonSetup(t)

	ctx := context.Background()
	lesson := s.lesson(t, "A", nil, false)
	key := "materials/" + uuid.NewString() + ".pdf"
	material := s.file(t, lesson.ID, key)

	require.NoError(t, s.e.Svc.Lesson.DeleteMaterial(ctx, testutil.ActorOf(s.owner), material.ID))
	require.Contains(t, s.e.Storage.Deleted, key)

	_, err := s.e.Svc.Lesson.GetMaterial(ctx, testutil.ActorOf(s.owner), material.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestMaterial_Delete_Text(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "x")

	require.NoError(t, s.e.Svc.Lesson.DeleteMaterial(context.Background(), testutil.ActorOf(s.owner), material.ID))
	require.Empty(t, s.e.Storage.Deleted)
}

func TestMaterial_Delete_NotTheOwner(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	material := s.text(t, lesson.ID, "x")
	stranger := s.e.NewUser(t, models.RoleInstructor)

	err := s.e.Svc.Lesson.DeleteMaterial(context.Background(), testutil.ActorOf(stranger), material.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestMaterial_NotFound(t *testing.T) {
	s := newLessonSetup(t)

	_, err := s.e.Svc.Lesson.GetMaterial(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	testutil.RequireCode(t, s.e.Svc.Lesson.DeleteMaterial(context.Background(), testutil.AdminActor(), uuid.New()), apperror.CodeNotFound)
}

func TestMaterial_Create_UploadedVideo(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)

	for _, contentType := range []string{"video/mp4", "video/webm"} {
		key := "materials/" + uuid.NewString() + ".video"
		s.e.Storage.Upload(key, 300<<20, contentType)

		material, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
			Type:      models.MaterialTypeFile,
			ObjectKey: strPtr(key),
		})
		require.NoError(t, err, contentType)
		require.Equal(t, contentType, *material.MimeType, "the client plays it by its type")
		require.Equal(t, int64(300<<20), *material.FileSize)
		require.Equal(t, "http://files.test/"+key, *material.FileURL)
	}
}

func TestMaterial_Create_FileTypesAndSizes(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)

	cases := []struct {
		name        string
		contentType string
		size        int64
		allowed     bool
	}{
		{"pdf at the limit", "application/pdf", 100 << 20, true},
		{"pdf over the limit", "application/pdf", 101 << 20, false},
		{"presentation", "application/vnd.openxmlformats-officedocument.presentationml.presentation", 20 << 20, true},
		{"old presentation", "application/vnd.ms-powerpoint", 20 << 20, true},
		{"video at the limit", "video/mp4", 500 << 20, true},
		{"video over the limit", "video/mp4", 501 << 20, false},
		{"program", "application/x-msdownload", 1 << 20, false},
		{"html page", "text/html", 1 << 10, false},
		{"unknown type", "application/octet-stream", 1 << 10, false},
	}

	for _, tt := range cases {
		key := "materials/" + uuid.NewString()
		s.e.Storage.Upload(key, tt.size, tt.contentType)

		_, err := s.e.Svc.Lesson.CreateMaterial(context.Background(), testutil.ActorOf(s.owner), lesson.ID, models.CreateLessonMaterial{
			Type:      models.MaterialTypeFile,
			ObjectKey: strPtr(key),
		})

		if tt.allowed {
			require.NoError(t, err, tt.name)
			continue
		}

		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
		require.Contains(t, s.e.Storage.Deleted, key, "a rejected file is removed: "+tt.name)
	}
}

func TestMaterial_Create_FileMovesOutOfTheTemporaryFolder(t *testing.T) {
	s := newLessonSetup(t)

	lesson := s.lesson(t, "A", nil, false)
	name := uuid.NewString()

	material := s.file(t, lesson.ID, "tmp/materials/"+name+".pdf")

	permanent := "materials/" + name + ".pdf"
	require.Equal(t, permanent, *material.ObjectKey)
	require.Contains(t, s.e.Storage.Objects, permanent)
	require.Equal(t, "http://files.test/"+permanent, *material.FileURL)
}
