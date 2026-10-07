package catalog

import (
	"context"
	"net/url"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type Lesson struct {
	repo    *repo.Repository
	storage core.Storage
	cache   *core.Cache
}

func NewLesson(deps core.Dependencies) *Lesson {
	return &Lesson{
		repo:    deps.Repo,
		storage: deps.Storage,
		cache:   core.NewCache(deps.Redis),
	}
}

// saved forgets the cached course pages, whose syllabus has the lessons, after
// a change was written; a failed change leaves the cache alone.
func (s *Lesson) saved(ctx context.Context, err error) error {
	if err == nil {
		s.cache.Invalidate(ctx, core.CacheCourses)
	}

	return err
}

func lessonIDs(lessons []models.Lesson) []uuid.UUID {
	ids := make([]uuid.UUID, len(lessons))

	for i, lesson := range lessons {
		ids[i] = lesson.ID
	}

	return ids
}

func setLessonOrder(tx *repo.Repository) func(context.Context, uuid.UUID, int) error {
	return func(ctx context.Context, id uuid.UUID, number int) error {
		return tx.Lesson.UpdateOrder(ctx, models.UpdateLessonOrder{ID: id, OrderNumber: number})
	}
}

// lessonContext loads a lesson with the module and the course above it.
func (s *Lesson) lessonContext(ctx context.Context, lessonID uuid.UUID) (*models.Lesson, *models.Module, *models.Course, error) {
	lesson, err := s.repo.Lesson.GetByID(ctx, lessonID)
	if err != nil {
		return nil, nil, nil, err
	}

	module, err := s.repo.Module.GetByID(ctx, lesson.ModuleID)
	if err != nil {
		return nil, nil, nil, err
	}

	course, err := s.repo.Course.GetByID(ctx, module.CourseID)
	if err != nil {
		return nil, nil, nil, err
	}

	return lesson, module, course, nil
}

// Create adds a lesson to a module; the order works like in Module.Create.
func (s *Lesson) Create(
	ctx context.Context,
	actor models.Actor,
	moduleID uuid.UUID,
	req models.CreateLessonRequest,
) (*models.Lesson, error) {
	module, err := s.repo.Module.GetByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}

	if _, err := core.ManageableCourse(ctx, s.repo, actor, module.CourseID, "CreateLesson"); err != nil {
		return nil, err
	}

	var id uuid.UUID

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		existing, err := tx.Lesson.GetListByModuleID(ctx, moduleID)
		if err != nil {
			return err
		}

		position := len(existing) + 1
		if req.OrderNumber != nil && *req.OrderNumber < position {
			position = *req.OrderNumber
		}

		id, err = tx.Lesson.Create(ctx, models.CreateLesson{
			ModuleID:    moduleID,
			Title:       req.Title,
			Duration:    req.Duration,
			OrderNumber: core.TempOrderBase * 2,
			IsPreview:   req.IsPreview,
		})
		if err != nil {
			return err
		}

		return core.Renumber(ctx, core.MoveTo(lessonIDs(existing), id, position), setLessonOrder(tx))
	})
	if err := s.saved(ctx, err); err != nil {
		return nil, err
	}

	return s.repo.Lesson.GetByID(ctx, id)
}

// GetByID returns a lesson without its materials; the syllabus of a published
// course is open to everybody.
func (s *Lesson) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.Lesson, error) {
	lesson, _, course, err := s.lessonContext(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := courseVisible(actor, course, "GetLesson"); err != nil {
		return nil, err
	}

	return lesson, nil
}

func (s *Lesson) GetListByModule(ctx context.Context, actor models.Actor, moduleID uuid.UUID) ([]models.Lesson, error) {
	module, err := s.repo.Module.GetByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}

	if _, err := core.VisibleCourse(ctx, s.repo, actor, module.CourseID, "GetLessonList"); err != nil {
		return nil, err
	}

	return s.repo.Lesson.GetListByModuleID(ctx, moduleID)
}

func (s *Lesson) Update(
	ctx context.Context,
	actor models.Actor,
	id uuid.UUID,
	req models.UpdateLesson,
) (*models.Lesson, error) {
	_, _, course, err := s.lessonContext(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, "UpdateLesson"); err != nil {
		return nil, err
	}

	req.ID = id

	updated, err := s.repo.Lesson.Update(ctx, req)
	if err := s.saved(ctx, err); err != nil {
		return nil, err
	}

	return updated, nil
}

