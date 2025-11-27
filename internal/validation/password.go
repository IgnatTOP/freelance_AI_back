package validation

import (
	"fmt"
	"unicode"
)

// ValidatePassword проверяет пароль на соответствие требованиям безопасности.
// Требования:
// - Минимум 8 символов
// - Должен содержать заглавные буквы
// - Должен содержать строчные буквы
// - Должен содержать цифры
// - Опционально: специальные символы
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("пароль должен быть не менее 8 символов")
	}

	var (
		hasUpper  = false
		hasLower  = false
		hasNumber = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("пароль должен содержать хотя бы одну заглавную букву")
	}
	if !hasLower {
		return fmt.Errorf("пароль должен содержать хотя бы одну строчную букву")
	}
	if !hasNumber {
		return fmt.Errorf("пароль должен содержать хотя бы одну цифру")
	}

	return nil
}
