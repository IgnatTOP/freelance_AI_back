package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/service"
)

// NotificationHandler обслуживает маршруты уведомлений.
type NotificationHandler struct {
	notifications *service.NotificationService
}

// NewNotificationHandler создаёт новый хэндлер.
func NewNotificationHandler(notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

// ListNotifications обрабатывает GET /notifications.
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	limit := common.ParseIntQuery(c, "limit", 20)
	offset := common.ParseIntQuery(c, "offset", 0)
	unreadOnly := c.Query("unread_only") == "true"

	notifications, err := h.notifications.ListNotifications(c.Request.Context(), userID, limit, offset, unreadOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetNotification обрабатывает GET /notifications/:id.
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор уведомления"})
		return
	}

	notification, err := h.notifications.GetNotification(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if notification.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет доступа к этому уведомлению"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// MarkAsRead обрабатывает PUT /notifications/:id/read.
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор уведомления"})
		return
	}

	if err := h.notifications.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "уведомление отмечено как прочитанное"})
}

// MarkAllAsRead обрабатывает PUT /notifications/read-all.
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.notifications.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "все уведомления отмечены как прочитанные"})
}

// DeleteNotification обрабатывает DELETE /notifications/:id.
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный идентификатор уведомления"})
		return
	}

	if err := h.notifications.DeleteNotification(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "уведомление не найдено"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "уведомление удалено"})
}

// CountUnread обрабатывает GET /notifications/unread/count.
func (h *NotificationHandler) CountUnread(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	count, err := h.notifications.CountUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

