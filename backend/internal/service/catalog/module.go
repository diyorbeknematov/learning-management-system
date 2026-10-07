package catalog

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/google/uuid"
)

type Module struct {
	repo  *repo.Repository
	cache *core.Cache
}

func NewModule(deps core.Dependencies) *Module {
	return &Module{
		repo:  deps.Repo,
		cache: core.NewCache(deps.Redis),
	}
}

// saved forgets the cached course pages, whose syllabus has the modules, after
// a change was written; a failed change leaves the cache alone.
func (s *Module) saved(ctx context.Context, err error) error {
	if err == nil {
		s.cache.Invalidate(ctx, core.CacheCourses)
	}

	return err
}

func moduleIDs(modules []models.Module) []uuid.UUID {
	ids := make([]uuid.UUID, len(modules))

	for i, module := range modules {
		ids[i] = module.ID
	}

	return ids
}

func setModuleOrder(tx *repo.Repository) func(context.Context, uuid.UUID, int) error {
	return func(ctx context.Context, id uuid.UUID, number int) error {
		return tx.Module.UpdateOrder(ctx, models.UpdateModuleOrder{ID: id, OrderNumber: number})
	}
}

// Create adds a module to the course of the actor. The other modules move
// down when the new one takes a place in the middle.
func (s *Module) Create(
	ctx context.Context,
	actor models.Actor,
	courseID uuid.UUID,
	req models.CreateModuleRequest,
) (*models.Module, error) {
	if _, err := core.ManageableCourse(ctx, s.repo, actor, courseID, "CreateModule"); err != nil {
		return nil, err
	}

	var id uuid.UUID

	err := s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		existing, err := tx.Module.GetListByCourseID(ctx, courseID)
		if err != nil {
			return err
		}

		position := len(existing) + 1
		if req.OrderNumber != nil && *req.OrderNumber < position {
			position = *req.OrderNumber
		}

		id, err = tx.Module.Create(ctx, models.CreateModule{
			CourseID:    courseID,
			Title:       req.Title,
			Description: req.Description,
			OrderNumber: core.TempOrderBase * 2,
		})
		if err != nil {
			return err
		}

		return core.Renumber(ctx, core.MoveTo(moduleIDs(existing), id, position), setModuleOrder(tx))
	})
	if err := s.saved(ctx, err); err != nil {
		return nil, err
	}

	return s.repo.Module.GetByID(ctx, id)
}

// GetByID returns a module with its lessons. The syllabus of a published
// course is open to everybody.
func (s *Module) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.ModuleDetail, error) {
	module, err := s.repo.Module.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := core.VisibleCourse(ctx, s.repo, actor, module.CourseID, "GetModule"); err != nil {
		return nil, err
	}

	lessons, err := s.repo.Lesson.GetListByModuleID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &models.ModuleDetail{Module: *module, Lessons: lessons}, nil
}

// GetListByCourse returns the modules of a course with their lessons, in order.
func (s *Module) GetListByCourse(ctx context.Context, actor models.Actor, courseID uuid.UUID) ([]models.ModuleDetail, error) {
	if _, err := core.VisibleCourse(ctx, s.repo, actor, courseID, "GetModuleList"); err != nil {
		return nil, err
	}

	modules, err := s.repo.Module.GetListByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	details := make([]models.ModuleDetail, 0, len(modules))

	for _, module := range modules {
		lessons, err := s.repo.Lesson.GetListByModuleID(ctx, module.ID)
		if err != nil {
			return nil, err
		}

		details = append(details, models.ModuleDetail{Module: module, Lessons: lessons})
	}

	return details, nil
}

func (s *Module) Update(
	ctx context.Context,
	actor models.Actor,
	id uuid.UUID,
	req models.UpdateModule,
) (*models.Module, error) {
	module, err := s.repo.Module.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := core.ManageableCourse(ctx, s.repo, actor, module.CourseID, "UpdateModule"); err != nil {
		return nil, err
	}

	req.ID = id

	updated, err := s.repo.Module.Update(ctx, req)
	if err := s.saved(ctx, err); err != nil {
		return nil, err
	}

	return updated, nil
}

// UpdateOrder moves the module to the given place; the modules in between
// shift by one.
func (s *Module) UpdateOrder(ctx context.Context, actor models.Actor, id uuid.UUID, order int) error {
	module, err := s.repo.Module.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if _, err := core.ManageableCourse(ctx, s.repo, actor, module.CourseID, "UpdateModuleOrder"); err != nil {
		return err
	}

	return s.saved(ctx, s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		modules, err := tx.Module.GetListByCourseID(ctx, module.CourseID)
		if err != nil {
			return err
		}

		return core.Renumber(ctx, core.MoveTo(moduleIDs(modules), id, order), setModuleOrder(tx))
	}))
}

// Delete removes the module with its lessons and closes the gap in the order.
func (s *Module) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	module, err := s.repo.Module.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if _, err := core.ManageableCourse(ctx, s.repo, actor, module.CourseID, "DeleteModule"); err != nil {
		return err
	}

	return s.saved(ctx, s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		if err := tx.Lesson.DeleteByModuleID(ctx, id); err != nil {
			return err
		}

		if err := tx.Module.Delete(ctx, id); err != nil {
			return err
		}

		remaining, err := tx.Module.GetListByCourseID(ctx, module.CourseID)
		if err != nil {
			return err
		}

		return core.Renumber(ctx, moduleIDs(remaining), setModuleOrder(tx))
	}))
}
