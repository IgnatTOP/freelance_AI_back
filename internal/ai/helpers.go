package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/models"
)

// PortfolioItem представляет элемент портфолио для AI.
type PortfolioItem struct {
	Title       string
	Description string
	AITags      []string
}

// normalizePortfolioItems преобразует различные типы портфолио в единый формат.
func normalizePortfolioItems(portfolioItems interface{}) []PortfolioItem {
	if portfolioItems == nil {
		return nil
	}

	var items []PortfolioItem

	// Поддерживаем разные типы входных данных
	if src, ok := portfolioItems.([]models.PortfolioItemForAI); ok {
		items = make([]PortfolioItem, len(src))
		for i, it := range src {
			items[i] = PortfolioItem{
				Title:       it.Title,
				Description: it.Description,
				AITags:      it.AITags,
			}
		}
	} else if src, ok := portfolioItems.([]struct {
		Title       string
		Description string
		AITags      []string
	}); ok {
		items = make([]PortfolioItem, len(src))
		for i, it := range src {
			items[i] = PortfolioItem{
				Title:       it.Title,
				Description: it.Description,
				AITags:      it.AITags,
			}
		}
	}

	return items
}

// formatRequirementsStr формирует строку с требованиями заказа.
func formatRequirementsStr(requirements []models.OrderRequirement) string {
	if len(requirements) == 0 {
		return ""
	}
	requiredSkills := make([]string, 0, len(requirements))
	for _, req := range requirements {
		requiredSkills = append(requiredSkills, req.Skill+" ("+req.Level+")")
	}
	return "\nТребуемые навыки: " + strings.Join(requiredSkills, ", ")
}

// formatRequirementsStrSimple формирует строку с требованиями заказа без уровней.
func formatRequirementsStrSimple(requirements []models.OrderRequirement) string {
	if len(requirements) == 0 {
		return ""
	}
	requiredSkills := make([]string, 0, len(requirements))
	for _, req := range requirements {
		requiredSkills = append(requiredSkills, req.Skill)
	}
	return "\nТребуемые навыки для заказа: " + strings.Join(requiredSkills, ", ")
}

// formatSkillsStr формирует строку со навыками.
func formatSkillsStr(skills []string) string {
	if len(skills) == 0 {
		return ""
	}
	return "\nНавыки: " + strings.Join(skills, ", ")
}

// formatSkillsStrForProposal формирует строку со навыками для отклика.
func formatSkillsStrForProposal(skills []string) string {
	if len(skills) == 0 {
		return ""
	}
	return "\nМои навыки: " + strings.Join(skills, ", ")
}

