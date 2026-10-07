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

// newModule adds a module with a title to the course as its owner.
func newModule(t *testing.T, e *testutil.Env, owner *models.User, courseID uuid.UUID, title string, order *int) *models.Module {
	t.Helper()

	module, err := e.Svc.Module.Create(context.Background(), testutil.ActorOf(owner), courseID, models.CreateModuleRequest{
		Title:       title,
		OrderNumber: order,
	})
	require.NoError(t, err)

	return module
}

// moduleTitles returns the titles of the course modules in their order.
func moduleTitles(t *testing.T, e *testutil.Env, owner *models.User, courseID uuid.UUID) []string {
	t.Helper()

	modules, err := e.Svc.Module.GetListByCourse(context.Background(), testutil.ActorOf(owner), courseID)
	require.NoError(t, err)

	titles := make([]string, len(modules))

	for i, module := range modules {
		require.Equal(t, i+1, module.OrderNumber, "the numbers must be 1..n without gaps")
		titles[i] = module.Title
	}

	return titles
}

func intPtr(n int) *int {
	return &n
}

func TestModule_Create_AppendsToTheEnd(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	first := newModule(t, e, owner, course.ID, "A", nil)
	second := newModule(t, e, owner, course.ID, "B", nil)

	require.Equal(t, 1, first.OrderNumber)
	require.Equal(t, 2, second.OrderNumber)
	require.Equal(t, course.ID, second.CourseID)
}

func TestModule_Create_InsertsInTheMiddle(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	newModule(t, e, owner, course.ID, "A", nil)
	newModule(t, e, owner, course.ID, "B", nil)
	inserted := newModule(t, e, owner, course.ID, "X", intPtr(2))

	require.Equal(t, 2, inserted.OrderNumber)
	require.Equal(t, []string{"A", "X", "B"}, moduleTitles(t, e, owner, course.ID))
}

func TestModule_Create_PositionPastTheEndGoesLast(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	newModule(t, e, owner, course.ID, "A", nil)
	last := newModule(t, e, owner, course.ID, "Z", intPtr(50))

	require.Equal(t, 2, last.OrderNumber)
}

func TestModule_Create_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	_, err := e.Svc.Module.Create(context.Background(), testutil.ActorOf(stranger), course.ID, models.CreateModuleRequest{Title: "x"})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestModule_Create_SuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	_, err := e.Svc.Module.Create(context.Background(), testutil.AdminActor(), course.ID, models.CreateModuleRequest{Title: "x"})
	require.NoError(t, err)
}

