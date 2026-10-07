package catalog

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/google/uuid"
)

type Category struct {
	repo  *repo.Repository
	cache *core.Cache
}

func NewCategory(deps core.Dependencies) *Category {
	return &Category{
		repo:  deps.Repo,
		cache: core.NewCache(deps.Redis),
	}
}

// changed forgets the cached categories, and the cached courses, which show
// the name of their category.
func (s *Category) changed(ctx context.Context) {
	s.cache.Invalidate(ctx, core.CacheCategories)
	s.cache.Invalidate(ctx, core.CacheCourses)
}

func (s *Category) Create(ctx context.Context, req models.CreateCategory) (*models.Category, error) {
	id, err := s.repo.Category.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	s.changed(ctx)

	return s.repo.Category.GetByID(ctx, id)
}

func (s *Category) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return core.Cached(ctx, s.cache, core.CacheCategories, "get:"+id.String(), func() (*models.Category, bool, error) {
		category, err := s.repo.Category.GetByID(ctx, id)

		return category, err == nil, err
	})
}

func (s *Category) GetList(ctx context.Context, filter models.CategoryFilter) (*models.ListResponse[models.Category], error) {
	return core.Cached(ctx, s.cache, core.CacheCategories, core.CacheKey("list", filter), func() (*models.ListResponse[models.Category], bool, error) {
		list, err := s.list(ctx, filter)

		return list, err == nil, err
	})
}

func (s *Category) list(ctx context.Context, filter models.CategoryFilter) (*models.ListResponse[models.Category], error) {
	categories, total, err := s.repo.Category.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}

	response := core.NewListResponse(categories, total, filter.Page, filter.Limit)

	return &response, nil
}

func (s *Category) Update(ctx context.Context, req models.UpdateCategory) (*models.Category, error) {
	category, err := s.repo.Category.Update(ctx, req)
	if err != nil {
		return nil, err
	}

	s.changed(ctx)

	return category, nil
}

func (s *Category) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Category.Delete(ctx, id); err != nil {
		return err
	}

	s.changed(ctx)

	return nil
}
