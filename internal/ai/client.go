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

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SummarizeOrder формирует краткое описание заказа.
func (c *Client) SummarizeOrder(ctx context.Context, title, description string) (string, error) {
	prompt := fmt.Sprintf(`Создай краткое резюме (2-3 предложения) для заказа на фриланс-платформе.

Заголовок: %s
Описание: %s

Резюме должно быть информативным и привлекательным для исполнителей.`, title, description)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Создавай краткие и информативные резюме заказов."},
		{"role": "user", "content": prompt},
	}

	summary, err := c.chatCompletion(ctx, messages)
	if err == nil && summary != "" {
		return strings.TrimSpace(summary), nil
	}

	// Фолбэк: строим краткое описание из первых предложений.
	return fallbackSummary(title, description), nil
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
// По аналогии с SummarizeOrder, но результат приходит чанками в onDelta.
func (c *Client) StreamSummarizeOrder(
	ctx context.Context,
	title, description string,
	onDelta func(chunk string) error,
) error {
	prompt := fmt.Sprintf(`Создай краткое резюме (2-3 предложения) для заказа на фриланс-платформе.

Заголовок: %s
Описание: %s

	Резюме должно быть информативным и привлекательным для исполнителей.`, title, description)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// ProposalFeedback формирует рекомендации по отклику для исполнителя.
func (c *Client) ProposalFeedback(ctx context.Context, order *models.Order, coverLetter string) (string, error) {
	prompt := fmt.Sprintf(`Проанализируй отклик на заказ и дай очень краткие рекомендации по улучшению (1-3 лаконичных пункта).

Заказ: %s
Описание заказа: %s
Отклик: %s

Дай только конкретные и краткие советы, без лишних деталей и воды. Максимум 3 коротких пункта.`, order.Title, order.Description, coverLetter)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Давай очень краткие, практичные советы по улучшению откликов (1-3 коротких пункта, без воды)."},
		{"role": "user", "content": prompt},
	}

	feedback, err := c.chatCompletion(ctx, messages)
	if err == nil && feedback != "" {
		return strings.TrimSpace(feedback), nil
	}

	return fallbackFeedback(order.Title, coverLetter), nil
}

// StreamProposalFeedback выполняет тот же анализ, что ProposalFeedback,
// но возвращает результат потоково через callback onDelta.
// Использует Bothub Responses API со stream=true.
func (c *Client) StreamProposalFeedback(
	ctx context.Context,
	order *models.Order,
	coverLetter string,
	onDelta func(chunk string) error,
) error {
	prompt := fmt.Sprintf(`Проанализируй отклик на заказ и дай очень краткие рекомендации по улучшению (1-3 лаконичных пункта).

Заказ: %s
Описание заказа: %s
Отклик: %s

	Дай только конкретные и краткие советы, без лишних деталей и воды. Максимум 3 коротких пункта.`, order.Title, order.Description, coverLetter)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// ProposalAnalysisForClient анализирует отклик с точки зрения заказчика.
// Объясняет, почему стоит выбрать этого исполнителя, анализируя его навыки, профиль, предложение, портфолио и цену.
// portfolioItems должен быть []models.PortfolioItemForAI либо эквивалентной по полям структурой.
func (c *Client) ProposalAnalysisForClient(ctx context.Context, order *models.Order, proposal *models.Proposal, freelancerProfile *models.Profile, requirements []models.OrderRequirement, portfolioItems interface{}, otherProposals []*models.Proposal) (string, error) {
	// Формируем информацию о навыках исполнителя
	skillsStr := formatSkillsStr(freelancerProfile.Skills)
	if skillsStr != "" {
		skillsStr = "\nНавыки исполнителя" + skillsStr[7:] // Убираем "\nНавыки: " и добавляем "Навыки исполнителя: "
	}

	// Формируем информацию о профиле
	profileInfo := formatProfileInfo(freelancerProfile)

	// Формируем информацию о требованиях заказа
	requirementsStr := formatRequirementsStrSimple(requirements)

	// Формируем информацию о цене
	priceInfo := formatPriceInfo(proposal.ProposedAmount, order.BudgetMin, order.BudgetMax)

	// Формируем информацию о портфолио исполнителя
	items := normalizePortfolioItems(portfolioItems)
	portfolioStr := formatPortfolioStr(items, "\n\nРаботы из портфолио исполнителя:\n")

	// Формируем информацию о других откликах для сравнения
	comparisonInfo := formatComparisonInfo(otherProposals)

	prompt := fmt.Sprintf(`Ты помощник для заказчика на фриланс-платформе. Проанализируй отклик исполнителя и очень кратко объясни, почему стоит выбрать именно этого исполнителя для данного проекта.

Заказ: %s
Описание заказа: %s%s

Отклик исполнителя: %s%s%s%s%s%s

Сформулируй ответ в виде краткого анализа для заказчика (2-3 предложения, максимум ~400 символов), который помогает принять решение. Не повторяй длинно текст заказа или отклика, фокусируйся на 2-3 ключевых плюсах исполнителя.`,
		order.Title,
		order.Description,
		requirementsStr,
		proposal.CoverLetter,
		skillsStr,
		profileInfo,
		priceInfo,
		portfolioStr,
		comparisonInfo)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для заказчика на фриланс-платформе. Отвечай очень кратко (2-3 предложения, максимум ~400 символов), только по сути и без воды."},
		{"role": "user", "content": prompt},
	}

	analysis, err := c.chatCompletion(ctx, messages)
	if err == nil && analysis != "" {
		return strings.TrimSpace(analysis), nil
	}

	// Фолбэк анализ
	return fallbackClientAnalysis(order.Title, proposal.CoverLetter, freelancerProfile.Skills, proposal.ProposedAmount), nil
}

