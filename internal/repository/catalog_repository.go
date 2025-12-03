package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

type CatalogRepository struct {
	db *sqlx.DB
}

func NewCatalogRepository(db *sqlx.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

// ListCategories возвращает все активные категории.
func (r *CatalogRepository) ListCategories(ctx context.Context) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.SelectContext(ctx, &categories, `
		SELECT id, slug, name, description, icon, parent_id, sort_order, is_active, created_at
		FROM categories WHERE is_active = TRUE ORDER BY sort_order, name
	`)
	return categories, err
}

// ListRootCategories возвращает только корневые категории (без parent_id).
func (r *CatalogRepository) ListRootCategories(ctx context.Context) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.SelectContext(ctx, &categories, `
		SELECT id, slug, name, description, icon, parent_id, sort_order, is_active, created_at
		FROM categories WHERE is_active = TRUE AND parent_id IS NULL ORDER BY sort_order, name
	`)
	return categories, err
}

// ListSubcategories возвращает подкатегории для указанной категории.
func (r *CatalogRepository) ListSubcategories(ctx context.Context, parentID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.SelectContext(ctx, &categories, `
		SELECT id, slug, name, description, icon, parent_id, sort_order, is_active, created_at
		FROM categories WHERE is_active = TRUE AND parent_id = $1 ORDER BY sort_order, name
	`, parentID)
	return categories, err
}

// GetCategoryBySlug возвращает категорию по slug.
func (r *CatalogRepository) GetCategoryBySlug(ctx context.Context, slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.GetContext(ctx, &category, `
		SELECT id, slug, name, description, icon, parent_id, sort_order, is_active, created_at
		FROM categories WHERE slug = $1 AND is_active = TRUE
	`, slug)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetCategoryByID возвращает категорию по ID.
func (r *CatalogRepository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	var category models.Category
	err := r.db.GetContext(ctx, &category, `
		SELECT id, slug, name, description, icon, parent_id, sort_order, is_active, created_at
		FROM categories WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// ListSkills возвращает все активные навыки.
func (r *CatalogRepository) ListSkills(ctx context.Context) ([]models.Skill, error) {
	var skills []models.Skill
	err := r.db.SelectContext(ctx, &skills, `
		SELECT id, slug, name, category_id, is_active, created_at
		FROM skills WHERE is_active = TRUE ORDER BY name
	`)
	return skills, err
}

// ListSkillsByCategory возвращает навыки для указанной категории.
func (r *CatalogRepository) ListSkillsByCategory(ctx context.Context, categoryID uuid.UUID) ([]models.Skill, error) {
	var skills []models.Skill
	err := r.db.SelectContext(ctx, &skills, `
		SELECT id, slug, name, category_id, is_active, created_at
		FROM skills WHERE is_active = TRUE AND category_id = $1 ORDER BY name
	`, categoryID)
	return skills, err
}

// GetSkillBySlug возвращает навык по slug.
func (r *CatalogRepository) GetSkillBySlug(ctx context.Context, slug string) (*models.Skill, error) {
	var skill models.Skill
	err := r.db.GetContext(ctx, &skill, `
		SELECT id, slug, name, category_id, is_active, created_at
		FROM skills WHERE slug = $1 AND is_active = TRUE
	`, slug)
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

// GetCategoriesWithChildren возвращает категории с вложенными подкатегориями.
func (r *CatalogRepository) GetCategoriesWithChildren(ctx context.Context) ([]models.Category, error) {
	categories, err := r.ListRootCategories(ctx)
	if err != nil {
		return nil, err
	}

	for i := range categories {
		children, err := r.ListSubcategories(ctx, categories[i].ID)
		if err != nil {
			continue
		}
		categories[i].Children = children
	}

	return categories, nil
}
