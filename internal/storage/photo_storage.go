package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PhotoStorage отвечает за файловое хранилище изображений.
type PhotoStorage struct {
	rootPath       string
	maxUploadBytes int64
}

// NewPhotoStorage создаёт файловое хранилище.
func NewPhotoStorage(rootPath string, maxUploadMB int64) (*PhotoStorage, error) {
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		return nil, fmt.Errorf("storage: не удалось создать каталог %s: %w", rootPath, err)
	}

	return &PhotoStorage{
		rootPath:       rootPath,
		maxUploadBytes: maxUploadMB * 1024 * 1024,
	}, nil
}

// Save сохраняет файл и возвращает относительный путь.
func (s *PhotoStorage) Save(ctx context.Context, userID uuid.UUID, originalName string, r io.Reader) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}

	safeName := sanitizeFilename(originalName)
	fileName := fmt.Sprintf("%s_%d%s", userID.String(), time.Now().UnixNano(), filepath.Ext(safeName))

	userDir := filepath.Join(s.rootPath, userID.String())
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return "", 0, fmt.Errorf("storage: не удалось создать каталог пользователя: %w", err)
	}

	targetPath := filepath.Join(userDir, fileName)
	tempPath := targetPath + ".tmp"

	f, err := os.Create(tempPath)
	if err != nil {
		return "", 0, fmt.Errorf("storage: не удалось создать файл: %w", err)
	}
	defer f.Close()

	limitedReader := io.LimitedReader{R: r, N: s.maxUploadBytes + 1}
	written, err := io.Copy(f, &limitedReader)
	if err != nil {
		_ = os.Remove(tempPath)
		return "", 0, fmt.Errorf("storage: ошибка записи файла: %w", err)
	}

	if written > s.maxUploadBytes {
		_ = os.Remove(tempPath)
		return "", 0, fmt.Errorf("storage: размер файла превышает лимит %d байт", s.maxUploadBytes)
	}

	if err := f.Close(); err != nil {
		return "", 0, fmt.Errorf("storage: ошибка закрытия файла: %w", err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return "", 0, fmt.Errorf("storage: не удалось переименовать файл: %w", err)
	}

	relative := filepath.Join(userID.String(), fileName)
	return relative, written, nil
}

// Delete удаляет файл из хранилища.
func (s *PhotoStorage) Delete(ctx context.Context, relativePath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	target := filepath.Join(s.rootPath, relativePath)
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: не удалось удалить файл: %w", err)
	}
	return nil
}

// sanitizeFilename удаляет потенциально опасные символы.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" {
		name = "photo"
	}
	return name
}