// RecommendBestProposal анализирует все отклики и рекомендует лучшего исполнителя для проекта.
// Возвращает ID лучшего отклика и обоснование выбора.
func (c *Client) RecommendBestProposal(ctx context.Context, order *models.Order, proposals []*models.Proposal, freelancerProfiles map[uuid.UUID]*models.Profile, requirements []models.OrderRequirement) (*uuid.UUID, string, error) {
	if len(proposals) == 0 {
		return nil, "", fmt.Errorf("нет откликов для анализа")
	}

	// Формируем информацию о требованиях заказа
	requirementsStr := formatRequirementsStrSimple(requirements)

	// Формируем информацию о бюджете
	budgetStr := formatBudgetStr(order.BudgetMin, order.BudgetMax)

	// Формируем детальную информацию о каждом отклике
	proposalsInfo := formatProposalsInfo(proposals, freelancerProfiles, requirements)

	prompt := fmt.Sprintf(`Ты помощник для заказчика на фриланс-платформе. Проанализируй все отклики на заказ и выбери ЛУЧШЕГО исполнителя для данного проекта.

ЗАКАЗ:
Название: %s
Описание: %s%s%s

ОТКЛИКИ НА ЗАКАЗ:%s

КРИТЕРИИ ВЫБОРА (в порядке приоритета):
1. СООТВЕТСТВИЕ НАВЫКОВ ТРЕБОВАНИЯМ - это ГЛАВНЫЙ критерий. Исполнитель должен иметь опыт работы с требуемыми технологиями и навыками, указанными в требованиях заказа.
2. УРОВЕНЬ ОПЫТА - senior > middle > junior. Опыт работы с конкретными технологиями важнее общего опыта.
3. КАЧЕСТВО СОПРОВОДИТЕЛЬНОГО ПИСЬМА - показывает понимание задачи и готовность к работе.
4. ЦЕНА - учитывается только если несколько кандидатов имеют одинаковое соответствие навыкам.

ВАЖНО:
- Приоритет отдавай исполнителю с БОЛЬШИМ количеством соответствующих навыков из требований заказа
- Если исполнитель упоминает в письме конкретные технологии из требований - это большой плюс
- Если у исполнителя есть опыт работы с технологиями из требований (Go, PostgreSQL, API Security и т.д.) - это критически важно
- НЕ выбирай исполнителя только потому что он дешевле, если у него меньше соответствующих навыков
- Если заказ требует бекенд-разработки (Go, PostgreSQL, API), то исполнитель с опытом фронтенда (React, UI/UX) НЕ подходит, даже если он готов учиться

Твоя задача: выбрать ОДНОГО лучшего исполнителя на основе СООТВЕТСТВИЯ НАВЫКОВ требованиям заказа.

Верни ответ в следующем формате:
ID_ОТКЛИКА: [UUID лучшего отклика]
ОБОСНОВАНИЕ: [Краткое обоснование выбора (до 3-4 предложений), объясняющее почему именно этот исполнитель лучше всего подходит. Обязательно укажи какие конкретные навыки из требований заказа соответствуют навыкам исполнителя.]

Важно: Выбери только ОДНОГО исполнителя и дай четкое обоснование с указанием конкретных соответствующих навыков.`,
		order.Title,
		order.Description,
		requirementsStr,
		budgetStr,
		proposalsInfo)

	messages := []map[string]string{
		{"role": "system", "content": "Ты эксперт по анализу технических навыков фрилансеров. Твоя задача - объективно выбрать лучшего исполнителя на основе соответствия его навыков требованиям заказа. Приоритет: соответствие навыков > уровень опыта > качество письма > цена. Отвечай кратко и по делу, указывая конкретные соответствующие навыки."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, "", err
	}

	// Парсим ответ, чтобы извлечь ID отклика и обоснование
	response = strings.TrimSpace(response)
	bestProposalID, justification := parseBestProposalResponse(response, proposals)

	if bestProposalID == nil {
		// Если не удалось распарсить, выбираем лучшего по соответствию навыков
		bestProposalID, justification = selectBestBySkills(proposals, freelancerProfiles, requirements, order.Title)
		if bestProposalID == nil {
			// Если все равно не выбрали, берем первого
			bestProposalID = &proposals[0].ID
			justification = fallbackBestProposalJustification(order.Title, proposals[0], freelancerProfiles[proposals[0].FreelancerID])
		}
	}

	return bestProposalID, justification, nil
}

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

// GenerateOrderDescription помогает создать описание заказа на основе краткого описания.
func (c *Client) GenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string) (string, error) {
	skillsStr := formatSkillsStr(skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:] // Убираем "\n" в начале
	}

	prompt := fmt.Sprintf(`Помоги создать профессиональное и подробное описание заказа для фриланс-платформы.

Заголовок: %s
Краткое описание: %s
%s

Создай подробное описание заказа (3-5 предложений), которое:
- Четко описывает задачу
- Указывает ожидаемый результат
- Помогает исполнителям понять требования
- Профессионально и привлекательно

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, title, briefDescription, skillsStr)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Помогай создавать профессиональные описания заказов. Всегда возвращай только обычный текст без markdown и форматирования."},
		{"role": "user", "content": prompt},
	}

	description, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(description), nil
}

// StreamGenerateOrderDescription создаёт описание заказа потоково через Bothub Responses API.
func (c *Client) StreamGenerateOrderDescription(
	ctx context.Context,
	title, briefDescription string,
	skills []string,
	onDelta func(chunk string) error,
) error {
	skillsStr := formatSkillsStr(skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:] // Убираем "\n" в начале
	}

	prompt := fmt.Sprintf(`Помоги создать профессиональное и подробное описание заказа для фриланс-платформы.

Заголовок: %s
Краткое описание: %s
%s

Создай подробное описание заказа (3-5 предложений), которое:
- Четко описывает задачу
- Указывает ожидаемый результат
- Помогает исполнителям понять требования
- Профессионально и привлекательно

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, title, briefDescription, skillsStr)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// GenerateProposal помогает создать отклик на заказ.
// portfolioItems должны соответствовать models.PortfolioItemForAI по структуре
func (c *Client) GenerateProposal(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, userSkills []string, userExperience string, portfolioItems interface{}) (string, error) {
	// Преобразуем portfolioItems в нужный формат
	items := normalizePortfolioItems(portfolioItems)

	skillsStr := formatSkillsStrForProposal(userSkills)
	experienceStr := formatExperienceStr(userExperience)
	requirementsStr := formatRequirementsStrSimple(requirements)
	portfolioStr := formatPortfolioStr(items, "\n\nМои работы из портфолио:\n")

	prompt := fmt.Sprintf(`Помоги создать профессиональный, но честный отклик на заказ для фриланс-платформы.

Заказ: %s
Описание заказа: %s%s%s%s%s

Создай отклик (3-4 предложения), который:
- Показывает понимание задачи и требований заказа
- Подчеркивает только тот опыт и навыки исполнителя, которые явно указаны в разделе "Мои навыки" и "Мой опыт и описание" (user_skills, user_experience, user_bio)
- НЕ придумывает несуществующий опыт и технологии: если в навыках/описании исполнителя нет упоминания Gin, Fiber, Docker, CI/CD и т.п., НЕ утверждай, что он с ними работал
- Упоминает конкретные работы из портфолио только если они есть в данных портфолио
- Может честно указать, что исполнитель готов работать с указанным в заказе стеком и изучать недостающие технологии, если они ему интересны
- Профессиональный и убедительный, но без ложных утверждений

Если профайл исполнителя практически пустой, сделай упор на мотивацию, готовность разобраться в задаче и задавай уточняющие вопросы, не придумывая опыт.

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, order.Title, order.Description, requirementsStr, skillsStr, experienceStr, portfolioStr)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Помогай создавать профессиональные отклики на заказы. Всегда возвращай только обычный текст без markdown и форматирования."},
		{"role": "user", "content": prompt},
	}

	proposal, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(proposal), nil
}

// StreamGenerateProposal создаёт отклик на заказ потоково через Bothub Responses API.
// По аналогии с GenerateProposal, но результат приходит чанками в onDelta.
func (c *Client) StreamGenerateProposal(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	userSkills []string,
	userExperience string,
	portfolioItems interface{},
	onDelta func(chunk string) error,
) error {
	// Преобразуем portfolioItems в срез удобной структуры (та же логика, что в GenerateProposal).
	items := normalizePortfolioItems(portfolioItems)

	skillsStr := formatSkillsStrForProposal(userSkills)
	experienceStr := formatExperienceStr(userExperience)
	requirementsStr := formatRequirementsStrSimple(requirements)
	portfolioStr := formatPortfolioStr(items, "\n\nМои работы из портфолио:\n")

	prompt := fmt.Sprintf(`Помоги создать профессиональный, но честный отклик на заказ для фриланс-платформы.

Заказ: %s
Описание заказа: %s%s%s%s%s

Создай отклик (3-4 предложения), который:
- Показывает понимание задачи и требований заказа
- Подчеркивает только тот опыт и навыки исполнителя, которые явно указаны в разделе "Мои навыки" и "Мой опыт и описание" (user_skills, user_experience, user_bio)
- НЕ придумывает несуществующий опыт и технологии: если в навыках/описании исполнителя нет упоминания Gin, Fiber, Docker, CI/CD и т.п., НЕ утверждай, что он с ними работал
- Упоминает конкретные работы из портфолио только если они есть в данных портфолио
- Может честно указать, что исполнитель готов работать с указанным в заказе стеком и изучать недостающие технологии, если они ему интересны
- Профессиональный и убедительный, но без ложных утверждений

Если профайл исполнителя практически пустой, сделай упор на мотивацию, готовность разобраться в задаче и задавай уточняющие вопросы, не придумывая опыт.

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, order.Title, order.Description, requirementsStr, skillsStr, experienceStr, portfolioStr)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// ImproveOrderDescription улучшает существующее описание заказа.
func (c *Client) ImproveOrderDescription(ctx context.Context, title, description string) (string, error) {
	prompt := fmt.Sprintf(`Улучши описание заказа, сделав его более профессиональным и привлекательным для исполнителей.

Заголовок: %s
Текущее описание: %s

Улучшенное описание должно:
- Быть более структурированным
- Четко описывать задачу и ожидаемый результат
- Быть профессиональным и привлекательным
- Сохранять основную суть

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, title, description)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Помогай улучшать описания заказов. Всегда возвращай только обычный текст без markdown и форматирования."},
		{"role": "user", "content": prompt},
	}

	improved, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(improved), nil
}

// StreamImproveOrderDescription улучшает описание заказа потоково через Bothub Responses API.
func (c *Client) StreamImproveOrderDescription(
	ctx context.Context,
	title, description string,
	onDelta func(chunk string) error,
) error {
	prompt := fmt.Sprintf(`Улучши описание заказа, сделав его более профессиональным и привлекательным для исполнителей.

Заголовок: %s
Текущее описание: %s

Улучшенное описание должно:
- Быть более структурированным
- Четко описывать задачу и ожидаемый результат
- Быть профессиональным и привлекательным
- Сохранять основную суть

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, title, description)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// SummarizeConversation создаёт резюме переписки в чате.
func (c *Client) SummarizeConversation(ctx context.Context, messages []models.Message, orderTitle string) (*models.ChatSummary, error) {
	if len(messages) == 0 {
		return &models.ChatSummary{
			Summary:       "Переписка пуста",
			NextSteps:     []string{},
			Agreements:    []string{},
			OpenQuestions: []string{},
		}, nil
	}

	// Формируем текст переписки
	conversationText := formatConversationText(messages)

	prompt := fmt.Sprintf(`Проанализируй переписку по заказу и создай структурированное резюме.

Заказ: %s

Переписка:
%s

Верни ответ в формате JSON:
{
  "summary": "Краткое резюме переписки (2-3 предложения)",
  "next_steps": ["Шаг 1", "Шаг 2"],
  "agreements": ["Что согласовано 1", "Что согласовано 2"],
  "open_questions": ["Открытый вопрос 1", "Открытый вопрос 2"]
}

Верни только валидный JSON без дополнительных комментариев.`, orderTitle, conversationText)

	messagesAI := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Анализируй переписки и создавай структурированные резюме. Всегда отвечай валидным JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messagesAI)
	if err != nil {
		return nil, err
	}

	// Парсим JSON ответ
	var summary models.ChatSummary
	if err := json.Unmarshal([]byte(response), &summary); err != nil {
		// Fallback: создаём простое резюме
		return &models.ChatSummary{
			Summary:       strings.TrimSpace(response),
			NextSteps:     []string{},
			Agreements:    []string{},
			OpenQuestions: []string{},
		}, nil
	}

	return &summary, nil
}

// StreamSummarizeConversation создаёт резюме переписки потоково.
func (c *Client) StreamSummarizeConversation(
	ctx context.Context,
	messages []models.Message,
	orderTitle string,
	onDelta func(chunk string) error,
) error {
	if len(messages) == 0 {
		return onDelta("Переписка пуста")
	}

	conversationText := formatConversationText(messages)

	prompt := fmt.Sprintf(`Проанализируй переписку по заказу и создай краткое резюме (2-3 предложения).

Заказ: %s

Переписка:
%s

Создай краткое резюме переписки, выделив основные моменты и договорённости.`, orderTitle, conversationText)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// RecommendRelevantOrders анализирует профиль фрилансера и рекомендует подходящие заказы.
func (c *Client) RecommendRelevantOrders(
	ctx context.Context,
	freelancerProfile *models.Profile,
	portfolioItems []models.PortfolioItemForAI,
	orders []models.Order,
) ([]models.RecommendedOrder, string, error) {
	if len(orders) == 0 {
		return []models.RecommendedOrder{}, "", nil
	}

	// Формируем информацию о фрилансере
	skillsStr := formatSkillsStr(freelancerProfile.Skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:] // Убираем "\n" в начале
	}

	experienceStr := formatProfileInfo(freelancerProfile)

	// Формируем информацию о портфолио
	items := normalizePortfolioItems(portfolioItems)
	portfolioStr := formatPortfolioStr(items, "\n\nРаботы из портфолио:\n")

	// Формируем список заказов
	ordersInfo := formatOrdersInfo(orders)

	prompt := fmt.Sprintf(`Ты помощник для фрилансера на фриланс-платформе. Проанализируй профиль фрилансера и список заказов, затем выбери заказы, которые подходят на 70%% или более.

Профиль фрилансера:
%s%s%s

Доступные заказы:%s

Верни ответ в формате JSON:
{
  "recommended_orders": [
    {
      "order_id": "uuid1",
      "match_score": 9.5,
      "explanation": "Краткое объяснение почему этот заказ подходит (1-2 предложения)"
    },
    {
      "order_id": "uuid2",
      "match_score": 8.7,
      "explanation": "Краткое объяснение почему этот заказ подходит (1-2 предложения)"
    }
  ],
  "explanation": "Общее объяснение почему эти заказы подходят (2-3 предложения)"
}

КРИТИЧЕСКИ ВАЖНО: 
- Выбери заказы с match_score >= 7.0 (70%% совпадения или выше)
- В первую очередь выбирай заказы с match_score >= 8.0 (80%%+)
- Если есть заказы с 70-79%%, которые хорошо подходят, можешь их включить
- Верни самые подходящие заказы, максимум 10 штук
- Отсортируй по убыванию match_score (самые подходящие первые)
- Если есть хотя бы 1 заказ с match_score >= 7.0, обязательно верни его
- Если подходящих заказов нет (все меньше 70%%), верни пустой массив recommended_orders: []
- Для каждого заказа укажи explanation - почему именно этот заказ подходит (1-2 предложения)
- Верни только валидный JSON`, skillsStr, experienceStr, portfolioStr, ordersInfo)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Анализируй профили и заказы, рекомендую наиболее подходящие. Всегда отвечай валидным JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, "", err
	}

	// Парсим JSON
	var result struct {
		RecommendedOrders []struct {
			OrderID     string  `json:"order_id"`
			MatchScore  float64 `json:"match_score"`
			Explanation string  `json:"explanation"`
		} `json:"recommended_orders"`
		Explanation string `json:"explanation"`
	}

	// Пробуем новый формат с match_score
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Если новый формат не подошел, возвращаем пустой список
		// (без match_score мы не можем определить подходящие заказы)
		return []models.RecommendedOrder{}, "Не удалось проанализировать заказы. Попробуйте позже.", nil
	}

	// Фильтруем заказы по match_score >= 7.0 (70%) и сортируем
	// Сначала берем самые подходящие (8.0+), затем хорошие (7.0-7.9)
	const minMatchScore = 7.0
	type orderWithScore struct {
		ID          uuid.UUID
		Score       float64
		Explanation string
	}
	ordersWithScores := make([]orderWithScore, 0, len(result.RecommendedOrders))
	for _, rec := range result.RecommendedOrders {
		if rec.MatchScore >= minMatchScore {
			if id, err := uuid.Parse(rec.OrderID); err == nil {
				ordersWithScores = append(ordersWithScores, orderWithScore{
					ID:          id,
					Score:       rec.MatchScore,
					Explanation: rec.Explanation,
				})
			}
		}
	}

	// Сортируем по убыванию match_score
	for i := 0; i < len(ordersWithScores)-1; i++ {
		for j := i + 1; j < len(ordersWithScores); j++ {
			if ordersWithScores[i].Score < ordersWithScores[j].Score {
				ordersWithScores[i], ordersWithScores[j] = ordersWithScores[j], ordersWithScores[i]
			}
		}
	}

	// Берем только топ заказы (максимум 10, но только те что >= 7.0)
	const maxRecommended = 10
	recommendedOrders := make([]models.RecommendedOrder, 0, maxRecommended)
	for i, order := range ordersWithScores {
		if i >= maxRecommended {
			break
		}
		recommendedOrders = append(recommendedOrders, models.RecommendedOrder{
			OrderID:     order.ID,
			MatchScore:  order.Score,
			Explanation: order.Explanation,
		})
	}

	return recommendedOrders, result.Explanation, nil
}

