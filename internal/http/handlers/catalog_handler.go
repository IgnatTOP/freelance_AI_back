package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/repository"
)

type CatalogHandler struct {
	catalog *repository.CatalogRepository
}

func NewCatalogHandler(catalog *repository.CatalogRepository) *CatalogHandler {
	return &CatalogHandler{catalog: catalog}
}

// ListCategories GET /catalog/categories
func (h *CatalogHandler) ListCategories(c *gin.Context) {
	categories, err := h.catalog.GetCategoriesWithChildren(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// GetCategory GET /catalog/categories/:slug
func (h *CatalogHandler) GetCategory(c *gin.Context) {
	slug := c.Param("slug")
	category, err := h.catalog.GetCategoryBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "категория не найдена"})
		return
	}

	// Получаем подкатегории
	children, _ := h.catalog.ListSubcategories(c.Request.Context(), category.ID)
	category.Children = children

	// Получаем навыки категории
	skills, _ := h.catalog.ListSkillsByCategory(c.Request.Context(), category.ID)

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"skills":   skills,
	})
}

// ListSkills GET /catalog/skills
func (h *CatalogHandler) ListSkills(c *gin.Context) {
	categoryID := c.Query("category_id")

	var skills interface{}
	var err error

	if categoryID != "" {
		id, parseErr := uuid.Parse(categoryID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверный category_id"})
			return
		}
		skills, err = h.catalog.ListSkillsByCategory(c.Request.Context(), id)
	} else {
		skills, err = h.catalog.ListSkills(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"skills": skills})
}
