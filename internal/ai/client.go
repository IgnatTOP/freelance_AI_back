package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// normalizeText нормализует текст: добавляет пробелы между словами, которые склеены вместе
// Использует эвристики для определения границ слов
func normalizeText(text string) string {
	if text == "" {
		return text
	}

	// Сначала нормализуем множественные пробелы
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Убираем пробелы перед знаками препинания
	text = regexp.MustCompile(`\s+([.,!?;:—\-()[\]{}«»""''])`).ReplaceAllString(text, "$1")

	// Добавляем пробелы после знаков препинания (если их нет)
	text = regexp.MustCompile(`([.,!?;:—])([а-яА-Яa-zA-Z])`).ReplaceAllString(text, "$1 $2")

	// Основная задача: добавить пробелы между склеенными словами
	// Ищем последовательности букв без пробелов (более 5 символов) и пытаемся разбить их на слова
	text = regexp.MustCompile(`([а-яА-Яa-zA-Z]{6,})`).ReplaceAllStringFunc(text, func(match string) string {
		var result strings.Builder
		runes := []rune(match)

		if len(runes) <= 5 {
			return match
		}

		for i := 0; i < len(runes); i++ {
			if i > 0 {
				prev := runes[i-1]
				curr := runes[i]

				shouldAddSpace := false

				if unicode.IsLetter(prev) && unicode.IsLetter(curr) {
					// Переход с маленькой на большую букву (camelCase) - граница слова
					if unicode.IsLower(prev) && unicode.IsUpper(curr) {
						shouldAddSpace = true
					}
					// Переход между кириллицей и латиницей - граница слова
					isPrevCyrillic := (prev >= 'а' && prev <= 'я') || (prev >= 'А' && prev <= 'Я') || prev == 'ё' || prev == 'Ё'
					isCurrCyrillic := (curr >= 'а' && curr <= 'я') || (curr >= 'А' && curr <= 'Я') || curr == 'ё' || curr == 'Ё'
					isPrevLatin := (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z')
					isCurrLatin := (curr >= 'a' && curr <= 'z') || (curr >= 'A' && curr <= 'Z')

					if (isPrevCyrillic && isCurrLatin) || (isPrevLatin && isCurrCyrillic) {
						shouldAddSpace = true
					}
				} else if (unicode.IsLetter(prev) && unicode.IsDigit(curr)) ||
					(unicode.IsDigit(prev) && unicode.IsLetter(curr)) {
					shouldAddSpace = true
				}

				if shouldAddSpace {
					result.WriteRune(' ')
				}
			}
			result.WriteRune(runes[i])
		}

		return result.String()
	})

	// Финальная нормализация: убираем множественные пробелы
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	// Убираем пробелы перед знаками препинания
	normalized = regexp.MustCompile(`\s+([.,!?;:—\-()[\]{}«»""''])`).ReplaceAllString(normalized, "$1")
	// Добавляем пробелы после знаков препинания (если их нет)
	normalized = regexp.MustCompile(`([.,!?;:—])([а-яА-Яa-zA-Z])`).ReplaceAllString(normalized, "$1 $2")

	return strings.TrimSpace(normalized)
}

// Client реализует простого AI помощника через OpenAI-совместимый API (Bothub).
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient создаёт экземпляр клиента.
func NewClient(baseURL, model string) *Client {
	apiKey := os.Getenv("BOTHUB_ACCESS_TOKEN")
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}

	// Fallback модели если основная не указана
	if model == "" {
		model = "grok-4.1-fast:free" // Лучшая бесплатная: 2M контекст, быстрая
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Уменьшено с 120s - grok быстрее
		},
	}
}