// RecommendPriceAndTimeline анализирует заказ и рекомендует цену и сроки.
func (c *Client) RecommendPriceAndTimeline(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	freelancerProfile *models.Profile,
	otherProposals []*models.Proposal,
) (*models.PriceTimelineRecommendation, error) {
	// Формируем информацию о заказе
	budgetStr := formatBudgetStr(order.BudgetMin, order.BudgetMax)
	requirementsStr := formatRequirementsStr(requirements)

	// Информация о фрилансере
	hourlyRateStr := ""
	if freelancerProfile.HourlyRate != nil {
		hourlyRateStr = fmt.Sprintf("\nСтавка фрилансера за час: $%.2f", *freelancerProfile.HourlyRate)
	}

	// Информация о других предложениях
	otherPricesStr := formatOtherPricesStr(otherProposals)

	prompt := fmt.Sprintf(`Ты помощник для фрилансера. Проанализируй заказ и рекомендую подходящую цену и сроки выполнения.

Заказ: %s
Описание: %s%s%s%s%s

Верни ответ в формате JSON:
{
  "recommended_amount": 1000.0,
  "min_amount": 800.0,
  "max_amount": 1200.0,
  "recommended_days": 14,
  "min_days": 10,
  "max_days": 20,
  "explanation": "Краткое объяснение рекомендации (2-3 предложения)"
}

Учти бюджет заказа, сложность задачи, ставку фрилансера и цены других откликов. Верни только валидный JSON.`,
		order.Title, order.Description, budgetStr, requirementsStr, hourlyRateStr, otherPricesStr)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Рекомендую цены и сроки на основе анализа заказов. Всегда отвечай валидным JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	var recommendation models.PriceTimelineRecommendation
	if err := json.Unmarshal([]byte(response), &recommendation); err != nil {
		// Fallback
		recommendedAmount := 0.0
		if order.BudgetMin != nil && order.BudgetMax != nil {
			recommendedAmount = (*order.BudgetMin + *order.BudgetMax) / 2
		}
		return &models.PriceTimelineRecommendation{
			RecommendedAmount: &recommendedAmount,
			Explanation:       "Рекомендуется указать цену в пределах бюджета заказа",
		}, nil
	}

	return &recommendation, nil
}