// UpdateOrder moves the lesson to the given place in its module; the lessons
// in between shift by one.
func (s *Lesson) UpdateOrder(ctx context.Context, actor models.Actor, id uuid.UUID, order int) error {
	lesson, _, course, err := s.lessonContext(ctx, id)
	if err != nil {
		return err
	}

	if err := core.CanManage(actor, course.InstructorID, "UpdateLessonOrder"); err != nil {
		return err
	}

	return s.saved(ctx, s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		lessons, err := tx.Lesson.GetListByModuleID(ctx, lesson.ModuleID)
		if err != nil {
			return err
		}

		return core.Renumber(ctx, core.MoveTo(lessonIDs(lessons), id, order), setLessonOrder(tx))
	}))
}

// Delete removes the lesson and closes the gap in the order.
func (s *Lesson) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	lesson, _, course, err := s.lessonContext(ctx, id)
	if err != nil {
		return err
	}

	if err := core.CanManage(actor, course.InstructorID, "DeleteLesson"); err != nil {
		return err
	}

	return s.saved(ctx, s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		if err := tx.Lesson.Delete(ctx, id); err != nil {
			return err
		}

		remaining, err := tx.Lesson.GetListByModuleID(ctx, lesson.ModuleID)
		if err != nil {
			return err
		}

		return core.Renumber(ctx, lessonIDs(remaining), setLessonOrder(tx))
	}))
}

// courseVisible is visibleCourse for a course that is already loaded.
func courseVisible(actor models.Actor, course *models.Course, op string) error {
	if course.Status != models.CourseStatusPublished && core.CanManage(actor, course.InstructorID, op) != nil {
		return apperror.NotFound("service", op, "not found", apperror.ErrNotFound)
	}

	return nil
}

// canOpen decides who can read the materials of a lesson: the owner and the
// SuperAdmin, anybody for a preview lesson, and a student who is enrolled.
func (s *Lesson) canOpen(ctx context.Context, actor models.Actor, lesson *models.Lesson, course *models.Course, op string) error {
	if err := courseVisible(actor, course, op); err != nil {
		return err
	}

	if core.CanManage(actor, course.InstructorID, op) == nil || lesson.IsPreview {
		return nil
	}

	if actor.UserID != uuid.Nil {
		enrollment, err := s.repo.Enrollment.GetByStudentAndCourse(ctx, actor.UserID, course.ID)
		if err == nil && enrollment.Status != models.EnrollmentStatusDropped {
			return nil
		}

		if err != nil && !core.IsNotFound(err) {
			return err
		}
	}

	return apperror.Forbidden("service", op, "enroll in the course to open this lesson", apperror.ErrForbidden)
}

// withFileURL adds a temporary download link to the material of a file.
func (s *Lesson) withFileURL(ctx context.Context, material *models.LessonMaterial) error {
	if material.Type != models.MaterialTypeFile || material.ObjectKey == nil {
		return nil
	}

	link, err := s.storage.PresignDownload(ctx, *material.ObjectKey)
	if err != nil {
		return err
	}

	material.FileURL = &link

	return nil
}

