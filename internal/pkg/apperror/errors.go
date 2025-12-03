package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	ErrCodeNotFound        ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden       ErrorCode = "FORBIDDEN"
	ErrCodeBadRequest      ErrorCode = "BAD_REQUEST"
	ErrCodeConflict        ErrorCode = "CONFLICT"
	ErrCodeInternal        ErrorCode = "INTERNAL_ERROR"
	ErrCodeValidation      ErrorCode = "VALIDATION_ERROR"
	ErrCodeDatabaseError   ErrorCode = "DATABASE_ERROR"
)

type AppError struct {
	Code       ErrorCode
	Message    string
	HTTPStatus int
	Cause      error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
	}
}

func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
		Cause:      err,
	}
}

func codeToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeBadRequest, ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func IsNotFound(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == ErrCodeNotFound
}

func IsForbidden(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == ErrCodeForbidden
}

func IsValidation(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == ErrCodeValidation
}

var (
	ErrOrderNotFound        = New(ErrCodeNotFound, "заказ не найден")
	ErrProposalNotFound     = New(ErrCodeNotFound, "предложение не найдено")
	ErrConversationNotFound = New(ErrCodeNotFound, "беседа не найдена")
	ErrUserNotFound         = New(ErrCodeNotFound, "пользователь не найден")
	ErrUnauthorized         = New(ErrCodeUnauthorized, "требуется авторизация")
	ErrForbidden            = New(ErrCodeForbidden, "недостаточно прав")
	ErrInvalidCredentials   = New(ErrCodeUnauthorized, "неверные учетные данные")
)
