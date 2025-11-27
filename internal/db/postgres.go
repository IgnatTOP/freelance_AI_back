package db

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// NewPostgres создаёт подключение к PostgreSQL с заданным DSN.
func NewPostgres(ctx context.Context, dsn string) (*sqlx.DB, error) {
	conn, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: не удалось подключиться: %w", err)
	}

	// Настраиваем пул соединений для оптимальной производительности.
	// MaxOpenConns: максимальное количество открытых соединений
	// MaxIdleConns: количество соединений в пуле простоя
	// ConnMaxLifetime: максимальное время жизни соединения
	conn.SetMaxOpenConns(100)
	conn.SetMaxIdleConns(25)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return conn, nil
}

// RunMigrations выполняет SQL файлы из каталога с миграциями.
func RunMigrations(ctx context.Context, conn *sqlx.DB, migrationsDir string) error {
	// Создаём таблицу для отслеживания выполненных миграций
	if err := initMigrationsTable(ctx, conn); err != nil {
		return fmt.Errorf("postgres: не удалось инициализировать таблицу миграций: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("postgres: не удалось прочитать каталог миграций: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		migrationName := entry.Name()
		
		// Проверяем, была ли миграция уже выполнена
		alreadyApplied, err := isMigrationApplied(ctx, conn, migrationName)
		if err != nil {
			return fmt.Errorf("postgres: не удалось проверить статус миграции %s: %w", migrationName, err)
		}
		
		if alreadyApplied {
			continue
		}

		path := filepath.Join(migrationsDir, migrationName)
		if err := applyMigration(ctx, conn, path, migrationName); err != nil {
			return err
		}
	}

	return nil
}

// initMigrationsTable создаёт таблицу для отслеживания выполненных миграций.
func initMigrationsTable(ctx context.Context, conn *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := conn.ExecContext(ctx, query)
	return err
}

// isMigrationApplied проверяет, была ли миграция уже выполнена.
func isMigrationApplied(ctx context.Context, conn *sqlx.DB, migrationName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM schema_migrations WHERE name = $1`
	err := conn.GetContext(ctx, &count, query, migrationName)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyMigration читает и выполняет конкретный SQL файл.
func applyMigration(ctx context.Context, conn *sqlx.DB, path string, migrationName string) error {
	sqlBytes, err := fs.ReadFile(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if err != nil {
		return fmt.Errorf("postgres: не удалось прочитать миграцию %s: %w", path, err)
	}

	// Выполняем миграцию в транзакции
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: не удалось начать транзакцию для миграции %s: %w", migrationName, err)
	}
	defer tx.Rollback()

	// Выполняем SQL миграции
	if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("postgres: не удалось выполнить миграцию %s: %w", path, err)
	}

	// Отмечаем миграцию как выполненную
	_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, migrationName)
	if err != nil {
		return fmt.Errorf("postgres: не удалось отметить миграцию %s как выполненную: %w", migrationName, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: не удалось зафиксировать транзакцию для миграции %s: %w", migrationName, err)
	}

	return nil
}
