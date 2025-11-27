package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Константы валидации
const (
	MinUsernameLength = 3
	MaxUsernameLength = 30
	MinDisplayNameLength = 2
	MaxDisplayNameLength = 100
	MinOrderTitleLength = 3
	MaxOrderTitleLength = 200
	MinOrderDescriptionLength = 10
	MaxOrderDescriptionLength = 5000
	MinProposalCoverLetterLength = 10
	MaxProposalCoverLetterLength = 2000
	MinPortfolioTitleLength = 1
	MaxPortfolioTitleLength = 200
	MaxPortfolioDescriptionLength = 2000
	MaxBioLength = 1000
	MaxLocationLength = 100
	MaxSkillLength = 50
	MaxSkillsCount = 50
	MinBudget = 0.0
	MaxBudget = 100000000.0 // 100 миллионов
	MinHourlyRate = 0.0
	MaxHourlyRate = 100000.0
	MinMessageLength = 1
	MaxMessageLength = 5000
	MaxExternalLinkLength = 500
)

// ValidateLength проверяет длину строки.
func ValidateLength(fieldName, value string, min, max int) error {
	length := utf8.RuneCountInString(value)
	if min > 0 && length < min {
		return fmt.Errorf("%s должен быть не менее %d символов", fieldName, min)
	}
	if max > 0 && length > max {
		return fmt.Errorf("%s должен быть не более %d символов", fieldName, max)
	}
	return nil
}

// ValidateEmail проверяет формат email.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email обязателен")
	}

	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	// Базовая проверка формата
	if !strings.Contains(email, "@") {
		return fmt.Errorf("email должен содержать символ @")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("некорректный формат email")
	}

	localPart := parts[0]
	domainPart := parts[1]

	if len(localPart) == 0 || len(localPart) > 64 {
		return fmt.Errorf("локальная часть email должна быть от 1 до 64 символов")
	}

	if len(domainPart) == 0 || len(domainPart) > 255 {
		return fmt.Errorf("доменная часть email должна быть от 1 до 255 символов")
	}

	if !strings.Contains(domainPart, ".") {
		return fmt.Errorf("доменная часть email должна содержать точку")
	}

	// Проверка на валидные символы в локальной части
	emailRegex := regexp.MustCompile(`^[a-z0-9._+-]+$`)
	if !emailRegex.MatchString(localPart) {
		return fmt.Errorf("локальная часть email содержит недопустимые символы")
	}

	// Проверка на валидные символы в доменной части
	domainRegex := regexp.MustCompile(`^[a-z0-9.-]+\.[a-z]{2,}$`)
	if !domainRegex.MatchString(domainPart) {
		return fmt.Errorf("доменная часть email имеет некорректный формат")
	}

	return nil
}

// ValidateNonEmpty проверяет, что строка не пустая.
func ValidateNonEmpty(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s не может быть пустым", fieldName)
	}
	return nil
}

// ValidateUsername проверяет имя пользователя.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("имя пользователя обязательно")
	}

	username = strings.TrimSpace(username)

	// Проверка длины
	if err := ValidateLength("имя пользователя", username, MinUsernameLength, MaxUsernameLength); err != nil {
		return err
	}

	// Проверка на допустимые символы (только буквы, цифры и подчеркивание)
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("имя пользователя может содержать только буквы, цифры и подчеркивание")
	}

	// Проверка, что не начинается с цифры
	if len(username) > 0 && unicode.IsDigit(rune(username[0])) {
		return fmt.Errorf("имя пользователя не может начинаться с цифры")
	}

	return nil
}

// ValidateDisplayName проверяет отображаемое имя.
func ValidateDisplayName(displayName string) error {
	if displayName == "" {
		return fmt.Errorf("отображаемое имя обязательно")
	}

	displayName = strings.TrimSpace(displayName)

	// Проверка длины
	if err := ValidateLength("отображаемое имя", displayName, MinDisplayNameLength, MaxDisplayNameLength); err != nil {
		return err
	}

	// Проверка на недопустимые символы (только буквы, цифры, пробелы и некоторые спецсимволы)
	displayNameRegex := regexp.MustCompile(`^[a-zA-Zа-яА-ЯёЁ0-9\s\-_.,!?()]+$`)
	if !displayNameRegex.MatchString(displayName) {
		return fmt.Errorf("отображаемое имя содержит недопустимые символы")
	}

	return nil
}

// ValidateOrderTitle проверяет заголовок заказа.
func ValidateOrderTitle(title string) error {
	if title == "" {
		return fmt.Errorf("заголовок заказа обязателен")
	}

	title = strings.TrimSpace(title)

	if err := ValidateLength("заголовок заказа", title, MinOrderTitleLength, MaxOrderTitleLength); err != nil {
		return err
	}

	return nil
}

// ValidateOrderDescription проверяет описание заказа.
func ValidateOrderDescription(description string) error {
	if description == "" {
		return fmt.Errorf("описание заказа обязательно")
	}

	description = strings.TrimSpace(description)

	if err := ValidateLength("описание заказа", description, MinOrderDescriptionLength, MaxOrderDescriptionLength); err != nil {
		return err
	}

	return nil
}

