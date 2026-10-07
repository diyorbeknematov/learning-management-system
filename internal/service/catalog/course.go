package catalog

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type Course struct {
	repo    *repo.Repository
	storage core.Storage
	cache   *core.Cache
}

func NewCourse(deps core.Dependencies) *Course {
	return &Course{
		repo:    deps.Repo,
		storage: deps.Storage,
		cache:   core.NewCache(deps.Redis),
	}
}

// Create makes a course owned by the actor. A SuperAdmin may create it for
// another instructor and is the only one who can set the payout.
func (s *Course) Create(
	ctx context.Context,
	actor models.Actor,
	req models.CreateCourseRequest,
) (*models.CourseDetail, error) {
	if req.Cover != nil {
		if _, err := core.CheckUpload(ctx, s.storage, models.UploadPurposeCourseCover, req.Cover, "CreateCourse"); err != nil {
			return nil, err
		}
	}

	instructorID := actor.UserID

	if actor.IsSuperAdmin() && req.InstructorID != nil {
		instructorID = *req.InstructorID
	}

	payoutType, payoutValue := req.PayoutType, req.PayoutValue
	if !actor.IsSuperAdmin() {
		payoutType, payoutValue = nil, nil
	}

	var id uuid.UUID

	err := s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		var err error

		id, err = tx.Course.Create(ctx, models.CreateCourse{
			InstructorID:  instructorID,
			CategoryID:    req.CategoryID,
			Title:         req.Title,
			Cover:         req.Cover,
			Description:   req.Description,
			Difficulty:    req.Difficulty,
			TotalDuration: req.TotalDuration,
			Language:      req.Language,
			Price:         req.Price,
			PayoutType:    payoutType,
			PayoutValue:   payoutValue,
		})
		if err != nil {
			return err
		}

		return saveCourseLists(ctx, tx, id, req.LearningOutcomes, req.Requirements)
	})
	if err != nil {
		return nil, err
	}

	s.cache.Invalidate(ctx, core.CacheCourses)

	return s.detail(ctx, actor, id)
}

// GetByID returns the course page: the course, its learning outcomes and
// requirements, the instructor and the syllabus. A draft is visible only to
// its owner and the SuperAdmin; everybody else gets "not found".
//
// The page of a published course is the same for everybody but the
// SuperAdmin (who also sees the payout), so it is cached.
func (s *Course) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.CourseDetail, error) {
	if actor.IsSuperAdmin() {
		return s.detail(ctx, actor, id)
	}

	return core.Cached(ctx, s.cache, core.CacheCourses, "detail:"+id.String(), func() (*models.CourseDetail, bool, error) {
		detail, err := s.detail(ctx, actor, id)
		if err != nil {
			return nil, false, err
		}

		return detail, detail.Status == models.CourseStatusPublished, nil
	})
}

