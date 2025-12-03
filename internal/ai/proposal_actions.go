package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

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
