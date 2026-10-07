package catalog_test

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/stretchr/testify/require"
)

// In these tests a change is first made straight in the database, where the
// service cannot see it: a page that still shows the old value came from the
// cache. Then the change is made through the service, which must refresh it.

func TestCache_Category_ReadsAreCachedUntilTheCategoryChanges(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	category := e.NewCategory(t)

	got, err := e.Svc.Category.GetByID(ctx, category.ID)
	require.NoError(t, err)
	require.Equal(t, category.Name, got.Name)

	e.Exec(t, `UPDATE categories SET name = 'changed in the database' WHERE id = $1`, category.ID)

	got, err = e.Svc.Category.GetByID(ctx, category.ID)
	require.NoError(t, err)
	require.Equal(t, category.Name, got.Name, "the cached category")

	name := "changed by the service"

	_, err = e.Svc.Category.Update(ctx, models.UpdateCategory{ID: category.ID, Name: &name})
	require.NoError(t, err)

	got, err = e.Svc.Category.GetByID(ctx, category.ID)
	require.NoError(t, err)
	require.Equal(t, name, got.Name)
}

func TestCache_Category_ListIsRefreshedWhenOneIsCreated(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()

	// the list is filtered by a name of its own: other tests share the database
	// and create categories at the same time, so the total of all of them moves
	name := "cat-list-" + testutil.Uniq()
	filter := models.CategoryFilter{Name: &name, Limit: 100}

	before, err := e.Svc.Category.GetList(ctx, filter)
	require.NoError(t, err)
	require.Zero(t, before.Total)

	category, err := e.Svc.Category.Create(ctx, models.CreateCategory{Name: name})
	require.NoError(t, err)

	t.Cleanup(func() {
		e.Exec(t, `DELETE FROM categories WHERE id = $1`, category.ID)
	})

	after, err := e.Svc.Category.GetList(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, 1, after.Total)

	require.NoError(t, e.Svc.Category.Delete(ctx, category.ID))

	gone, err := e.Svc.Category.GetList(ctx, filter)
	require.NoError(t, err)
	require.Zero(t, gone.Total)
}

func TestCache_Course_PublicPageIsCachedAndRefreshedByTheOwner(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	e.Publish(t, owner, course.ID)

	visitor := models.Actor{}

	page, err := e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	require.Equal(t, course.Title, page.Title)

	e.Exec(t, `UPDATE courses SET title = 'changed in the database' WHERE id = $1`, course.ID)

	page, err = e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	require.Equal(t, course.Title, page.Title, "the cached page")

	title := "changed by the owner"

	updated, err := e.Svc.Course.Update(ctx, testutil.ActorOf(owner), course.ID, models.UpdateCourseRequest{Title: &title})
	require.NoError(t, err)
	require.Equal(t, title, updated.Title, "the owner gets the new page at once")

	page, err = e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	require.Equal(t, title, page.Title)
}