func TestModule_Create_UnknownCourse(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Module.Create(context.Background(), testutil.AdminActor(), uuid.New(), models.CreateModuleRequest{Title: "x"})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestModule_GetByID(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	_, err := e.Svc.Lesson.Create(ctx, testutil.ActorOf(owner), module.ID, models.CreateLessonRequest{Title: "L1"})
	require.NoError(t, err)

	found, err := e.Svc.Module.GetByID(ctx, testutil.ActorOf(owner), module.ID)
	require.NoError(t, err)
	require.Equal(t, module.ID, found.ID)
	require.Len(t, found.Lessons, 1)
	require.Equal(t, "L1", found.Lessons[0].Title)
}

func TestModule_GetByID_DraftCourseIsHidden(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	_, err := e.Svc.Module.GetByID(ctx, models.Actor{}, module.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	e.Publish(t, owner, course.ID)

	_, err = e.Svc.Module.GetByID(ctx, models.Actor{}, module.ID)
	require.NoError(t, err)
}

func TestModule_GetByID_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Module.GetByID(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestModule_GetListByCourse(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	newModule(t, e, owner, course.ID, "A", nil)
	second := newModule(t, e, owner, course.ID, "B", nil)

	_, err := e.Svc.Lesson.Create(ctx, testutil.ActorOf(owner), second.ID, models.CreateLessonRequest{Title: "L1"})
	require.NoError(t, err)

	modules, err := e.Svc.Module.GetListByCourse(ctx, testutil.ActorOf(owner), course.ID)
	require.NoError(t, err)
	require.Len(t, modules, 2)
	require.Empty(t, modules[0].Lessons)
	require.Len(t, modules[1].Lessons, 1)

	_, err = e.Svc.Module.GetListByCourse(ctx, models.Actor{}, course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestModule_Update(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	title := "Renamed"

	updated, err := e.Svc.Module.Update(context.Background(), testutil.ActorOf(owner), module.ID, models.UpdateModule{Title: &title})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Title)
	require.Equal(t, 1, updated.OrderNumber)
}

func TestModule_Update_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	title := "Hijacked"

	_, err := e.Svc.Module.Update(context.Background(), testutil.ActorOf(stranger), module.ID, models.UpdateModule{Title: &title})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestModule_UpdateOrder_MoveUp(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	newModule(t, e, owner, course.ID, "A", nil)
	newModule(t, e, owner, course.ID, "B", nil)
	third := newModule(t, e, owner, course.ID, "C", nil)

	require.NoError(t, e.Svc.Module.UpdateOrder(context.Background(), testutil.ActorOf(owner), third.ID, 1))
	require.Equal(t, []string{"C", "A", "B"}, moduleTitles(t, e, owner, course.ID))
}

func TestModule_UpdateOrder_MoveDown(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	first := newModule(t, e, owner, course.ID, "A", nil)
	newModule(t, e, owner, course.ID, "B", nil)
	newModule(t, e, owner, course.ID, "C", nil)

	require.NoError(t, e.Svc.Module.UpdateOrder(context.Background(), testutil.ActorOf(owner), first.ID, 3))
	require.Equal(t, []string{"B", "C", "A"}, moduleTitles(t, e, owner, course.ID))
}

func TestModule_UpdateOrder_SwapNeighbours(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	first := newModule(t, e, owner, course.ID, "A", nil)
	newModule(t, e, owner, course.ID, "B", nil)

	require.NoError(t, e.Svc.Module.UpdateOrder(context.Background(), testutil.ActorOf(owner), first.ID, 2))
	require.Equal(t, []string{"B", "A"}, moduleTitles(t, e, owner, course.ID))
}

func TestModule_UpdateOrder_PastTheEndGoesLast(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	first := newModule(t, e, owner, course.ID, "A", nil)
	newModule(t, e, owner, course.ID, "B", nil)

	require.NoError(t, e.Svc.Module.UpdateOrder(context.Background(), testutil.ActorOf(owner), first.ID, 99))
	require.Equal(t, []string{"B", "A"}, moduleTitles(t, e, owner, course.ID))
}

func TestModule_UpdateOrder_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	err := e.Svc.Module.UpdateOrder(context.Background(), testutil.ActorOf(stranger), module.ID, 1)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestModule_Delete_ClosesTheGap(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	newModule(t, e, owner, course.ID, "A", nil)
	middle := newModule(t, e, owner, course.ID, "B", nil)
	newModule(t, e, owner, course.ID, "C", nil)

	require.NoError(t, e.Svc.Module.Delete(ctx, testutil.ActorOf(owner), middle.ID))

	require.Equal(t, []string{"A", "C"}, moduleTitles(t, e, owner, course.ID))

	_, err := e.Svc.Module.GetByID(ctx, testutil.ActorOf(owner), middle.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestModule_Delete_RemovesItsLessons(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	lesson, err := e.Svc.Lesson.Create(ctx, testutil.ActorOf(owner), module.ID, models.CreateLessonRequest{Title: "L1"})
	require.NoError(t, err)

	require.NoError(t, e.Svc.Module.Delete(ctx, testutil.ActorOf(owner), module.ID))

	_, err = e.Svc.Lesson.GetByID(ctx, testutil.ActorOf(owner), lesson.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestModule_Delete_NotTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	owner := e.NewUser(t, models.RoleInstructor)
	stranger := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	module := newModule(t, e, owner, course.ID, "A", nil)

	err := e.Svc.Module.Delete(context.Background(), testutil.ActorOf(stranger), module.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}