// ImproveProfile улучшает описание профиля пользователя.
func (c *Client) ImproveProfile(ctx context.Context, currentBio string, skills []string, experienceLevel string) (string, error) {
	skillsStr := formatSkillsStr(skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:] // Убираем "\n" в начале
	}

	prompt := fmt.Sprintf(`Улучши описание профиля фрилансера, сделав его более профессиональным и привлекательным.

Текущее описание: %s
%s
Уровень опыта: %s

Улучшенное описание должно:
- Быть профессиональным и структурированным
- Подчеркивать ключевые навыки и опыт
- Быть привлекательным для потенциальных клиентов
- Сохранять основную суть

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, currentBio, skillsStr, experienceLevel)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Помогай улучшать описания профилей. Всегда возвращай только обычный текст без markdown и форматирования."},
		{"role": "user", "content": prompt},
	}

	improved, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(improved), nil
}

// StreamImproveProfile улучшает описание профиля потоково через Bothub Responses API.
func (c *Client) StreamImproveProfile(
	ctx context.Context,
	currentBio string,
	skills []string,
	experienceLevel string,
	onDelta func(chunk string) error,
) error {
	skillsStr := formatSkillsStr(skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:] // Убираем "\n" в начале
	}

	prompt := fmt.Sprintf(`Улучши описание профиля фрилансера, сделав его более профессиональным и привлекательным.

Текущее описание: %s
%s
Уровень опыта: %s

Улучшенное описание должно:
- Быть профессиональным и структурированным
- Подчеркивать ключевые навыки и опыт
- Быть привлекательным для потенциальных клиентов
- Сохранять основную суть

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, currentBio, skillsStr, experienceLevel)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// ImprovePortfolioItem улучшает описание работы в портфолио.
func (c *Client) ImprovePortfolioItem(ctx context.Context, title, description string, aiTags []string) (string, error) {
	tagsStr := formatTagsStr(aiTags)

	prompt := fmt.Sprintf(`Улучши описание работы в портфолио, сделав его более профессиональным и привлекательным.

Название: %s
Текущее описание: %s
%s

Улучшенное описание должно:
- Четко описывать задачу и результат
- Подчеркивать использованные технологии и навыки
- Быть структурированным и читаемым
- Сохранять основную суть

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, title, description, tagsStr)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Помогай улучшать описания работ в портфолио. Всегда возвращай только обычный текст без markdown и форматирования."},
		{"role": "user", "content": prompt},
	}

	improved, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(improved), nil
}

// StreamImprovePortfolioItem улучшает описание работы в портфолио потоково через Bothub Responses API.
func (c *Client) StreamImprovePortfolioItem(
	ctx context.Context,
	title, description string,
	aiTags []string,
	onDelta func(chunk string) error,
) error {
	tagsStr := formatTagsStr(aiTags)

	prompt := fmt.Sprintf(`Улучши описание работы в портфолио, сделав его более профессиональным и привлекательным.

Название: %s
Текущее описание: %s
%s

Улучшенное описание должно:
- Четко описывать задачу и результат
- Подчеркивать использованные технологии и навыки
- Быть структурированным и читаемым
- Сохранять основную суть

ВАЖНО: Верни только обычный текст без markdown, форматирования, звездочек (*), подчеркиваний (_), решеток (#) и других специальных символов. Только чистый текст с пробелами между словами.`, title, description, tagsStr)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// EvaluateOrderQuality анализирует заказ и оценивает его качество.
func (c *Client) EvaluateOrderQuality(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
) (*models.OrderQualityEvaluation, error) {
	requirementsStr := formatRequirementsStr(requirements)
	budgetStr := formatBudgetStrSimple(order.BudgetMin, order.BudgetMax)
	deadlineStr := formatDeadlineStr(order.DeadlineAt)

	prompt := fmt.Sprintf(`Проанализируй качество заказа и оцени его по шкале от 1 до 10.

Заголовок: %s
Описание: %s%s%s%s

Верни ответ в формате JSON:
{
  "score": 8,
  "strengths": ["Сильная сторона 1", "Сильная сторона 2"],
  "weaknesses": ["Слабая сторона 1", "Слабая сторона 2"],
  "recommendations": ["Рекомендация 1", "Рекомендация 2"]
}

Оцени насколько заказ:
- Четко описывает задачу
- Указывает конкретные требования
- Имеет реалистичный бюджет
- Имеет разумный дедлайн
- Привлекателен для исполнителей

Верни только валидный JSON.`, order.Title, order.Description, requirementsStr, budgetStr, deadlineStr)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Оценивай качество заказов и давай конструктивные рекомендации. Всегда отвечай валидным JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	var evaluation models.OrderQualityEvaluation
	if err := json.Unmarshal([]byte(response), &evaluation); err != nil {
		// Fallback
		return &models.OrderQualityEvaluation{
			Score:           5,
			Strengths:       []string{"Заказ создан"},
			Weaknesses:      []string{"Требуется дополнительная информация"},
			Recommendations: []string{"Добавьте больше деталей в описание"},
		}, nil
	}

	return &evaluation, nil
}

// FindSuitableFreelancers анализирует заказ и находит подходящих фрилансеров.
func (c *Client) FindSuitableFreelancers(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	freelancerProfiles []*models.Profile,
	freelancerPortfolios map[uuid.UUID][]models.PortfolioItemForAI,
) ([]models.SuitableFreelancer, error) {
	if len(freelancerProfiles) == 0 {
		return []models.SuitableFreelancer{}, nil
	}

	// Формируем информацию о заказе
	requirementsStr := formatRequirementsStr(requirements)

	// Формируем информацию о фрилансерах
	freelancersInfo := formatFreelancersInfo(freelancerProfiles, freelancerPortfolios)

	prompt := fmt.Sprintf(`Ты помощник для заказчика. Проанализируй заказ и список фрилансеров, затем выбери ТОП-5 наиболее подходящих исполнителей.

Заказ: %s
Описание: %s%s

Доступные фрилансеры:%s

Верни ответ в формате JSON:
{
  "recommended_freelancers": [
    {
      "user_id": "uuid",
      "match_score": 9.5,
      "explanation": "Краткое объяснение почему подходит (1-2 предложения)"
    }
  ]
}

Выбери максимум 5 фрилансеров, которые лучше всего подходят для заказа. Верни только валидный JSON.`, order.Title, order.Description, requirementsStr, freelancersInfo)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Анализируй заказы и профили, рекомендую наиболее подходящих исполнителей. Всегда отвечай валидным JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	var result struct {
		RecommendedFreelancers []struct {
			UserID      string  `json:"user_id"`
			MatchScore  float64 `json:"match_score"`
			Explanation string  `json:"explanation"`
		} `json:"recommended_freelancers"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: возвращаем первых 5 фрилансеров
		suitable := make([]models.SuitableFreelancer, 0, 5)
		for i := 0; i < len(freelancerProfiles) && i < 5; i++ {
			suitable = append(suitable, models.SuitableFreelancer{
				UserID:      freelancerProfiles[i].UserID,
				MatchScore:  7.0,
				Explanation: "Подходит на основе базовых критериев",
			})
		}
		return suitable, nil
	}

	// Преобразуем в нужный формат
	suitable := make([]models.SuitableFreelancer, 0, len(result.RecommendedFreelancers))
	for _, rec := range result.RecommendedFreelancers {
		if id, err := uuid.Parse(rec.UserID); err == nil {
			suitable = append(suitable, models.SuitableFreelancer{
				UserID:      id,
				MatchScore:  rec.MatchScore,
				Explanation: rec.Explanation,
			})
		}
	}

	return suitable, nil
}

// StreamFindSuitableFreelancers анализирует заказ и находит подходящих фрилансеров потоково.
// Стримит explanation, затем возвращает структурированные данные через onComplete.
func (c *Client) StreamFindSuitableFreelancers(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	freelancerProfiles []*models.Profile,
	freelancerPortfolios map[uuid.UUID][]models.PortfolioItemForAI,
	onDelta func(chunk string) error,
	onComplete func(data []models.SuitableFreelancer) error,
) error {
	if len(freelancerProfiles) == 0 {
		return onComplete([]models.SuitableFreelancer{})
	}

	// Формируем информацию о заказе
	requirementsStr := formatRequirementsStr(requirements)

	// Формируем информацию о фрилансерах
	freelancersInfo := formatFreelancersInfo(freelancerProfiles, freelancerPortfolios)

	prompt := fmt.Sprintf(`Ты помощник для заказчика. Проанализируй заказ и список фрилансеров, затем выбери ТОП-5 наиболее подходящих исполнителей.

Заказ: %s
Описание: %s%s

Доступные фрилансеры:%s

Сначала дай краткое объяснение (2-3 предложения) почему ты выбираешь этих исполнителей. Затем верни ответ в формате JSON:
{
  "recommended_freelancers": [
    {
      "user_id": "uuid",
      "match_score": 9.5,
      "explanation": "Краткое объяснение почему подходит (1-2 предложения)"
    }
  ]
}

Выбери максимум 5 фрилансеров, которые лучше всего подходят для заказа.`, order.Title, order.Description, requirementsStr, freelancersInfo)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	// Стримим explanation
	var fullText strings.Builder
	err := c.streamInput(ctx, input, func(chunk string) error {
		fullText.WriteString(chunk)
		return onDelta(chunk)
	})
	if err != nil {
		return err
	}

	// Парсим JSON из полного ответа
	response := fullText.String()
	var result struct {
		RecommendedFreelancers []struct {
			UserID      string  `json:"user_id"`
			MatchScore  float64 `json:"match_score"`
			Explanation string  `json:"explanation"`
		} `json:"recommended_freelancers"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: возвращаем первых 5 фрилансеров
		suitable := make([]models.SuitableFreelancer, 0, 5)
		for i := 0; i < len(freelancerProfiles) && i < 5; i++ {
			suitable = append(suitable, models.SuitableFreelancer{
				UserID:      freelancerProfiles[i].UserID,
				MatchScore:  7.0,
				Explanation: "Подходит на основе базовых критериев",
			})
		}
		return onComplete(suitable)
	}

	// Преобразуем в нужный формат
	suitable := make([]models.SuitableFreelancer, 0, len(result.RecommendedFreelancers))
	for _, rec := range result.RecommendedFreelancers {
		if id, err := uuid.Parse(rec.UserID); err == nil {
			suitable = append(suitable, models.SuitableFreelancer{
				UserID:      id,
				MatchScore:  rec.MatchScore,
				Explanation: rec.Explanation,
			})
		}
	}

	return onComplete(suitable)
}

// StreamRecommendRelevantOrders анализирует профиль фрилансера и рекомендует подходящие заказы потоково.
// Стримит explanation, затем возвращает структурированные данные через onComplete.
func (c *Client) StreamRecommendRelevantOrders(
	ctx context.Context,
	freelancerProfile *models.Profile,
	portfolioItems []models.PortfolioItemForAI,
	orders []models.Order,
	onDelta func(chunk string) error,
	onComplete func(recommendedOrders []models.RecommendedOrder, generalExplanation string) error,
) error {
	if len(orders) == 0 {
		return onComplete([]models.RecommendedOrder{}, "")
	}

	// Формируем информацию о фрилансере
	skillsStr := formatSkillsStr(freelancerProfile.Skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:] // Убираем "\n" в начале
	}

	experienceStr := formatProfileInfo(freelancerProfile)

	// Формируем информацию о портфолио
	items := normalizePortfolioItems(portfolioItems)
	portfolioStr := formatPortfolioStr(items, "\n\nРаботы из портфолио:\n")

	// Формируем список заказов
	ordersInfo := formatOrdersInfo(orders)

	prompt := fmt.Sprintf(`Ты помощник для фрилансера на фриланс-платформе. Проанализируй профиль фрилансера и список заказов, затем выбери заказы, которые подходят на 70%% или более.

Профиль фрилансера:
%s%s%s

Доступные заказы:%s

Сначала дай краткое объяснение (2-3 предложения) почему ты выбираешь эти заказы. Затем верни ответ в формате JSON:
{
  "recommended_orders": [
    {
      "order_id": "uuid1",
      "match_score": 9.5,
      "explanation": "Краткое объяснение почему этот заказ подходит (1-2 предложения)"
    },
    {
      "order_id": "uuid2",
      "match_score": 8.7,
      "explanation": "Краткое объяснение почему этот заказ подходит (1-2 предложения)"
    }
  ],
  "explanation": "Общее объяснение почему эти заказы подходят (2-3 предложения)"
}

КРИТИЧЕСКИ ВАЖНО: 
- Выбери заказы с match_score >= 7.0 (70%% совпадения или выше)
- В первую очередь выбирай заказы с match_score >= 8.0 (80%%+)
- Если есть заказы с 70-79%%, которые хорошо подходят, можешь их включить
- Верни самые подходящие заказы, МАКСИМУМ 10 штук
- Отсортируй по убыванию match_score (самые подходящие первые)
- Если есть хотя бы 1 заказ с match_score >= 7.0, обязательно верни его
- Если подходящих заказов нет (все меньше 70%%), верни пустой массив recommended_orders: []
- Для каждого заказа укажи explanation - почему именно этот заказ подходит (1-2 предложения)`, skillsStr, experienceStr, portfolioStr, ordersInfo)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	// Стримим explanation
	var fullText strings.Builder
	err := c.streamInput(ctx, input, func(chunk string) error {
		fullText.WriteString(chunk)
		return onDelta(chunk)
	})
	if err != nil {
		return err
	}

	// Парсим JSON из полного ответа
	response := fullText.String()
	var result struct {
		RecommendedOrders []struct {
			OrderID     string  `json:"order_id"`
			MatchScore  float64 `json:"match_score"`
			Explanation string  `json:"explanation"`
		} `json:"recommended_orders"`
		Explanation string `json:"explanation"`
	}

	// Пробуем новый формат с match_score
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Если новый формат не подошел, возвращаем пустой список
		// (без match_score мы не можем определить подходящие заказы)
		return onComplete([]models.RecommendedOrder{}, "Не удалось проанализировать заказы. Попробуйте позже.")
	}

	// Фильтруем заказы по match_score >= 7.0 (70%) и сортируем
	// Сначала берем самые подходящие (8.0+), затем хорошие (7.0-7.9)
	const minMatchScore = 7.0
	type orderWithScore struct {
		ID          uuid.UUID
		Score       float64
		Explanation string
	}
	ordersWithScores := make([]orderWithScore, 0, len(result.RecommendedOrders))
	for _, rec := range result.RecommendedOrders {
		if rec.MatchScore >= minMatchScore {
			if id, err := uuid.Parse(rec.OrderID); err == nil {
				ordersWithScores = append(ordersWithScores, orderWithScore{
					ID:          id,
					Score:       rec.MatchScore,
					Explanation: rec.Explanation,
				})
			}
		}
	}

	// Сортируем по убыванию match_score
	for i := 0; i < len(ordersWithScores)-1; i++ {
		for j := i + 1; j < len(ordersWithScores); j++ {
			if ordersWithScores[i].Score < ordersWithScores[j].Score {
				ordersWithScores[i], ordersWithScores[j] = ordersWithScores[j], ordersWithScores[i]
			}
		}
	}

	// Берем только топ заказы (максимум 10, но только те что >= 7.0)
	// Показываем только самые подходящие заказы с процентом совместимости от 70%
	const maxRecommended = 10
	recommendedOrders := make([]models.RecommendedOrder, 0, maxRecommended)
	for i, order := range ordersWithScores {
		if i >= maxRecommended {
			break
		}
		recommendedOrders = append(recommendedOrders, models.RecommendedOrder{
			OrderID:     order.ID,
			MatchScore:  order.Score,
			Explanation: order.Explanation,
		})
	}

	return onComplete(recommendedOrders, result.Explanation)
}

// StreamEvaluateOrderQuality анализирует заказ и оценивает его качество потоково.
// Стримит explanation, затем возвращает структурированные данные через onComplete.
func (c *Client) StreamEvaluateOrderQuality(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	onDelta func(chunk string) error,
	onComplete func(evaluation *models.OrderQualityEvaluation) error,
) error {
	requirementsStr := formatRequirementsStr(requirements)
	budgetStr := formatBudgetStrSimple(order.BudgetMin, order.BudgetMax)
	deadlineStr := formatDeadlineStr(order.DeadlineAt)

	prompt := fmt.Sprintf(`Проанализируй качество заказа и оцени его по шкале от 1 до 10.

Заголовок: %s
Описание: %s%s%s%s

Сначала дай краткое объяснение (2-3 предложения) своей оценки. Затем верни ответ в формате JSON:
{
  "score": 8,
  "strengths": ["Сильная сторона 1", "Сильная сторона 2"],
  "weaknesses": ["Слабая сторона 1", "Слабая сторона 2"],
  "recommendations": ["Рекомендация 1", "Рекомендация 2"]
}

Оцени насколько заказ:
- Четко описывает задачу
- Указывает конкретные требования
- Имеет реалистичный бюджет
- Имеет разумный дедлайн
- Привлекателен для исполнителей`, order.Title, order.Description, requirementsStr, budgetStr, deadlineStr)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	// Стримим explanation
	var fullText strings.Builder
	err := c.streamInput(ctx, input, func(chunk string) error {
		fullText.WriteString(chunk)
		return onDelta(chunk)
	})
	if err != nil {
		return err
	}

	// Парсим JSON из полного ответа
	response := fullText.String()
	var evaluation models.OrderQualityEvaluation
	if err := json.Unmarshal([]byte(response), &evaluation); err != nil {
		// Fallback
		evaluation = models.OrderQualityEvaluation{
			Score:           5,
			Strengths:       []string{"Заказ создан"},
			Weaknesses:      []string{"Требуется дополнительная информация"},
			Recommendations: []string{"Добавьте больше деталей в описание"},
		}
	}

	return onComplete(&evaluation)
}

// StreamRecommendPriceAndTimeline анализирует заказ и рекомендует цену и сроки потоково.
// Стримит explanation, затем возвращает структурированные данные через onComplete.
func (c *Client) StreamRecommendPriceAndTimeline(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	freelancerProfile *models.Profile,
	otherProposals []*models.Proposal,
	onDelta func(chunk string) error,
	onComplete func(recommendation *models.PriceTimelineRecommendation) error,
) error {
	// Формируем информацию о заказе
	budgetStr := formatBudgetStr(order.BudgetMin, order.BudgetMax)
	requirementsStr := formatRequirementsStr(requirements)

	// Информация о фрилансере
	hourlyRateStr := ""
	if freelancerProfile.HourlyRate != nil {
		hourlyRateStr = fmt.Sprintf("\nСтавка фрилансера за час: $%.2f", *freelancerProfile.HourlyRate)
	}

	// Информация о других предложениях
	otherPricesStr := formatOtherPricesStr(otherProposals)

	prompt := fmt.Sprintf(`Ты помощник для фрилансера. Проанализируй заказ и рекомендую подходящую цену и сроки выполнения.

Заказ: %s
Описание: %s%s%s%s%s

Сначала дай краткое объяснение (2-3 предложения) своих рекомендаций. Затем верни ответ в формате JSON:
{
  "recommended_amount": 1500.00,
  "min_amount": 1200.00,
  "max_amount": 1800.00,
  "recommended_days": 7,
  "min_days": 5,
  "max_days": 10,
  "explanation": "Краткое объяснение рекомендаций"
}

Учти:
- Сложность задачи
- Требуемые навыки и уровень
- Бюджет заказа
- Ставку фрилансера
- Цены других откликов (если есть)`, order.Title, order.Description, requirementsStr, budgetStr, hourlyRateStr, otherPricesStr)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	// Стримим explanation
	var fullText strings.Builder
	err := c.streamInput(ctx, input, func(chunk string) error {
		fullText.WriteString(chunk)
		return onDelta(chunk)
	})
	if err != nil {
		return err
	}

	// Парсим JSON из полного ответа
	response := fullText.String()
	var recommendation models.PriceTimelineRecommendation
	if err := json.Unmarshal([]byte(response), &recommendation); err != nil {
		// Fallback
		recommendation = models.PriceTimelineRecommendation{
			Explanation: "Рекомендация на основе базовых критериев",
		}
	}

	return onComplete(&recommendation)
}

// AIChatAssistant обрабатывает запросы к AI помощнику.
func (c *Client) AIChatAssistant(
	ctx context.Context,
	userMessage string,
	userRole string,
	contextData map[string]interface{},
) (string, error) {
	// Формируем контекст
	contextStr := ""
	if len(contextData) > 0 {
		contextStr = "\n\nКонтекст пользователя:\n"
		for key, value := range contextData {
			contextStr += fmt.Sprintf("- %s: %v\n", key, value)
		}
	}

	systemPrompt := "Ты помощник для фриланс-платформы. Помогай пользователям с вопросами о платформе, создании заказов, откликах и работе на платформе. Отвечай кратко и по делу."
	if userRole == "client" {
		systemPrompt += " Пользователь - заказчик. Помогай с созданием заказов, выбором исполнителей и управлением проектами."
	} else if userRole == "freelancer" {
		systemPrompt += " Пользователь - фрилансер. Помогай с поиском заказов, созданием откликов и управлением портфолио."
	}

	prompt := fmt.Sprintf(`%s%s

Вопрос пользователя: %s`, systemPrompt, contextStr, userMessage)

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

// StreamAIChatAssistant обрабатывает запросы к AI помощнику потоково.
func (c *Client) StreamAIChatAssistant(
	ctx context.Context,
	userMessage string,
	userRole string,
	contextData map[string]interface{},
	onDelta func(chunk string) error,
) error {
	contextStr := ""
	if len(contextData) > 0 {
		contextStr = "\n\nКонтекст пользователя:\n"
		for key, value := range contextData {
			contextStr += fmt.Sprintf("- %s: %v\n", key, value)
		}
	}

	systemPrompt := "Ты помощник для фриланс-платформы. Помогай пользователям с вопросами о платформе."
	if userRole == "client" {
		systemPrompt += " Пользователь - заказчик."
	} else if userRole == "freelancer" {
		systemPrompt += " Пользователь - фрилансер."
	}

	prompt := fmt.Sprintf(`%s%s

Вопрос пользователя: %s`, systemPrompt, contextStr, userMessage)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// chatCompletion выполняет запрос к OpenAI-совместимому API.
func (c *Client) chatCompletion(ctx context.Context, messages []map[string]string) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("ai: baseURL не задан")
	}

	payload := map[string]any{
		"model":    c.model,
		"messages": messages,
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

// GenerateOrderSuggestions генерирует предложения для создания заказа (навыки, бюджет, сроки и т.д.)
func (c *Client) GenerateOrderSuggestions(ctx context.Context, title, description string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Ты - AI помощник для создания заказов на фриланс-платформе.

На основе заказа:
Название: "%s"
Описание: "%s"

Проанализируй заказ и предложи оптимальные значения для:
1. Навыки (skills) - список технологий/инструментов, которые нужны для выполнения заказа (массив строк, минимум 2-3 навыка)
2. Бюджет (budget_min и budget_max) - минимальная и максимальная стоимость в рублях (числа)
3. Срок (deadline_days) - количество дней на выполнение от сегодня (число)
4. Файлы (needs_attachments) - нужны ли прикрепленные файлы (boolean)
5. Описание файлов (attachment_description) - зачем нужны файлы (строка, если needs_attachments = true)

КРИТИЧЕСКИ ВАЖНО: 
- Ответь ТОЛЬКО валидным JSON объектом
- НЕ добавляй никакого текста до или после JSON
- НЕ используй markdown код блоки
- JSON должен начинаться с { и заканчиваться }
- Все поля обязательны, используй пустые значения если не уверен

Пример правильного ответа:
{
  "skills": ["Vue.js", "TypeScript", "Node.js"],
  "budget_min": 50000,
  "budget_max": 100000,
  "deadline_days": 30,
  "needs_attachments": true,
  "attachment_description": "Рекомендуется прикрепить примеры дизайна или техническое задание"
}`, title, description)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Анализируй заказы и предлагай оптимальные значения для создания. Всегда отвечай валидным JSON без дополнительного текста."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Парсим JSON ответ
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: пытаемся извлечь JSON из текста
		result = parseJSONFromText(response)
	}

	return result, nil
}

// StreamGenerateOrderSuggestions генерирует предложения для создания заказа потоково
func (c *Client) StreamGenerateOrderSuggestions(
	ctx context.Context,
	title, description string,
	onDelta func(chunk string) error,
) error {
	prompt := fmt.Sprintf(`Ты - AI помощник для создания заказов на фриланс-платформе.

На основе заказа:
Название: "%s"
Описание: "%s"

Проанализируй заказ и предложи оптимальные значения для:
1. Навыки (skills) - список технологий/инструментов, которые нужны для выполнения заказа (массив строк, минимум 2-3 навыка)
2. Бюджет (budget_min и budget_max) - минимальная и максимальная стоимость в рублях (числа)
3. Срок (deadline_days) - количество дней на выполнение от сегодня (число)
4. Файлы (needs_attachments) - нужны ли прикрепленные файлы (boolean)
5. Описание файлов (attachment_description) - зачем нужны файлы (строка, если needs_attachments = true)

КРИТИЧЕСКИ ВАЖНО: 
- Ответь ТОЛЬКО валидным JSON объектом
- НЕ добавляй никакого текста до или после JSON
- НЕ используй markdown код блоки
- JSON должен начинаться с { и заканчиваться }
- Все поля обязательны, используй пустые значения если не уверен

Пример правильного ответа:
{
  "skills": ["Vue.js", "TypeScript", "Node.js"],
  "budget_min": 50000,
  "budget_max": 100000,
  "deadline_days": 30,
  "needs_attachments": true,
  "attachment_description": "Рекомендуется прикрепить примеры дизайна или техническое задание"
}`, title, description)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// GenerateOrderSkills генерирует список навыков для заказа
func (c *Client) GenerateOrderSkills(ctx context.Context, title, description string) ([]string, error) {
	prompt := fmt.Sprintf(`Ты - AI помощник для создания заказов на фриланс-платформе.

На основе заказа:
Название: "%s"
Описание: "%s"

Определи необходимые навыки и технологии (массив строк).

ВАЖНО: Ответь ТОЛЬКО валидным JSON массивом без дополнительного текста:
["React", "TypeScript", "Node.js"]`, title, description)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Анализируй заказы и определяй необходимые навыки. Всегда отвечай валидным JSON массивом без дополнительного текста."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Парсим JSON ответ
	var skills []string
	if err := json.Unmarshal([]byte(response), &skills); err != nil {
		// Пытаемся распарсить объект с полем skills
		var result map[string]interface{}
		if err2 := json.Unmarshal([]byte(response), &result); err2 == nil {
			if skillsVal, ok := result["skills"].([]interface{}); ok {
				skills = make([]string, len(skillsVal))
				for i, v := range skillsVal {
					if str, ok := v.(string); ok {
						skills[i] = str
					}
				}
			}
		}
	}

	return skills, nil
}

// StreamGenerateOrderSkills генерирует список навыков для заказа потоково
func (c *Client) StreamGenerateOrderSkills(
	ctx context.Context,
	title, description string,
	onDelta func(chunk string) error,
) error {
	prompt := fmt.Sprintf(`Ты - AI помощник для создания заказов на фриланс-платформе.

На основе заказа:
Название: "%s"
Описание: "%s"

Определи необходимые навыки и технологии (массив строк).

ВАЖНО: Ответь ТОЛЬКО валидным JSON массивом без дополнительного текста:
["React", "TypeScript", "Node.js"]`, title, description)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// GenerateOrderBudget генерирует предложение бюджета для заказа
func (c *Client) GenerateOrderBudget(ctx context.Context, title, description string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Ты - AI помощник для создания заказов на фриланс-платформе.

На основе заказа:
Название: "%s"
Описание: "%s"

Определи оптимальный бюджет в рублях (минимальная и максимальная стоимость).

ВАЖНО: Ответь ТОЛЬКО валидным JSON без дополнительного текста:
{
  "budget_min": 50000,
  "budget_max": 100000
}`, title, description)

	messages := []map[string]string{
		{"role": "system", "content": "Ты помощник для фриланс-платформы. Анализируй заказы и предлагай оптимальный бюджет. Всегда отвечай валидным JSON без дополнительного текста."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Парсим JSON ответ
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: пытаемся извлечь JSON из текста
		result = parseJSONFromText(response)
	}

	return result, nil
}

// StreamGenerateOrderBudget генерирует предложение бюджета для заказа потоково
func (c *Client) StreamGenerateOrderBudget(
	ctx context.Context,
	title, description string,
	onDelta func(chunk string) error,
) error {
	prompt := fmt.Sprintf(`Ты - AI помощник для создания заказов на фриланс-платформе.

На основе заказа:
Название: "%s"
Описание: "%s"

Определи оптимальный бюджет в рублях (минимальная и максимальная стоимость).

ВАЖНО: Ответь ТОЛЬКО валидным JSON без дополнительного текста:
{
  "budget_min": 50000,
  "budget_max": 100000
}`, title, description)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
}

// GenerateWelcomeMessage генерирует приветственное сообщение для нового пользователя
func (c *Client) GenerateWelcomeMessage(ctx context.Context, userRole string) (string, error) {
	roleText := "заказчика"
	if userRole == "freelancer" {
		roleText = "фрилансера"
	}

	prompt := fmt.Sprintf(`Привет! Я AI-помощник. Помоги новому %s начать работу на платформе. Дай краткое приветствие (2-3 предложения) и объясни, как я могу помочь.`, roleText)

	systemPrompt := "Ты помощник для фриланс-платформы. Создавай дружелюбные и полезные приветственные сообщения для новых пользователей."
	if userRole == "client" {
		systemPrompt += " Пользователь - заказчик. Расскажи о возможностях создания заказов и поиска исполнителей."
	} else if userRole == "freelancer" {
		systemPrompt += " Пользователь - фрилансер. Расскажи о возможностях поиска заказов и создания портфолио."
	}

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

// StreamGenerateWelcomeMessage генерирует приветственное сообщение потоково
func (c *Client) StreamGenerateWelcomeMessage(
	ctx context.Context,
	userRole string,
	onDelta func(chunk string) error,
) error {
	roleText := "заказчика"
	if userRole == "freelancer" {
		roleText = "фрилансера"
	}

	prompt := fmt.Sprintf(`Привет! Я AI-помощник. Помоги новому %s начать работу на платформе. Дай краткое приветствие (2-3 предложения) и объясни, как я могу помочь.`, roleText)

	input := []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": prompt,
				},
			},
		},
	}

	return c.streamInput(ctx, input, onDelta)
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