func TestCache_Course_ADraftIsNeverCached(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))

	_, err := e.Svc.Course.GetByID(ctx, testutil.ActorOf(owner), course.ID)
	require.NoError(t, err)

	e.Exec(t, `UPDATE courses SET title = 'changed in the database' WHERE id = $1`, course.ID)

	page, err := e.Svc.Course.GetByID(ctx, testutil.ActorOf(owner), course.ID)
	require.NoError(t, err)
	require.Equal(t, "changed in the database", page.Title, "a draft is read from the database every time")

	_, err = e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	e.Publish(t, owner, course.ID)

	page, err = e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	require.NoError(t, err)
	require.Equal(t, "changed in the database", page.Title, "published, so visible")

	// back to a draft: the cached page must go
	require.NoError(t, e.Svc.Course.UpdateStatus(ctx, testutil.ActorOf(owner), models.UpdateCourseStatus{ID: course.ID, Status: models.CourseStatusDraft}))

	_, err = e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCache_Course_PayoutIsHiddenFromEverybodyButTheSuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	payoutType, payoutValue := models.PayoutTypePercentage, 40.0

	course, err := e.Svc.Course.Create(ctx, testutil.AdminActor(), models.CreateCourseRequest{
		CategoryID:   category.ID,
		InstructorID: &owner.ID,
		Title:        "With a payout " + testutil.Uniq(),
		Price:        50,
		PayoutType:   &payoutType,
		PayoutValue:  &payoutValue,
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)
	e.Publish(t, owner, course.ID)

	student := testutil.ActorOf(e.NewUser(t, models.RoleStudent))

	for _, actor := range []models.Actor{testutil.AdminActor(), student, testutil.AdminActor(), student, testutil.ActorOf(owner)} {
		page, err := e.Svc.Course.GetByID(ctx, actor, course.ID)
		require.NoError(t, err)

		if actor.IsSuperAdmin() {
			require.NotNil(t, page.PayoutType, "the SuperAdmin sees the payout")
			require.Equal(t, 40.0, *page.PayoutValue)
		} else {
			require.Nil(t, page.PayoutType, "nobody else does, cached or not")
			require.Nil(t, page.PayoutValue)
		}
	}
}

func TestCache_Course_ListOfVisitorsIsCachedButTheOwnListIsNot(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	e.Publish(t, owner, course.ID)

	visitors := models.CourseFilter{InstructorID: &owner.ID}
	mine := testutil.ActorOf(owner)

	list, err := e.Svc.Course.GetList(ctx, models.Actor{}, visitors)
	require.NoError(t, err)
	require.Equal(t, course.Title, list.Items[0].Title)

	e.Exec(t, `UPDATE courses SET title = 'changed in the database' WHERE id = $1`, course.ID)

	list, err = e.Svc.Course.GetList(ctx, models.Actor{}, visitors)
	require.NoError(t, err)
	require.Equal(t, course.Title, list.Items[0].Title, "the cached list")

	list, err = e.Svc.Course.GetList(ctx, mine, visitors)
	require.NoError(t, err)
	require.Equal(t, "changed in the database", list.Items[0].Title, "the owner's list comes from the database")

	title := "changed by the owner"

	_, err = e.Svc.Course.Update(ctx, mine, course.ID, models.UpdateCourseRequest{Title: &title})
	require.NoError(t, err)

	list, err = e.Svc.Course.GetList(ctx, models.Actor{}, visitors)
	require.NoError(t, err)
	require.Equal(t, title, list.Items[0].Title)
}

func TestCache_Course_SyllabusFollowsModulesAndLessons(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	e.Publish(t, owner, course.ID)

	visitor := models.Actor{}

	page, err := e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	modules := len(page.Modules)

	module, err := e.Svc.Module.Create(ctx, testutil.ActorOf(owner), course.ID, models.CreateModuleRequest{Title: "New module"})
	require.NoError(t, err)

	page, err = e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	require.Len(t, page.Modules, modules+1, "a new module shows at once")

	lessons := page.LessonCount

	_, err = e.Svc.Lesson.Create(ctx, testutil.ActorOf(owner), module.ID, models.CreateLessonRequest{Title: "New lesson"})
	require.NoError(t, err)

	page, err = e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	require.Equal(t, lessons+1, page.LessonCount, "a new lesson shows at once")

	require.NoError(t, e.Svc.Module.Delete(ctx, testutil.ActorOf(owner), module.ID))

	page, err = e.Svc.Course.GetByID(ctx, visitor, course.ID)
	require.NoError(t, err)
	require.Len(t, page.Modules, modules)
	require.Equal(t, lessons, page.LessonCount)
}

func TestCache_Course_DeletedCoursePageIsGone(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	owner := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, owner, e.NewCategory(t))
	e.Publish(t, owner, course.ID)

	_, err := e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	require.NoError(t, err)

	require.NoError(t, e.Svc.Course.Delete(ctx, testutil.ActorOf(owner), course.ID))

	_, err = e.Svc.Course.GetByID(ctx, models.Actor{}, course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}
