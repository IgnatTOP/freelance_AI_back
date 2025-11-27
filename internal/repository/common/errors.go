package common

import "errors"

// Общие ошибки для всех репозиториев
var (
	ErrNotFound      = errors.New("entity not found")
	ErrAlreadyExists = errors.New("entity already exists")
	ErrInvalidInput  = errors.New("invalid input")
)
