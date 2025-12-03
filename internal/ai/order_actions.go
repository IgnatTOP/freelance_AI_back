package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

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

func (c *Client) ImproveOrderDescription(ctx context.Context, title, description string) (string, error) {
	prompt := fmt.Sprintf(`Улучши описание заказа (структурированно, профессионально, без markdown):

Заголовок: %s
Описание: %s`, title, description)

	messages := []map[string]string{
		{"role": "system", "content": "Улучшай описания заказов. Возвращай только чистый текст без markdown."},
		{"role": "user", "content": prompt},
	}

	improved, err := c.chatCompletionWithOptions(ctx, messages, 800, 0.7)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(improved), nil
}

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

func (c *Client) RecommendRelevantOrders(
	ctx context.Context,
	freelancerProfile *models.Profile,
	portfolioItems []models.PortfolioItemForAI,
	orders []models.Order,
) ([]models.RecommendedOrder, string, error) {
	if len(orders) == 0 {
		return []models.RecommendedOrder{}, "", nil
	}

	skillsStr := formatSkillsStr(freelancerProfile.Skills)
	if skillsStr != "" {
		skillsStr = skillsStr[2:]
	}
	experienceStr := formatProfileInfo(freelancerProfile)
	items := normalizePortfolioItems(portfolioItems)
	portfolioStr := formatPortfolioStr(items, "\nПортфолио:\n")
	ordersInfo := formatOrdersInfo(orders)

	prompt := fmt.Sprintf(`Выбери заказы (match_score>=7.0) для фрилансера:
%s%s%s

Заказы:%s

JSON: {"recommended_orders":[{"order_id":"uuid","match_score":9.5,"explanation":"причина"}],"explanation":"общее"}`,
		skillsStr, experienceStr, portfolioStr, ordersInfo)

	messages := []map[string]string{
		{"role": "system", "content": "Рекомендуй заказы. Отвечай только JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletionWithOptions(ctx, messages, 512, 0.5)
	if err != nil {
		return nil, "", err
	}

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

func (c *Client) RecommendPriceAndTimeline(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
	freelancerProfile *models.Profile,
	otherProposals []*models.Proposal,
) (*models.PriceTimelineRecommendation, error) {
	budgetStr := formatBudgetStr(order.BudgetMin, order.BudgetMax)
	requirementsStr := formatRequirementsStr(requirements)
	hourlyRateStr := ""
	if freelancerProfile.HourlyRate != nil {
		hourlyRateStr = fmt.Sprintf("\nСтавка: $%.2f/час", *freelancerProfile.HourlyRate)
	}
	otherPricesStr := formatOtherPricesStr(otherProposals)

	prompt := fmt.Sprintf(`Рекомендуй цену и сроки:
Заказ: %s
Описание: %s%s%s%s%s

JSON: {"recommended_amount":1000,"min_amount":800,"max_amount":1200,"recommended_days":14,"min_days":10,"max_days":20,"explanation":"причина"}`,
		order.Title, order.Description, budgetStr, requirementsStr, hourlyRateStr, otherPricesStr)

	messages := []map[string]string{
		{"role": "system", "content": "Рекомендуй цены и сроки. Отвечай только JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletionWithOptions(ctx, messages, 256, 0.5)
	if err != nil {
		return nil, err
	}

	var recommendation models.PriceTimelineRecommendation
	if err := json.Unmarshal([]byte(response), &recommendation); err != nil {
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

func (c *Client) EvaluateOrderQuality(
	ctx context.Context,
	order *models.Order,
	requirements []models.OrderRequirement,
) (*models.OrderQualityEvaluation, error) {
	requirementsStr := formatRequirementsStr(requirements)
	budgetStr := formatBudgetStrSimple(order.BudgetMin, order.BudgetMax)
	deadlineStr := formatDeadlineStr(order.DeadlineAt)

	prompt := fmt.Sprintf(`Оцени заказ (1-10):
%s: %s%s%s%s

JSON: {"score":8,"strengths":["плюс"],"weaknesses":["минус"],"recommendations":["совет"]}`,
		order.Title, order.Description, requirementsStr, budgetStr, deadlineStr)

	messages := []map[string]string{
		{"role": "system", "content": "Оценивай заказы. Отвечай только JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletionWithOptions(ctx, messages, 384, 0.5)
	if err != nil {
		return nil, err
	}

	var evaluation models.OrderQualityEvaluation
	if err := json.Unmarshal([]byte(response), &evaluation); err != nil {
		return &models.OrderQualityEvaluation{
			Score:           5,
			Strengths:       []string{"Заказ создан"},
			Weaknesses:      []string{"Требуется дополнительная информация"},
			Recommendations: []string{"Добавьте больше деталей в описание"},
		}, nil
	}

	return &evaluation, nil
}

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

	requirementsStr := formatRequirementsStr(requirements)
	freelancersInfo := formatFreelancersInfo(freelancerProfiles, freelancerPortfolios)

	prompt := fmt.Sprintf(`Выбери ТОП-5 фрилансеров для заказа:
%s: %s%s

Фрилансеры:%s

JSON: {"recommended_freelancers":[{"user_id":"uuid","match_score":9.5,"explanation":"причина"}]}`,
		order.Title, order.Description, requirementsStr, freelancersInfo)

	messages := []map[string]string{
		{"role": "system", "content": "Выбирай подходящих фрилансеров. Отвечай только JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletionWithOptions(ctx, messages, 512, 0.5)
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