// streamInput выполняет запрос к Bothub Responses API с stream=true и
// передаёт текстовые чанки в onDelta.
func (c *Client) streamInput(
	ctx context.Context,
	input []map[string]any,
	onDelta func(chunk string) error,
) error {
	if c.baseURL == "" {
		return fmt.Errorf("ai: baseURL не задан")
	}

	payload := map[string]any{
		"model":  c.model,
		"input":  input,
		"stream": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := c.baseURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "responses"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errorBody)
		return fmt.Errorf("ai: код ответа %d: %v", resp.StatusCode, errorBody)
	}

	// Убеждаемся, что ответ декодируется как UTF-8
	reader := bufio.NewReader(resp.Body)

	// Буфер для накопления чанков перед отправкой
	buffer := strings.Builder{}
	var totalSentLength int         // Общая длина отправленного текста (для отслеживания дублирования)
	var hasReceivedDelta bool       // Флаг, что мы получали delta-чанки (инкрементальные обновления)
	const bufferFlushThreshold = 20 // Порог для отправки (меньше для более плавного стриминга)
	const maxBufferSize = 100       // Максимальный размер буфера
	const maxDeltaSize = 200        // Максимальный размер delta-чанка (если больше - это полный текст, игнорируем)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Отправляем остатки буфера перед завершением только если они есть
			if buffer.Len() > 0 {
				content := buffer.String()
				// Проверяем валидность UTF-8
				if !utf8.ValidString(content) {
					content = strings.ToValidUTF8(content, "")
				}
				if len(content) > 0 {
					if flushErr := onDelta(content); flushErr != nil {
						return flushErr
					}
					totalSentLength += len(content)
				}
				buffer.Reset()
			}
			if err == context.Canceled {
				return nil
			}
			if strings.Contains(err.Error(), "EOF") {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			if data == "[DONE]" {
				// Отправляем остатки буфера перед завершением
				if buffer.Len() > 0 {
					content := buffer.String()
					// Проверяем валидность UTF-8
					if !utf8.ValidString(content) {
						content = strings.ToValidUTF8(content, "")
					}
					if len(content) > 0 {
						if flushErr := onDelta(content); flushErr != nil {
							return flushErr
						}
						totalSentLength += len(content)
					}
					buffer.Reset()
				}
				// Явно завершаем, чтобы избежать повторной обработки при EOF
				return nil
			}
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			// Если это не JSON (или другой формат), пробуем трактовать как чистый текст.
			// Проверяем валидность UTF-8 перед добавлением в буфер
			if !utf8.ValidString(data) {
				// Если данные невалидны, пытаемся исправить, заменяя невалидные последовательности
				data = strings.ToValidUTF8(data, "")
			}
			buffer.WriteString(data)

			// Отправляем буфер если он достиг порога или максимального размера
			shouldFlush := buffer.Len() >= bufferFlushThreshold || buffer.Len() >= maxBufferSize

			if shouldFlush {
				content := buffer.String()
				// Проверяем валидность UTF-8
				if !utf8.ValidString(content) {
					content = strings.ToValidUTF8(content, "")
				}
				if len(content) > 0 {
					if flushErr := onDelta(content); flushErr != nil {
						return flushErr
					}
					totalSentLength += len(content)
				}
				buffer.Reset()
			}
			continue
		}

		// Пробуем извлечь текст из разных возможных полей
		var text string
		var isDelta bool // Флаг, что это delta-чанк (инкрементальное обновление)

		// Формат OpenAI/Bothub: choices[0].delta.content
		if choices, ok := event["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if delta, ok := choice["delta"].(map[string]any); ok {
					if txt, ok := delta["content"].(string); ok && txt != "" {
						text = txt
						isDelta = true
					}
				}
			}
		}

		// Поле "delta" (основной формат) - ПРИОРИТЕТНО
		if text == "" {
			if delta, ok := event["delta"].(string); ok && delta != "" {
				text = delta
				isDelta = true
			} else if delta, ok := event["delta"].(map[string]any); ok {
				// Если delta - это объект, пробуем извлечь text или content
				if txt, ok := delta["text"].(string); ok && txt != "" {
					text = txt
					isDelta = true
				} else if txt, ok := delta["content"].(string); ok && txt != "" {
					text = txt
					isDelta = true
				}
			}
		}

		// Альтернативные поля (response, text, content, message) - только если НЕ было delta-чанков
		// И только если размер небольшой (иначе это полный текст, который дублирует delta)
		// ВАЖНО: Если мы уже получали delta-чанки, полностью игнорируем эти поля
		if text == "" {
			if !hasReceivedDelta {
				// Если еще не получали delta-чанки, можем использовать альтернативные поля
				if txt, ok := event["text"].(string); ok && txt != "" && len(txt) <= maxDeltaSize {
					text = txt
				} else if txt, ok := event["content"].(string); ok && txt != "" && len(txt) <= maxDeltaSize {
					text = txt
				} else if txt, ok := event["message"].(string); ok && txt != "" && len(txt) <= maxDeltaSize {
					text = txt
				} else if response, ok := event["response"].(string); ok && response != "" && len(response) <= maxDeltaSize {
					text = response
				}
			}
			// Если hasReceivedDelta == true, полностью игнорируем альтернативные поля
		}

		// Если нашли текст, добавляем в буфер
		if text != "" {
			// Если это delta-чанк, отмечаем что мы получали инкрементальные обновления
			if isDelta {
				hasReceivedDelta = true
			}

			// Проверяем валидность UTF-8 перед добавлением в буфер
			if !utf8.ValidString(text) {
				// Если данные невалидны, пытаемся исправить, заменяя невалидные последовательности
				text = strings.ToValidUTF8(text, "")
			}
			buffer.WriteString(text)

			// Отправляем буфер если он достиг порога или максимального размера
			shouldFlush := buffer.Len() >= bufferFlushThreshold || buffer.Len() >= maxBufferSize

			if shouldFlush {
				content := buffer.String()
				// Проверяем валидность UTF-8
				if !utf8.ValidString(content) {
					content = strings.ToValidUTF8(content, "")
				}
				if len(content) > 0 {
					if err := onDelta(content); err != nil {
						return err
					}
					totalSentLength += len(content)
				}
				buffer.Reset()
			}
		}
	}
}