// ValidateProposalCoverLetter проверяет сопроводительное письмо.
func ValidateProposalCoverLetter(coverLetter string) error {
	if coverLetter == "" {
		return fmt.Errorf("сопроводительное письмо обязательно")
	}

	coverLetter = strings.TrimSpace(coverLetter)

	if err := ValidateLength("сопроводительное письмо", coverLetter, MinProposalCoverLetterLength, MaxProposalCoverLetterLength); err != nil {
		return err
	}

	return nil
}

// ValidateBudget проверяет бюджет.
func ValidateBudget(budgetMin, budgetMax *float64) error {
	if budgetMin != nil {
		if *budgetMin < MinBudget {
			return fmt.Errorf("минимальный бюджет не может быть отрицательным")
		}
		if *budgetMin > MaxBudget {
			return fmt.Errorf("минимальный бюджет не может превышать %.0f", MaxBudget)
		}
	}

	if budgetMax != nil {
		if *budgetMax < MinBudget {
			return fmt.Errorf("максимальный бюджет не может быть отрицательным")
		}
		if *budgetMax > MaxBudget {
			return fmt.Errorf("максимальный бюджет не может превышать %.0f", MaxBudget)
		}
	}

	if budgetMin != nil && budgetMax != nil {
		if *budgetMin > *budgetMax {
			return fmt.Errorf("минимальный бюджет не может быть больше максимального")
		}
	}

	return nil
}

// ValidateHourlyRate проверяет почасовую ставку.
func ValidateHourlyRate(rate *float64) error {
	if rate != nil {
		if *rate < MinHourlyRate {
			return fmt.Errorf("почасовая ставка не может быть отрицательной")
		}
		if *rate > MaxHourlyRate {
			return fmt.Errorf("почасовая ставка не может превышать %.0f", MaxHourlyRate)
		}
	}
	return nil
}

// ValidateSkills проверяет массив навыков.
func ValidateSkills(skills []string) error {
	if len(skills) > MaxSkillsCount {
		return fmt.Errorf("количество навыков не может превышать %d", MaxSkillsCount)
	}

	seen := make(map[string]bool)
	for _, skill := range skills {
		skill = strings.TrimSpace(skill)
		if skill == "" {
			return fmt.Errorf("навык не может быть пустым")
		}

		// Проверка длины навыка
		if utf8.RuneCountInString(skill) > MaxSkillLength {
			return fmt.Errorf("навык не может быть длиннее %d символов", MaxSkillLength)
		}

		// Проверка на дубликаты (без учета регистра)
		skillLower := strings.ToLower(skill)
		if seen[skillLower] {
			return fmt.Errorf("навык '%s' указан дважды", skill)
		}
		seen[skillLower] = true
	}

	return nil
}

// ValidateLocation проверяет местоположение.
func ValidateLocation(location *string) error {
	if location != nil && *location != "" {
		loc := strings.TrimSpace(*location)
		if err := ValidateLength("местоположение", loc, 0, MaxLocationLength); err != nil {
			return err
		}
	}
	return nil
}

// ValidateBio проверяет биографию.
func ValidateBio(bio *string) error {
	if bio != nil && *bio != "" {
		bioStr := strings.TrimSpace(*bio)
		if err := ValidateLength("биография", bioStr, 0, MaxBioLength); err != nil {
			return err
		}
	}
	return nil
}

// ValidatePortfolioTitle проверяет заголовок работы в портфолио.
func ValidatePortfolioTitle(title string) error {
	if title == "" {
		return fmt.Errorf("название работы обязательно")
	}

	title = strings.TrimSpace(title)

	if err := ValidateLength("название работы", title, MinPortfolioTitleLength, MaxPortfolioTitleLength); err != nil {
		return err
	}

	return nil
}

// ValidatePortfolioDescription проверяет описание работы в портфолио.
func ValidatePortfolioDescription(description *string) error {
	if description != nil && *description != "" {
		desc := strings.TrimSpace(*description)
		if err := ValidateLength("описание работы", desc, 0, MaxPortfolioDescriptionLength); err != nil {
			return err
		}
	}
	return nil
}

// ValidateExternalLink проверяет внешнюю ссылку.
func ValidateExternalLink(link *string) error {
	if link != nil && *link != "" {
		linkStr := strings.TrimSpace(*link)
		
		if err := ValidateLength("внешняя ссылка", linkStr, 0, MaxExternalLinkLength); err != nil {
			return err
		}

		// Проверка формата URL
		parsedURL, err := url.Parse(linkStr)
		if err != nil {
			return fmt.Errorf("некорректный формат URL")
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("ссылка должна начинаться с http:// или https://")
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("ссылка должна содержать доменное имя")
		}
	}
	return nil
}

// ValidateMessageContent проверяет содержимое сообщения.
func ValidateMessageContent(content string) error {
	if content == "" {
		return fmt.Errorf("сообщение не может быть пустым")
	}

	content = strings.TrimSpace(content)

	if err := ValidateLength("сообщение", content, MinMessageLength, MaxMessageLength); err != nil {
		return err
	}

	return nil
}

// ValidateRequirementSkill проверяет навык в требовании.
func ValidateRequirementSkill(skill string) error {
	if skill == "" {
		return fmt.Errorf("навык в требовании обязателен")
	}

	skill = strings.TrimSpace(skill)

	if err := ValidateLength("навык", skill, 1, MaxSkillLength); err != nil {
		return err
	}

	return nil
}
