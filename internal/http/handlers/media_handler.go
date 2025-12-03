package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/h2non/filetype"

	"github.com/ignatzorin/freelance-backend/internal/http/handlers/common"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
	"github.com/ignatzorin/freelance-backend/internal/storage"
)

// Разрешённые типы файлов для загрузки
var allowedMimeTypes = map[string]bool{
	"image/jpeg":    true,
	"image/jpg":     true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// Разрешённые расширения файлов
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".svg":  true,
}

// MediaHandler управляет загрузкой и удалением медиа-файлов.
type MediaHandler struct {
	repo    *repository.MediaRepository
	storage *storage.PhotoStorage
}

// NewMediaHandler создаёт новый хэндлер.
func NewMediaHandler(repo *repository.MediaRepository, storage *storage.PhotoStorage) *MediaHandler {
	return &MediaHandler{repo: repo, storage: storage}
}

// UploadPhoto обрабатывает POST /media/photos.
func (h *MediaHandler) UploadPhoto(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "поле file обязательно"})
		return
	}

	// Валидация размера файла
	if file.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл не может быть пустым"})
		return
	}

	// Валидация расширения файла
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("неподдерживаемый формат файла. Разрешены: %s", strings.Join(getAllowedExtensions(), ", ")),
		})
		return
	}

	// Открываем файл для проверки магических байтов
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	// Читаем первые 512 байт для проверки магических байтов
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать файл"})
		return
	}

	// Проверяем магические байты (реальный тип файла)
	kind, err := filetype.Match(buffer[:n])
	if err != nil || kind == filetype.Unknown {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "не удалось определить тип файла. Разрешены только изображения",
		})
		return
	}

	// Проверяем, что это изображение
	contentType := kind.MIME.Value
	if !allowedMimeTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("неподдерживаемый тип файла (%s). Разрешены изображения: %s", contentType, strings.Join(getAllowedMimeTypes(), ", ")),
		})
		return
	}

	// Проверяем, что расширение соответствует реальному типу файла
	expectedExt := "." + kind.Extension
	// .jpg и .jpeg - это одно и то же
	if ext != expectedExt && !(ext == ".jpg" && expectedExt == ".jpeg") && !(ext == ".jpeg" && expectedExt == ".jpg") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("расширение файла (%s) не соответствует реальному типу (%s)", ext, expectedExt),
		})
		return
	}

	// Сбрасываем позицию файла для сохранения
	if seeker, ok := src.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось сбросить позицию файла"})
			return
		}
	}

	relativePath, size, err := h.storage.Save(c.Request.Context(), userID, file.Filename, src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	media := &models.MediaFile{
		UserID:   &userID,
		FilePath: filepath.ToSlash(relativePath),
		FileType: contentType,
		FileSize: size,
		IsPublic: true,
	}

	if err := h.repo.Create(c.Request.Context(), media); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, media)
}

// DeleteMedia обрабатывает DELETE /media/:id.
func (h *MediaHandler) DeleteMedia(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	mediaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор"})
		return
	}

	media, err := h.repo.GetByID(c.Request.Context(), mediaID)
	if err != nil {
		if errors.Is(err, repository.ErrMediaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "файл не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверка прав доступа: пользователь может удалять только свои файлы
	if media.UserID == nil || *media.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "у вас нет прав на удаление этого файла"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), mediaID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.storage.Delete(c.Request.Context(), media.FilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// getAllowedExtensions возвращает список разрешённых расширений.
func getAllowedExtensions() []string {
	exts := make([]string, 0, len(allowedExtensions))
	for ext := range allowedExtensions {
		exts = append(exts, ext)
	}
	return exts
}

// getAllowedMimeTypes возвращает список разрешённых MIME типов.
func getAllowedMimeTypes() []string {
	types := make([]string, 0, len(allowedMimeTypes))
	for mimeType := range allowedMimeTypes {
		types = append(types, mimeType)
	}
	return types
}

// getMimeTypeByExtension возвращает MIME тип по расширению файла.
func getMimeTypeByExtension(ext string) string {
	mimeMap := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
