package goroutine

import (
	"context"
	"fmt"
	"runtime/debug"
)

// Logger интерфейс для логирования ошибок
type Logger interface {
	Errorf(format string, args ...interface{})
}

// RecoveryHandler обрабатывает panic в горутинах
type RecoveryHandler struct {
	logger Logger
}

// NewRecoveryHandler создает новый обработчик
func NewRecoveryHandler(logger Logger) *RecoveryHandler {
	return &RecoveryHandler{logger: logger}
}

// SafeGo запускает горутину с обработкой panic
func (rh *RecoveryHandler) SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				rh.logger.Errorf("Panic in goroutine: %v\nStack trace:\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}

// SafeGoWithContext запускает горутину с контекстом и обработкой panic
func (rh *RecoveryHandler) SafeGoWithContext(ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				rh.logger.Errorf("Panic in goroutine (with context): %v\nStack trace:\n%s", r, debug.Stack())
			}
		}()
		fn(ctx)
	}()
}

// SimpleLogger - простая реализация Logger для fmt.Printf
type SimpleLogger struct{}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// DefaultRecoveryHandler - глобальный обработчик с простым логированием
var DefaultRecoveryHandler = NewRecoveryHandler(&SimpleLogger{})

// SafeGo - упрощенная функция для запуска безопасной горутины
func SafeGo(fn func()) {
	DefaultRecoveryHandler.SafeGo(fn)
}

// SafeGoWithContext - упрощенная функция для запуска безопасной горутины с контекстом
func SafeGoWithContext(ctx context.Context, fn func(context.Context)) {
	DefaultRecoveryHandler.SafeGoWithContext(ctx, fn)
}