// formatPortfolioStr формирует строку с портфолио.
func formatPortfolioStr(items []PortfolioItem, prefix string) string {
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	if prefix != "" {
		b.WriteString(prefix)
	} else {
		b.WriteString("\n\nРаботы из портфолио:\n")
	}

	for i, it := range items {
		fmt.Fprintf(&b, "%d. %s", i+1, it.Title)
		if it.Description != "" {
			fmt.Fprintf(&b, ": %s", it.Description)
		}
		if len(it.AITags) > 0 {
			fmt.Fprintf(&b, " (Теги: %s)", strings.Join(it.AITags, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// formatBudgetStr формирует строку с бюджетом заказа.
func formatBudgetStr(budgetMin, budgetMax *float64) string {
	if budgetMin == nil || budgetMax == nil {
		return ""
	}
	return fmt.Sprintf("\nБюджет заказа: $%.2f - $%.2f", *budgetMin, *budgetMax)
}

// formatBudgetStrSimple формирует строку с бюджетом заказа (простой формат).
func formatBudgetStrSimple(budgetMin, budgetMax *float64) string {
	if budgetMin == nil || budgetMax == nil {
		return ""
	}
	return fmt.Sprintf("\nБюджет: $%.2f - $%.2f", *budgetMin, *budgetMax)
}

// formatProfileInfo формирует строку с информацией о профиле.
func formatProfileInfo(profile *models.Profile) string {
	var b strings.Builder
	if profile.Bio != nil && *profile.Bio != "" {
		fmt.Fprintf(&b, "\nО себе: %s", *profile.Bio)
	}
	fmt.Fprintf(&b, "\nУровень опыта: %s", profile.ExperienceLevel)
	if profile.HourlyRate != nil {
		fmt.Fprintf(&b, "\nСтавка за час: $%.2f", *profile.HourlyRate)
	}
	return b.String()
}

// formatPriceInfo формирует строку с информацией о цене.
func formatPriceInfo(proposedAmount *float64, budgetMin, budgetMax *float64) string {
	var b strings.Builder
	if proposedAmount != nil {
		fmt.Fprintf(&b, "\nПредложенная цена: $%.2f", *proposedAmount)
	}
	if budgetMin != nil && budgetMax != nil {
		fmt.Fprintf(&b, "\nБюджет заказа: $%.2f - $%.2f", *budgetMin, *budgetMax)
	}
	return b.String()
}

// formatOtherPricesStr формирует строку с ценами других откликов.
func formatOtherPricesStr(otherProposals []*models.Proposal) string {
	if len(otherProposals) == 0 {
		return ""
	}
	prices := make([]string, 0)
	for _, p := range otherProposals {
		if p.ProposedAmount != nil {
			prices = append(prices, fmt.Sprintf("$%.2f", *p.ProposedAmount))
		}
	}
	if len(prices) == 0 {
		return ""
	}
	return fmt.Sprintf("\nЦены других откликов: %s", strings.Join(prices, ", "))
}

// formatComparisonInfo формирует строку с информацией о сравнении откликов.
func formatComparisonInfo(otherProposals []*models.Proposal) string {
	if len(otherProposals) == 0 {
		return ""
	}
	info := fmt.Sprintf("\n\nВсего откликов на этот заказ: %d", len(otherProposals)+1)
	otherPrices := make([]string, 0)
	for _, other := range otherProposals {
		if other.ProposedAmount != nil {
			otherPrices = append(otherPrices, fmt.Sprintf("$%.2f", *other.ProposedAmount))
		}
	}
	if len(otherPrices) > 0 {
		info += "\nЦены других откликов: " + strings.Join(otherPrices, ", ")
	}
	return info
}

// formatFreelancersInfo формирует строку с информацией о фрилансерах.
func formatFreelancersInfo(
	freelancerProfiles []*models.Profile,
	freelancerPortfolios map[uuid.UUID][]models.PortfolioItemForAI,
) string {
	var b strings.Builder
	for i, profile := range freelancerProfiles {
		fmt.Fprintf(&b, "\n\n--- Фрилансер %d ---\n", i+1)
		fmt.Fprintf(&b, "ID: %s\n", profile.UserID)
		fmt.Fprintf(&b, "Имя: %s\n", profile.DisplayName)
		if len(profile.Skills) > 0 {
			fmt.Fprintf(&b, "Навыки: %s\n", strings.Join(profile.Skills, ", "))
		}
		fmt.Fprintf(&b, "Уровень опыта: %s\n", profile.ExperienceLevel)
		if profile.Bio != nil && *profile.Bio != "" {
			fmt.Fprintf(&b, "О себе: %s\n", *profile.Bio)
		}
		if profile.HourlyRate != nil {
			fmt.Fprintf(&b, "Ставка за час: $%.2f\n", *profile.HourlyRate)
		}

		// Добавляем информацию о портфолио
		if portfolio, ok := freelancerPortfolios[profile.UserID]; ok && len(portfolio) > 0 {
			b.WriteString("Портфолио:\n")
			for j, item := range portfolio {
				fmt.Fprintf(&b, "  %d. %s", j+1, item.Title)
				if item.Description != "" {
					fmt.Fprintf(&b, ": %s", item.Description)
				}
				if len(item.AITags) > 0 {
					fmt.Fprintf(&b, " (Теги: %s)", strings.Join(item.AITags, ", "))
				}
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

// formatOrdersInfo формирует строку с информацией о заказах.
func formatOrdersInfo(orders []models.Order) string {
	var b strings.Builder
	for i := range orders {
		order := &orders[i]
		fmt.Fprintf(&b, "\n\n--- Заказ %d ---\n", i+1)
		fmt.Fprintf(&b, "ID: %s\n", order.ID)
		fmt.Fprintf(&b, "Заголовок: %s\n", order.Title)
		fmt.Fprintf(&b, "Описание: %s\n", order.Description)
		if order.BudgetMin != nil && order.BudgetMax != nil {
			fmt.Fprintf(&b, "Бюджет: $%.2f - $%.2f\n", *order.BudgetMin, *order.BudgetMax)
		}
		if order.AISummary != nil {
			fmt.Fprintf(&b, "Краткое резюме: %s\n", *order.AISummary)
		}
	}
	return b.String()
}

// formatProposalsInfo формирует строку с информацией об откликах.
func formatProposalsInfo(
	proposals []*models.Proposal,
	freelancerProfiles map[uuid.UUID]*models.Profile,
	requirements []models.OrderRequirement,
) string {
	// Формируем карту требуемых навыков для сравнения
	requiredSkillsMap := make(map[string]string) // skill -> level
	for _, req := range requirements {
		requiredSkillsMap[strings.ToLower(req.Skill)] = req.Level
	}

	var b strings.Builder
	for i, proposal := range proposals {
		profile := freelancerProfiles[proposal.FreelancerID]
		if profile == nil {
			profile = &models.Profile{
				UserID:          proposal.FreelancerID,
				DisplayName:     "Исполнитель",
				ExperienceLevel: "middle",
				Skills:          []string{},
			}
		}

		// Анализируем соответствие навыков требованиям
		matchingSkills := []string{}
		missingSkills := []string{}
		profileSkillsLower := make(map[string]bool)
		for _, skill := range profile.Skills {
			profileSkillsLower[strings.ToLower(skill)] = true
		}

		for reqSkill, reqLevel := range requiredSkillsMap {
			found := false
			for profileSkill := range profileSkillsLower {
				if strings.Contains(profileSkill, reqSkill) || strings.Contains(reqSkill, profileSkill) {
					matchingSkills = append(matchingSkills, reqSkill+" ("+reqLevel+")")
					found = true
					break
				}
			}
			if !found {
				missingSkills = append(missingSkills, reqSkill+" ("+reqLevel+")")
			}
		}

		fmt.Fprintf(&b, "\n\n--- Отклик %d ---\n", i+1)
		fmt.Fprintf(&b, "ID отклика: %s\n", proposal.ID)
		fmt.Fprintf(&b, "Исполнитель: %s\n", profile.DisplayName)

		if len(profile.Skills) > 0 {
			fmt.Fprintf(&b, "Навыки исполнителя: %s\n", strings.Join(profile.Skills, ", "))
		}

		// Добавляем анализ соответствия
		if len(matchingSkills) > 0 {
			fmt.Fprintf(&b, "✓ Соответствующие навыки: %s\n", strings.Join(matchingSkills, ", "))
		}
		if len(missingSkills) > 0 {
			fmt.Fprintf(&b, "✗ Отсутствующие навыки: %s\n", strings.Join(missingSkills, ", "))
		}

		fmt.Fprintf(&b, "Уровень опыта: %s\n", profile.ExperienceLevel)

		if profile.Bio != nil && *profile.Bio != "" {
			fmt.Fprintf(&b, "О себе: %s\n", *profile.Bio)
		}

		if profile.HourlyRate != nil {
			fmt.Fprintf(&b, "Ставка за час: $%.2f\n", *profile.HourlyRate)
		}

		if proposal.ProposedAmount != nil {
			fmt.Fprintf(&b, "Предложенная цена: $%.2f\n", *proposal.ProposedAmount)
		}

		fmt.Fprintf(&b, "Сопроводительное письмо: %s\n", proposal.CoverLetter)
	}
	return b.String()
}

// formatConversationText формирует текст переписки.
func formatConversationText(messages []models.Message) string {
	var b strings.Builder
	for i := range messages {
		msg := &messages[i]
		author := "Клиент"
		if msg.AuthorType == "freelancer" {
			author = "Исполнитель"
		}
		fmt.Fprintf(&b, "[%s]: %s\n", author, msg.Content)
	}
	return b.String()
}

// formatExperienceStr формирует строку с опытом.
func formatExperienceStr(userExperience string) string {
	if userExperience == "" {
		return ""
	}
	return "\nМой опыт и описание: " + userExperience
}

// formatTagsStr формирует строку с тегами.
func formatTagsStr(aiTags []string) string {
	if len(aiTags) == 0 {
		return ""
	}
	return "Теги: " + strings.Join(aiTags, ", ")
}

// formatDeadlineStr формирует строку с дедлайном.
func formatDeadlineStr(deadlineAt *time.Time) string {
	if deadlineAt == nil {
		return ""
	}
	return fmt.Sprintf("\nДедлайн: %s", deadlineAt.Format("2006-01-02"))
}