// GetMaterials returns the materials of a lesson to somebody who may open it.
func (s *Lesson) GetMaterials(ctx context.Context, actor models.Actor, lessonID uuid.UUID) ([]models.LessonMaterial, error) {
	lesson, _, course, err := s.lessonContext(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	if err := s.canOpen(ctx, actor, lesson, course, "GetMaterials"); err != nil {
		return nil, err
	}

	materials, err := s.repo.Lesson.GetMaterials(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	for i := range materials {
		if err := s.withFileURL(ctx, &materials[i]); err != nil {
			return nil, err
		}
	}

	return materials, nil
}

func (s *Lesson) GetMaterial(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.LessonMaterial, error) {
	material, err := s.repo.Lesson.GetMaterialByID(ctx, id)
	if err != nil {
		return nil, err
	}

	lesson, _, course, err := s.lessonContext(ctx, material.LessonID)
	if err != nil {
		return nil, err
	}

	if err := s.canOpen(ctx, actor, lesson, course, "GetMaterial"); err != nil {
		return nil, err
	}

	if err := s.withFileURL(ctx, material); err != nil {
		return nil, err
	}

	return material, nil
}

// validateContent checks the content of a text or a video material. A video
// is a link.
func validateContent(materialType models.MaterialType, content *string, op string) error {
	if content == nil || strings.TrimSpace(*content) == "" {
		return apperror.InvalidInput("service", op, "content is required", apperror.ErrInvalidInput)
	}

	if materialType == models.MaterialTypeVideo {
		link, err := url.ParseRequestURI(*content)
		if err != nil || (link.Scheme != "http" && link.Scheme != "https") || link.Host == "" {
			return apperror.InvalidInput("service", op, "a video must be an http or https link", apperror.ErrInvalidInput)
		}
	}

	return nil
}

// checkFile checks that the file was uploaded to the materials folder, is of an
// allowed type and not too large, and returns its real size and type.
func (s *Lesson) checkFile(ctx context.Context, objectKey *string, op string) (size int64, contentType string, err error) {
	info, err := core.CheckUpload(ctx, s.storage, models.UploadPurposeMaterial, objectKey, op)
	if err != nil {
		return 0, "", err
	}

	return info.Size, info.ContentType, nil
}

// CreateMaterial adds a text, a video link or a file to a lesson.
func (s *Lesson) CreateMaterial(
	ctx context.Context,
	actor models.Actor,
	lessonID uuid.UUID,
	req models.CreateLessonMaterial,
) (*models.LessonMaterial, error) {
	_, _, course, err := s.lessonContext(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, "CreateMaterial"); err != nil {
		return nil, err
	}

	req.LessonID = lessonID

	switch req.Type {
	case models.MaterialTypeFile:
		size, contentType, err := s.checkFile(ctx, req.ObjectKey, "CreateMaterial")
		if err != nil {
			return nil, err
		}

		req.Content = nil
		req.FileSize = &size

		if req.MimeType == nil && contentType != "" {
			req.MimeType = &contentType
		}
	default:
		if err := validateContent(req.Type, req.Content, "CreateMaterial"); err != nil {
			return nil, err
		}

		req.ObjectKey, req.FileName, req.MimeType, req.FileSize = nil, nil, nil, nil
	}

	id, err := s.repo.Lesson.CreateMaterial(ctx, req)
	if err != nil {
		return nil, err
	}

	material, err := s.repo.Lesson.GetMaterialByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.withFileURL(ctx, material); err != nil {
		return nil, err
	}

	return material, nil
}

// UpdateMaterial changes a material. The type cannot change. When a file is
// replaced, the old file is removed from the storage.
func (s *Lesson) UpdateMaterial(
	ctx context.Context,
	actor models.Actor,
	id uuid.UUID,
	req models.UpdateLessonMaterial,
) (*models.LessonMaterial, error) {
	material, err := s.repo.Lesson.GetMaterialByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_, _, course, err := s.lessonContext(ctx, material.LessonID)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, "UpdateMaterial"); err != nil {
		return nil, err
	}

	req.ID = id

	if material.Type == models.MaterialTypeFile {
		req.Content = nil

		if req.ObjectKey != nil {
			size, contentType, err := s.checkFile(ctx, req.ObjectKey, "UpdateMaterial")
			if err != nil {
				return nil, err
			}

			req.FileSize = &size

			if req.MimeType == nil && contentType != "" {
				req.MimeType = &contentType
			}
		}
	} else {
		if req.Content != nil {
			if err := validateContent(material.Type, req.Content, "UpdateMaterial"); err != nil {
				return nil, err
			}
		}

		req.ObjectKey, req.FileName, req.MimeType, req.FileSize = nil, nil, nil, nil
	}

	updated, err := s.repo.Lesson.UpdateMaterial(ctx, req)
	if err != nil {
		return nil, err
	}

	replaced := req.ObjectKey != nil && material.ObjectKey != nil && *req.ObjectKey != *material.ObjectKey
	if replaced {
		// the old file is garbage now; failing to remove it is not worth failing the update
		_ = s.storage.Delete(ctx, *material.ObjectKey)
	}

	if err := s.withFileURL(ctx, updated); err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteMaterial removes the material and its file.
func (s *Lesson) DeleteMaterial(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	material, err := s.repo.Lesson.GetMaterialByID(ctx, id)
	if err != nil {
		return err
	}

	_, _, course, err := s.lessonContext(ctx, material.LessonID)
	if err != nil {
		return err
	}

	if err := core.CanManage(actor, course.InstructorID, "DeleteMaterial"); err != nil {
		return err
	}

	if err := s.repo.Lesson.DeleteMaterial(ctx, id); err != nil {
		return err
	}

	if material.ObjectKey != nil {
		_ = s.storage.Delete(ctx, *material.ObjectKey)
	}

	return nil
}