// StreamSummarizeOrder формирует краткое описание заказа потоково через Bothub Responses API.

// StreamProposalFeedback выполняет тот же анализ, что ProposalFeedback,
// но возвращает результат потоково через callback onDelta.

// ProposalAnalysisForClient анализирует отклик с точки зрения заказчика.
// Объясняет, почему стоит выбрать этого исполнителя, анализируя его навыки, профиль, предложение, портфолио и цену.

// RecommendBestProposal анализирует все отклики и рекомендует лучшего исполнителя для проекта.

// selectBestBySkills выбирает лучшего исполнителя на основе соответствия навыков требованиям
func selectBestBySkills(proposals []*models.Proposal, freelancerProfiles map[uuid.UUID]*models.Profile, requirements []models.OrderRequirement, orderTitle string) (*uuid.UUID, string) {
	if len(proposals) == 0 {
		return nil, ""
	}

	// Создаем карту требуемых навыков
	requiredSkillsMap := make(map[string]bool)
	for _, req := range requirements {
		requiredSkillsMap[strings.ToLower(req.Skill)] = true
	}

	bestProposal := proposals[0]
	bestMatchCount := 0
	bestProfile := freelancerProfiles[bestProposal.FreelancerID]

	// Находим исполнителя с наибольшим количеством соответствующих навыков
	for _, proposal := range proposals {
		profile := freelancerProfiles[proposal.FreelancerID]
		if profile == nil {
			continue
		}

		matchCount := 0
		for _, skill := range profile.Skills {
			skillLower := strings.ToLower(skill)
			for reqSkill := range requiredSkillsMap {
				if strings.Contains(skillLower, reqSkill) || strings.Contains(reqSkill, skillLower) {
					matchCount++
					break
				}
			}
		}

		if matchCount > bestMatchCount {
			bestMatchCount = matchCount
			bestProposal = proposal
			bestProfile = profile
		}
	}

	if bestMatchCount == 0 {
		return nil, ""
	}

	// Формируем обоснование
	matchingSkills := []string{}
	for _, skill := range bestProfile.Skills {
		skillLower := strings.ToLower(skill)
		for reqSkill := range requiredSkillsMap {
			if strings.Contains(skillLower, reqSkill) || strings.Contains(reqSkill, skillLower) {
				matchingSkills = append(matchingSkills, skill)
				break
			}
		}
	}

	justification := fmt.Sprintf("Рекомендован на основе соответствия навыков требованиям заказа. Исполнитель имеет опыт работы с: %s. ", strings.Join(matchingSkills, ", "))
	justification += fmt.Sprintf("Уровень опыта: %s. ", bestProfile.ExperienceLevel)
	if bestProposal.ProposedAmount != nil {
		justification += fmt.Sprintf("Предложенная цена: $%.2f. ", *bestProposal.ProposedAmount)
	}

	return &bestProposal.ID, justification
}