// detail builds the course page from the database.
func (s *Course) detail(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.CourseDetail, error) {
	course, err := s.repo.Course.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if course.Status != models.CourseStatusPublished && core.CanManage(actor, course.InstructorID, "GetCourse") != nil {
		return nil, apperror.NotFound("service", "GetCourse", "course not found", apperror.ErrNotFound)
	}

	course.LearningOutcomes, err = s.repo.Course.GetLearningOutcomes(ctx, id)
	if err != nil {
		return nil, err
	}

	course.Requirements, err = s.repo.Course.GetRequirements(ctx, id)
	if err != nil {
		return nil, err
	}

	category, err := s.repo.Category.GetByID(ctx, course.CategoryID)
	if err != nil {
		return nil, err
	}

	instructor, err := s.repo.User.GetByID(ctx, course.InstructorID.String())
	if err != nil {
		return nil, err
	}

	instructorRating, err := s.repo.Review.GetInstructorAverageRating(ctx, course.InstructorID)
	if err != nil {
		return nil, err
	}

	rating, err := s.repo.Review.GetAverageRating(ctx, id)
	if err != nil {
		return nil, err
	}

	// only the totals are needed, so one row per page is enough
	_, reviewCount, err := s.repo.Review.GetListByCourseID(ctx, id, models.ReviewFilter{Limit: 1})
	if err != nil {
		return nil, err
	}

	_, enrollmentCount, err := s.repo.Enrollment.GetListByCourseID(ctx, id, models.EnrollmentFilter{Limit: 1})
	if err != nil {
		return nil, err
	}

	modules, lessonCount, err := s.syllabus(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &models.CourseDetail{
		CourseListItem: models.CourseListItem{
			Course:          *course,
			CategoryName:    category.Name,
			InstructorName:  instructor.FirstName + " " + instructor.LastName,
			AvgRating:       rating,
			ReviewCount:     reviewCount,
			LessonCount:     lessonCount,
			EnrollmentCount: enrollmentCount,
		},
		Instructor: models.CourseInstructor{
			ID:        instructor.ID,
			FirstName: instructor.FirstName,
			LastName:  instructor.LastName,
			Avatar:    instructor.Avatar,
			Bio:       instructor.Bio,
			AvgRating: instructorRating,
		},
		Modules: modules,
	}

	hidePayout(actor, &detail.Course)

	detail.CoverURL, err = core.DownloadURL(ctx, s.storage, detail.Cover)
	if err != nil {
		return nil, err
	}

	detail.Instructor.AvatarURL, err = core.DownloadURL(ctx, s.storage, detail.Instructor.Avatar)
	if err != nil {
		return nil, err
	}

	return detail, nil
}

// GetList is the course catalog. Only published courses are listed, except
// for a SuperAdmin and for an instructor who lists their own courses.
func (s *Course) GetList(
	ctx context.Context,
	actor models.Actor,
	filter models.CourseFilter,
) (*models.ListResponse[models.CourseListItem], error) {
	listsOwn := filter.InstructorID != nil && actor.UserID != uuid.Nil && *filter.InstructorID == actor.UserID

	if actor.IsSuperAdmin() || listsOwn {
		return s.list(ctx, actor, filter)
	}

	// everybody else sees the published courses, the same list for all of
	// them, so it is cached by the filter
	published := models.CourseStatusPublished
	filter.Status = &published

	return core.Cached(ctx, s.cache, core.CacheCourses, core.CacheKey("list", filter), func() (*models.ListResponse[models.CourseListItem], bool, error) {
		list, err := s.list(ctx, actor, filter)

		return list, err == nil, err
	})
}

// list reads a page of the catalog from the database.
func (s *Course) list(
	ctx context.Context,
	actor models.Actor,
	filter models.CourseFilter,
) (*models.ListResponse[models.CourseListItem], error) {
	courses, total, err := s.repo.Course.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}

	for i := range courses {
		hidePayout(actor, &courses[i].Course)

		courses[i].CoverURL, err = core.DownloadURL(ctx, s.storage, courses[i].Cover)
		if err != nil {
			return nil, err
		}
	}

	response := core.NewListResponse(courses, total, filter.Page, filter.Limit)

	return &response, nil
}

// Update changes a course. Only its owner or a SuperAdmin can, and only a
// SuperAdmin can change the payout.
func (s *Course) Update(
	ctx context.Context,
	actor models.Actor,
	id uuid.UUID,
	req models.UpdateCourseRequest,
) (*models.CourseDetail, error) {
	course, err := s.repo.Course.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, "UpdateCourse"); err != nil {
		return nil, err
	}

	if req.Cover != nil {
		if _, err := core.CheckUpload(ctx, s.storage, models.UploadPurposeCourseCover, req.Cover, "UpdateCourse"); err != nil {
			return nil, err
		}
	}

	payoutType, payoutValue := req.PayoutType, req.PayoutValue
	if !actor.IsSuperAdmin() {
		payoutType, payoutValue = nil, nil
	}

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		_, err := tx.Course.Update(ctx, models.UpdateCourse{
			ID:            id,
			CategoryID:    req.CategoryID,
			Title:         req.Title,
			Cover:         req.Cover,
			Description:   req.Description,
			Difficulty:    req.Difficulty,
			TotalDuration: req.TotalDuration,
			Language:      req.Language,
			Price:         req.Price,
			PayoutType:    payoutType,
			PayoutValue:   payoutValue,
		})
		if err != nil {
			return err
		}

		if req.LearningOutcomes != nil {
			if err := tx.Course.DeleteLearningOutcomes(ctx, id); err != nil {
				return err
			}

			if err := saveCourseLists(ctx, tx, id, *req.LearningOutcomes, nil); err != nil {
				return err
			}
		}

		if req.Requirements != nil {
			if err := tx.Course.DeleteRequirements(ctx, id); err != nil {
				return err
			}

			if err := saveCourseLists(ctx, tx, id, nil, *req.Requirements); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// the old cover is garbage now; failing to remove it is not worth failing the update
	if req.Cover != nil && course.Cover != nil && *course.Cover != *req.Cover {
		_ = s.storage.Delete(ctx, *course.Cover)
	}

	s.cache.Invalidate(ctx, core.CacheCourses)

	return s.detail(ctx, actor, id)
}

