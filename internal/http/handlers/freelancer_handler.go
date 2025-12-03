package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

type FreelancerHandler struct {
	userRepo *repository.UserRepository
}

func NewFreelancerHandler(r *repository.UserRepository) *FreelancerHandler {
	return &FreelancerHandler{userRepo: r}
}

// SearchFreelancers GET /freelancers/search
func (h *FreelancerHandler) SearchFreelancers(c *gin.Context) {
	limit, offset := common.GetPagination(c)

	params := repository.FreelancerSearchParams{
		Query:           c.Query("q"),
		ExperienceLevel: c.Query("experience_level"),
		Location:        c.Query("location"),
		Limit:           limit,
		Offset:          offset,
	}

	if skills := c.Query("skills"); skills != "" {
		params.Skills = strings.Split(skills, ",")
	}
	if v := c.Query("min_hourly_rate"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			params.MinHourlyRate = &f
		}
	}
	if v := c.Query("max_hourly_rate"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			params.MaxHourlyRate = &f
		}
	}
	if v := c.Query("min_rating"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			params.MinRating = &f
		}
	}

	results, err := h.userRepo.SearchFreelancers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}
