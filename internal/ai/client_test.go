package ai

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ignatzorin/freelance-backend/internal/models"
)

// Тестовые данные
var (
	testOrderTitle       = "Разработка мобильного приложения для доставки еды"
	testOrderDescription = "Нужно разработать iOS и Android приложение для сервиса доставки еды. Функционал: каталог ресторанов, корзина, оплата, отслеживание заказа."
	testBriefDescription = "Приложение для доставки еды с каталогом и оплатой"
	testSkills           = []string{"Swift", "Kotlin", "Firebase", "REST API"}
	testCoverLetter      = "Здравствуйте! Имею 5 лет опыта в мобильной разработке. Разработал более 20 приложений, включая 3 проекта для доставки. Готов приступить немедленно."
	testUserBio          = "Senior Mobile Developer с опытом 5 лет. Специализируюсь на iOS и Android разработке."
	testExperienceLevel  = "senior"
)

func createTestOrder() *models.Order {
	budgetMin := 50000.0
	budgetMax := 100000.0
	return &models.Order{
		ID:          uuid.New(),
		ClientID:    uuid.New(),
		Title:       testOrderTitle,
		Description: testOrderDescription,
		BudgetMin:   &budgetMin,
		BudgetMax:   &budgetMax,
		Status:      "published",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func createTestRequirements() []models.OrderRequirement {
	return []models.OrderRequirement{
		{ID: uuid.New(), Skill: "Swift", Level: "senior"},
		{ID: uuid.New(), Skill: "Kotlin", Level: "senior"},
		{ID: uuid.New(), Skill: "Firebase", Level: "middle"},
	}
}

func createTestProfile() *models.Profile {
	bio := testUserBio
	return &models.Profile{
		UserID:          uuid.New(),
		Bio:             &bio,
		Skills:          testSkills,
		ExperienceLevel: testExperienceLevel,
	}
}

func createTestMessages() []models.Message {
	convID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	return []models.Message{
		{ID: uuid.New(), ConversationID: convID, AuthorID: &user1, AuthorType: "user", Content: "Здравствуйте! Интересует ваш заказ на разработку приложения.", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: uuid.New(), ConversationID: convID, AuthorID: &user2, AuthorType: "user", Content: "Добрый день! Расскажите о вашем опыте.", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: uuid.New(), ConversationID: convID, AuthorID: &user1, AuthorType: "user", Content: "У меня 5 лет опыта, разработал 20+ приложений.", CreatedAt: time.Now().Add(-30 * time.Minute)},
	}
}

// Создаём реальный клиент (для интеграционных тестов)
func createTestClient() *Client {
	baseURL := os.Getenv("AI_BASE_URL")
	model := os.Getenv("AI_MODEL")
	if baseURL == "" {
		baseURL = "https://bothub.chat/api/v2/openai/v1"
	}
	if model == "" {
		model = "grok-4.1-fast:free"
	}
	return NewClient(baseURL, model)
}

// ============ ТЕСТЫ ============

func TestSummarizeOrder(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	fmt.Println("\n========== TEST: SummarizeOrder ==========")
	fmt.Printf("INPUT:\n  Title: %s\n  Description: %s\n", testOrderTitle, testOrderDescription)

	result, err := client.SummarizeOrder(ctx, testOrderTitle, testOrderDescription)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("==========================================")

	// Даже если AI недоступен, должен вернуть fallback
	if result == "" && err != nil {
		t.Logf("AI недоступен, но fallback должен работать: %v", err)
	}
}

func TestGenerateOrderDescription(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	fmt.Println("\n========== TEST: GenerateOrderDescription ==========")
	fmt.Printf("INPUT:\n  Title: %s\n  Brief: %s\n  Skills: %v\n", testOrderTitle, testBriefDescription, testSkills)

	result, err := client.GenerateOrderDescription(ctx, testOrderTitle, testBriefDescription, testSkills)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("====================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestImproveOrderDescription(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	fmt.Println("\n========== TEST: ImproveOrderDescription ==========")
	fmt.Printf("INPUT:\n  Title: %s\n  Description: %s\n", testOrderTitle, testOrderDescription)

	result, err := client.ImproveOrderDescription(ctx, testOrderTitle, testOrderDescription)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("===================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestProposalFeedback(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	order := createTestOrder()

	fmt.Println("\n========== TEST: ProposalFeedback ==========")
	fmt.Printf("INPUT:\n  Order: %s\n  CoverLetter: %s\n", order.Title, testCoverLetter)

	result, err := client.ProposalFeedback(ctx, order, testCoverLetter)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("============================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestGenerateProposal(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	order := createTestOrder()
	requirements := createTestRequirements()

	fmt.Println("\n========== TEST: GenerateProposal ==========")
	fmt.Printf("INPUT:\n  Order: %s\n  Requirements: %d items\n  Skills: %v\n  Experience: %s\n",
		order.Title, len(requirements), testSkills, testUserBio)

	result, err := client.GenerateProposal(ctx, order, requirements, testSkills, testUserBio, nil)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("============================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestSummarizeConversation(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	messages := createTestMessages()

	fmt.Println("\n========== TEST: SummarizeConversation ==========")
	fmt.Printf("INPUT:\n  Messages: %d\n  OrderTitle: %s\n", len(messages), testOrderTitle)
	for i, m := range messages {
		authorID := "N/A"
		if m.AuthorID != nil {
			authorID = m.AuthorID.String()[:8]
		}
		content := m.Content
		if len(content) > 50 {
			content = content[:50]
		}
		fmt.Printf("  [%d] %s: %s\n", i+1, authorID, content)
	}

	result, err := client.SummarizeConversation(ctx, messages, testOrderTitle)

	fmt.Printf("\nOUTPUT:\n  Result: %+v\n  Error: %v\n", result, err)
	fmt.Println("=================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestImproveProfile(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	fmt.Println("\n========== TEST: ImproveProfile ==========")
	fmt.Printf("INPUT:\n  Bio: %s\n  Skills: %v\n  Level: %s\n", testUserBio, testSkills, testExperienceLevel)

	result, err := client.ImproveProfile(ctx, testUserBio, testSkills, testExperienceLevel)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("==========================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestImprovePortfolioItem(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	portfolioTitle := "Приложение для фитнес-трекера"
	portfolioDesc := "Разработал приложение для отслеживания тренировок"
	aiTags := []string{"iOS", "HealthKit", "SwiftUI"}

	fmt.Println("\n========== TEST: ImprovePortfolioItem ==========")
	fmt.Printf("INPUT:\n  Title: %s\n  Description: %s\n  Tags: %v\n", portfolioTitle, portfolioDesc, aiTags)

	result, err := client.ImprovePortfolioItem(ctx, portfolioTitle, portfolioDesc, aiTags)

	fmt.Printf("\nOUTPUT:\n  Result: %s\n  Error: %v\n", result, err)
	fmt.Println("================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestEvaluateOrderQuality(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	order := createTestOrder()
	requirements := createTestRequirements()

	fmt.Println("\n========== TEST: EvaluateOrderQuality ==========")
	desc := order.Description
	if len(desc) > 100 {
		desc = desc[:100]
	}
	fmt.Printf("INPUT:\n  Order: %s\n  Description: %s\n  Requirements: %d\n",
		order.Title, desc, len(requirements))

	result, err := client.EvaluateOrderQuality(ctx, order, requirements)

	fmt.Printf("\nOUTPUT:\n  Result: %+v\n  Error: %v\n", result, err)
	fmt.Println("================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestFindSuitableFreelancers(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	order := createTestOrder()
	requirements := createTestRequirements()

	// Создаём тестовых фрилансеров
	bio1 := testUserBio
	bio2 := "Junior разработчик, 1 год опыта"
	bio3 := "Middle iOS developer, 3 года опыта в Swift"
	profiles := []*models.Profile{
		{
			UserID:          uuid.New(),
			Bio:             &bio1,
			Skills:          testSkills,
			ExperienceLevel: "senior",
		},
		{
			UserID:          uuid.New(),
			Bio:             &bio2,
			Skills:          []string{"JavaScript", "React"},
			ExperienceLevel: "junior",
		},
		{
			UserID:          uuid.New(),
			Bio:             &bio3,
			Skills:          []string{"Swift", "iOS", "CoreData"},
			ExperienceLevel: "middle",
		},
	}

	portfolios := map[uuid.UUID][]models.PortfolioItemForAI{
		profiles[0].UserID: {
			{Title: "Delivery App", Description: "Приложение доставки еды", AITags: []string{"iOS", "Swift"}},
		},
		profiles[2].UserID: {
			{Title: "Fitness Tracker", Description: "Фитнес приложение", AITags: []string{"iOS", "HealthKit"}},
		},
	}

	fmt.Println("\n========== TEST: FindSuitableFreelancers ==========")
	fmt.Printf("INPUT:\n  Order: %s\n  Requirements: %d\n  Freelancers: %d\n",
		order.Title, len(requirements), len(profiles))
	for i, p := range profiles {
		bio := "N/A"
		if p.Bio != nil {
			b := *p.Bio
			if len(b) > 50 {
				b = b[:50]
			}
			bio = b
		}
		fmt.Printf("  [%d] Skills: %v, Bio: %s\n", i+1, p.Skills, bio)
	}

	result, err := client.FindSuitableFreelancers(ctx, order, requirements, profiles, portfolios)

	fmt.Printf("\nOUTPUT:\n  Found: %d freelancers\n", len(result))
	for i, f := range result {
		fmt.Printf("  [%d] Score: %.1f, Reason: %s\n", i+1, f.MatchScore, f.Explanation)
	}
	fmt.Printf("  Error: %v\n", err)
	fmt.Println("===================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestRecommendPriceAndTimeline(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	order := createTestOrder()
	requirements := createTestRequirements()
	profile := createTestProfile()

	fmt.Println("\n========== TEST: RecommendPriceAndTimeline ==========")
	fmt.Printf("INPUT:\n  Order: %s\n  Budget: %.0f-%.0f\n  Freelancer: %v\n",
		order.Title, *order.BudgetMin, *order.BudgetMax, profile.Skills)

	result, err := client.RecommendPriceAndTimeline(ctx, order, requirements, profile, nil)

	fmt.Printf("\nOUTPUT:\n  Result: %+v\n  Error: %v\n", result, err)
	fmt.Println("=====================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}

func TestRecommendRelevantOrders(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()
	profile := createTestProfile()

	portfolioItems := []models.PortfolioItemForAI{
		{Title: "Delivery App", Description: "Приложение доставки", AITags: []string{"iOS", "Swift"}},
	}

	orders := []models.Order{
		*createTestOrder(),
		{
			ID:          uuid.New(),
			Title:       "Разработка веб-сайта на React",
			Description: "Нужен современный сайт на React с админкой",
			Status:      "published",
		},
		{
			ID:          uuid.New(),
			Title:       "iOS приложение для банка",
			Description: "Мобильный банкинг на Swift",
			Status:      "published",
		},
	}

	fmt.Println("\n========== TEST: RecommendRelevantOrders ==========")
	fmt.Printf("INPUT:\n  Freelancer Skills: %v\n  Orders: %d\n", profile.Skills, len(orders))
	for i, o := range orders {
		fmt.Printf("  [%d] %s\n", i+1, o.Title)
	}

	result, explanation, err := client.RecommendRelevantOrders(ctx, profile, portfolioItems, orders)

	fmt.Printf("\nOUTPUT:\n  Recommended: %d orders\n  Explanation: %s\n", len(result), explanation)
	for i, r := range result {
		fmt.Printf("  [%d] Score: %.1f, Order: %s\n", i+1, r.MatchScore, r.OrderID)
	}
	fmt.Printf("  Error: %v\n", err)
	fmt.Println("===================================================")

	if err != nil {
		t.Logf("Ошибка (AI может быть недоступен): %v", err)
	}
}