// parseBestProposalResponse парсит ответ AI и извлекает ID отклика и обоснование.
func parseBestProposalResponse(response string, proposals []*models.Proposal) (*uuid.UUID, string) {
	lines := strings.Split(response, "\n")
	var proposalID *uuid.UUID
	var justification strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lineUpper := strings.ToUpper(line)
		if strings.HasPrefix(lineUpper, "ID_ОТКЛИКА:") || strings.HasPrefix(lineUpper, "ID:") {
			// Извлекаем ID без изменения регистра (UUID чувствителен к регистру)
			var idStr string
			if strings.HasPrefix(lineUpper, "ID_ОТКЛИКА:") {
				idStr = strings.TrimSpace(line[len("ID_ОТКЛИКА:"):])
			} else {
				idStr = strings.TrimSpace(line[len("ID:"):])
			}
			if id, err := uuid.Parse(idStr); err == nil {
				// Проверяем, что такой отклик существует
				for _, p := range proposals {
					if p.ID == id {
						proposalID = &id
						break
					}
				}
			}
		} else if strings.HasPrefix(lineUpper, "ОБОСНОВАНИЕ:") || strings.HasPrefix(lineUpper, "ОБОСНОВАНИЕ") {
			var justStr string
			if strings.HasPrefix(lineUpper, "ОБОСНОВАНИЕ:") {
				justStr = strings.TrimSpace(line[len("ОБОСНОВАНИЕ:"):])
			} else {
				justStr = strings.TrimSpace(line[len("ОБОСНОВАНИЕ"):])
			}
			if justStr != "" {
				justification.WriteString(justStr)
				justification.WriteString("\n")
			}
		} else if proposalID != nil && (justification.Len() > 0 || strings.TrimSpace(line) != "") {
			// Добавляем к обоснованию, если уже начали его собирать
			if justification.Len() > 0 {
				justification.WriteString("\n")
			}
			justification.WriteString(line)
		}
	}

	// Если не нашли ID в формате, пытаемся найти UUID в тексте
	if proposalID == nil {
		for _, line := range lines {
			words := strings.Fields(line)
			for _, word := range words {
				if id, err := uuid.Parse(word); err == nil {
					// Проверяем, что такой отклик существует
					for _, p := range proposals {
						if p.ID == id {
							proposalID = &id
							break
						}
					}
					if proposalID != nil {
						break
					}
				}
			}
			if proposalID != nil {
				break
			}
		}
	}

	// Если обоснование пустое, используем весь ответ как обоснование
	justificationStr := strings.TrimSpace(justification.String())
	if justificationStr == "" {
		justificationStr = response
	}

	return proposalID, justificationStr
}

// fallbackBestProposalJustification формирует простое обоснование выбора.
func fallbackBestProposalJustification(orderTitle string, proposal *models.Proposal, profile *models.Profile) string {
	justification := fmt.Sprintf("Рекомендация для заказа \"%s\": ", orderTitle)

	if profile != nil {
		if len(profile.Skills) > 0 {
			justification += fmt.Sprintf("Исполнитель имеет релевантные навыки: %s. ", strings.Join(profile.Skills, ", "))
		}
		justification += fmt.Sprintf("Уровень опыта: %s. ", profile.ExperienceLevel)
	}

	if proposal.ProposedAmount != nil {
		justification += fmt.Sprintf("Предложенная цена: $%.2f. ", *proposal.ProposedAmount)
	}

	justification += "Этот исполнитель хорошо подходит для данного проекта."

	return justification
}