// UpdateStatus publishes a course or turns it back into a draft. An empty
// course cannot be published.
func (s *Course) UpdateStatus(ctx context.Context, actor models.Actor, req models.UpdateCourseStatus) error {
	course, err := s.repo.Course.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}

	if err := core.CanManage(actor, course.InstructorID, "UpdateCourseStatus"); err != nil {
		return err
	}

	if req.Status == models.CourseStatusPublished && course.Status != models.CourseStatusPublished {
		_, lessonCount, err := s.syllabus(ctx, req.ID)
		if err != nil {
			return err
		}

		if lessonCount == 0 {
			return apperror.InvalidInput(
				"service",
				"UpdateCourseStatus",
				"a course needs at least one module with a lesson before it is published",
				apperror.ErrInvalidInput,
			)
		}
	}

	if err := s.repo.Course.UpdateStatus(ctx, req); err != nil {
		return err
	}

	s.cache.Invalidate(ctx, core.CacheCourses)

	return nil
}

func (s *Course) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	course, err := s.repo.Course.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := core.CanManage(actor, course.InstructorID, "DeleteCourse"); err != nil {
		return err
	}

	if err := s.repo.Course.Delete(ctx, id); err != nil {
		return err
	}

	s.cache.Invalidate(ctx, core.CacheCourses)

	return nil
}

// syllabus returns the modules of a course with their lessons and the number
// of lessons.
func (s *Course) syllabus(ctx context.Context, courseID uuid.UUID) ([]models.ModuleDetail, int, error) {
	modules, err := s.repo.Module.GetListByCourseID(ctx, courseID)
	if err != nil {
		return nil, 0, err
	}

	details := make([]models.ModuleDetail, 0, len(modules))
	lessonCount := 0

	for _, module := range modules {
		lessons, err := s.repo.Lesson.GetListByModuleID(ctx, module.ID)
		if err != nil {
			return nil, 0, err
		}

		lessonCount += len(lessons)
		details = append(details, models.ModuleDetail{Module: module, Lessons: lessons})
	}

	return details, lessonCount, nil
}

// saveCourseLists writes learning outcomes and requirements, numbering them in
// the order they were sent.
func saveCourseLists(ctx context.Context, r *repo.Repository, courseID uuid.UUID, outcomes, requirements []string) error {
	for i, content := range outcomes {
		err := r.Course.CreateLearningOutcome(ctx, models.CourseLearningOutcome{
			CourseID: courseID,
			Content:  content,
			Position: i + 1,
		})
		if err != nil {
			return err
		}
	}

	for i, content := range requirements {
		err := r.Course.CreateRequirement(ctx, models.CourseRequirement{
			CourseID: courseID,
			Content:  content,
			Position: i + 1,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// hidePayout removes the payout from what the instructor and the students
// see: only a SuperAdmin handles the finances.
func hidePayout(actor models.Actor, course *models.Course) {
	if !actor.IsSuperAdmin() {
		course.PayoutType = nil
		course.PayoutValue = nil
	}
}
