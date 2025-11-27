package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// GetByID - универсальная функция для получения сущности по ID
// Устраняет дубликаты кода GetByID во всех репозиториях
func GetByID[T any](ctx context.Context, db *sqlx.DB, table string, id interface{}, notFoundErr error) (*T, error) {
	var entity T
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", table)

	if err := db.GetContext(ctx, &entity, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFoundErr
		}
		return nil, fmt.Errorf("get by id from %s: %w", table, err)
	}

	return &entity, nil
}

// GetByField - универсальная функция для получения сущности по любому полю
func GetByField[T any](ctx context.Context, db *sqlx.DB, table, field string, value interface{}, notFoundErr error) (*T, error) {
	var entity T
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", table, field)

	if err := db.GetContext(ctx, &entity, query, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFoundErr
		}
		return nil, fmt.Errorf("get by %s from %s: %w", field, table, err)
	}

	return &entity, nil
}

// BatchInsert - универсальная функция для массовой вставки
// Устраняет N+1 проблемы при вставке в цикле
type BatchInserter struct {
	tx          *sqlx.Tx
	query       string
	batchSize   int
	values      []interface{}
	rowCount    int
	fieldsCount int
}

// NewBatchInserter создает новый batch inserter
func NewBatchInserter(tx *sqlx.Tx, baseQuery string, fieldsCount int, batchSize int) *BatchInserter {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &BatchInserter{
		tx:          tx,
		query:       baseQuery,
		batchSize:   batchSize,
		values:      make([]interface{}, 0, batchSize*fieldsCount),
		fieldsCount: fieldsCount,
	}
}

// Add добавляет строку для вставки
func (bi *BatchInserter) Add(ctx context.Context, rowValues ...interface{}) error {
	if len(rowValues) != bi.fieldsCount {
		return fmt.Errorf("expected %d fields, got %d", bi.fieldsCount, len(rowValues))
	}

	bi.values = append(bi.values, rowValues...)
	bi.rowCount++

	// Если достигли размера батча, выполняем вставку
	if bi.rowCount >= bi.batchSize {
		return bi.Flush(ctx)
	}

	return nil
}

// Flush выполняет вставку накопленных значений
func (bi *BatchInserter) Flush(ctx context.Context) error {
	if bi.rowCount == 0 {
		return nil
	}

	// Генерируем placeholders: ($1, $2, $3), ($4, $5, $6), ...
	placeholders := ""
	for i := 0; i < bi.rowCount; i++ {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "("
		for j := 0; j < bi.fieldsCount; j++ {
			if j > 0 {
				placeholders += ", "
			}
			placeholders += fmt.Sprintf("$%d", i*bi.fieldsCount+j+1)
		}
		placeholders += ")"
	}

	query := bi.query + " VALUES " + placeholders

	if _, err := bi.tx.ExecContext(ctx, query, bi.values...); err != nil {
		return fmt.Errorf("batch insert: %w", err)
	}

	// Очищаем буфер
	bi.values = bi.values[:0]
	bi.rowCount = 0

	return nil
}

// WithTransaction выполняет функцию внутри транзакции с правильной обработкой ошибок
func WithTransaction(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// При панике откатываем транзакцию
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	err = fn(tx)
	if err != nil {
		// При ошибке откатываем транзакцию
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
