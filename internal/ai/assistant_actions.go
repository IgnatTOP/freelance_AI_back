package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

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

	prompt := fmt.Sprintf(`Резюме переписки по заказу "%s":
%s

JSON ответ:
{"summary":"2-3 предложения","next_steps":["шаг"],"agreements":["согласовано"],"open_questions":["вопрос"]}`, orderTitle, conversationText)

	messagesAI := []map[string]string{
		{"role": "system", "content": "Анализируй переписки. Отвечай только JSON."},
		{"role": "user", "content": prompt},
	}

	response, err := c.chatCompletionWithOptions(ctx, messagesAI, 512, 0.5)
	if err != nil {
		return nil, err
	}

	var summary models.ChatSummary
	if err := json.Unmarshal([]byte(response), &summary); err != nil {
		return &models.ChatSummary{
			Summary:       strings.TrimSpace(response),
			NextSteps:     []string{},
			Agreements:    []string{},
			OpenQuestions: []string{},
		}, nil
	}

	return &summary, nil
}

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