// chatCompletion выполняет запрос к OpenAI-совместимому API.
func (c *Client) chatCompletion(ctx context.Context, messages []map[string]string) (string, error) {
	return c.chatCompletionWithOptions(ctx, messages, 1024, 0.7)
}

// chatCompletionWithOptions выполняет запрос с настраиваемыми параметрами.
func (c *Client) chatCompletionWithOptions(ctx context.Context, messages []map[string]string, maxTokens int, temperature float64) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("ai: baseURL не задан")
	}

	payload := map[string]any{
		"model":       c.model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := c.baseURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errorBody)
		return "", fmt.Errorf("ai: код ответа %d: %v", resp.StatusCode, errorBody)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("ai: пустой ответ")
	}

	return result.Choices[0].Message.Content, nil
}

// post выполняет HTTP запрос к AI сервису (старый метод для обратной совместимости).
func (c *Client) post(ctx context.Context, path string, payload any) (map[string]any, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("ai: baseURL не задан")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ai: код ответа %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseJSONFromText пытается извлечь JSON из текста, который может содержать markdown или другие символы
func parseJSONFromText(text string) map[string]interface{} {
	result := make(map[string]interface{})

	// Пытаемся найти JSON объект в тексте
	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonStr := text[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result
		}
	}

	// Пытаемся найти JSON в markdown блоке
	if strings.Contains(text, "```") {
		codeBlockMatch := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```").FindStringSubmatch(text)
		if len(codeBlockMatch) > 1 {
			if err := json.Unmarshal([]byte(codeBlockMatch[1]), &result); err == nil {
				return result
			}
		}
	}

	return result
}

// fallbackSummary формирует простое описание.
func fallbackSummary(title, description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return fmt.Sprintf("Проект \"%s\" пока без подробностей — уточните требования.", title)
	}

	sentences := strings.Split(desc, ".")
	if len(sentences) > 2 {
		sentences = sentences[:2]
	}

	return fmt.Sprintf("Проект \"%s\": %s.", title, strings.Join(sentences, "."))
}

// fallbackFeedback формирует простую рекомендацию.
func fallbackFeedback(orderTitle, coverLetter string) string {
	cover := strings.ToLower(coverLetter)
	hints := []string{}

	if !strings.Contains(cover, "опыт") {
		hints = append(hints, "Добавьте краткое описание опыта, связанного с задачей.")
	}
	if !strings.Contains(cover, "срок") {
		hints = append(hints, "Укажите ориентировочные сроки выполнения.")
	}
	if !strings.Contains(cover, "вопрос") {
		hints = append(hints, "Задайте уточняющий вопрос, чтобы показать вовлечённость.")
	}

	if len(hints) == 0 {
		hints = append(hints, "Отклик выглядит убедительно, продолжайте в том же духе.")
	}

	return fmt.Sprintf("Советы для проекта \"%s\": %s", orderTitle, strings.Join(hints, " "))
}

// fallbackClientAnalysis формирует простой анализ для заказчика.
func fallbackClientAnalysis(orderTitle, coverLetter string, skills []string, proposedAmount *float64) string {
	analysis := fmt.Sprintf("Анализ отклика на заказ \"%s\": ", orderTitle)

	if len(skills) > 0 {
		analysis += fmt.Sprintf("Исполнитель имеет навыки: %s. ", strings.Join(skills, ", "))
	}

	if proposedAmount != nil {
		analysis += fmt.Sprintf("Предложенная цена: $%.2f. ", *proposedAmount)
	}

	analysis += "Предложение выглядит профессионально и заслуживает внимания."

	return analysis
}
