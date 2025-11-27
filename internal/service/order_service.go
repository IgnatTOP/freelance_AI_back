package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ignatzorin/freelance-backend/internal/logger"
	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// OrderRepository описывает взаимодействие сервиса с хранилищем заказов.
type OrderRepository interface {
	Create(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, attachmentIDs []uuid.UUID) error
	List(ctx context.Context, params repository.ListFilterParams) (*repository.ListResult, error)
	ListMyOrders(ctx context.Context, userID uuid.UUID) ([]models.Order, []models.Order, error)
	Update(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, attachmentIDs []uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, clientID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*models.Order, []models.OrderRequirement, []models.OrderAttachment, error)
	ListRequirements(ctx context.Context, orderID uuid.UUID) ([]models.OrderRequirement, error)
	ListAttachments(ctx context.Context, orderID uuid.UUID) ([]models.OrderAttachment, error)
	GetProposalByID(ctx context.Context, id uuid.UUID) (*models.Proposal, error)
	UpdateProposalStatus(ctx context.Context, id uuid.UUID, status string) (*models.Proposal, error)
	CreateProposal(ctx context.Context, proposal *models.Proposal) error
	ListProposals(ctx context.Context, orderID uuid.UUID) ([]models.Proposal, error)
	GetMyProposalForOrder(ctx context.Context, orderID, freelancerID uuid.UUID) (*models.Proposal, error)
	ListMyProposals(ctx context.Context, freelancerID uuid.UUID) ([]models.Proposal, error)
	CreateConversation(ctx context.Context, conv *models.Conversation) error
	AddMessage(ctx context.Context, msg *models.Message) error
	GetConversationByParticipants(ctx context.Context, orderID uuid.UUID, clientID, freelancerID uuid.UUID) (*models.Conversation, error)
	GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error)
	ListMyConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
	GetLastMessageForConversation(ctx context.Context, conversationID uuid.UUID) (*models.Message, error)
	ListMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]models.Message, error)
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.Message, error)
	UpdateMessage(ctx context.Context, messageID uuid.UUID, newContent string) error
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error
	UpdateAISummary(ctx context.Context, orderID uuid.UUID, clientID uuid.UUID, summary string) error
	UpdateProposalAIFeedback(ctx context.Context, proposalID uuid.UUID, feedback string) error
	UpdateBestRecommendation(ctx context.Context, orderID uuid.UUID, proposalID *uuid.UUID, justification string) error
	GetProposalsLastUpdateTime(ctx context.Context, orderID uuid.UUID) (*time.Time, error)
}

// ProfileRepository описывает взаимодействие с профилями пользователей.
type ProfileRepository interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
}

// PortfolioRepositoryForAI описывает минимальный контракт для получения работ портфолио.
type PortfolioRepositoryForAI interface {
	// List возвращает все элементы портфолио пользователя.
	List(ctx context.Context, userID uuid.UUID) ([]models.PortfolioItem, error)
}

// UserRepositoryForAI описывает минимальный контракт для получения пользователей.
type UserRepositoryForAI interface {
	// ListFreelancers возвращает список всех активных фрилансеров.
	ListFreelancers(ctx context.Context, limit, offset int) ([]*models.User, error)
	// CountFreelancers возвращает общее количество активных фрилансеров.
	CountFreelancers(ctx context.Context) (int, error)
}

// AIHelper описывает упрощённый контракт с AI подсистемой.
type AIHelper interface {
	SummarizeOrder(ctx context.Context, title, description string) (string, error)
	StreamSummarizeOrder(ctx context.Context, title, description string, onDelta func(chunk string) error) error
	ProposalFeedback(ctx context.Context, order *models.Order, coverLetter string) (string, error)
	ProposalAnalysisForClient(ctx context.Context, order *models.Order, proposal *models.Proposal, freelancerProfile *models.Profile, requirements []models.OrderRequirement, portfolioItems interface{}, otherProposals []*models.Proposal) (string, error)
	RecommendBestProposal(ctx context.Context, order *models.Order, proposals []*models.Proposal, freelancerProfiles map[uuid.UUID]*models.Profile, requirements []models.OrderRequirement) (*uuid.UUID, string, error)
	GenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string) (string, error)
	StreamGenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string, onDelta func(chunk string) error) error
	GenerateProposal(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, userSkills []string, userExperience string, portfolioItems interface{}) (string, error)
	ImproveOrderDescription(ctx context.Context, title, description string) (string, error)
	StreamImproveOrderDescription(ctx context.Context, title, description string, onDelta func(chunk string) error) error
	StreamProposalFeedback(ctx context.Context, order *models.Order, coverLetter string, onDelta func(chunk string) error) error
	StreamGenerateProposal(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, userSkills []string, userExperience string, portfolioItems interface{}, onDelta func(chunk string) error) error
	SummarizeConversation(ctx context.Context, messages []models.Message, orderTitle string) (*models.ChatSummary, error)
	StreamSummarizeConversation(ctx context.Context, messages []models.Message, orderTitle string, onDelta func(chunk string) error) error
	RecommendRelevantOrders(ctx context.Context, freelancerProfile *models.Profile, portfolioItems []models.PortfolioItemForAI, orders []models.Order) ([]models.RecommendedOrder, string, error)
	RecommendPriceAndTimeline(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, freelancerProfile *models.Profile, otherProposals []*models.Proposal) (*models.PriceTimelineRecommendation, error)
	ImproveProfile(ctx context.Context, currentBio string, skills []string, experienceLevel string) (string, error)
	StreamImproveProfile(ctx context.Context, currentBio string, skills []string, experienceLevel string, onDelta func(chunk string) error) error
	ImprovePortfolioItem(ctx context.Context, title, description string, aiTags []string) (string, error)
	StreamImprovePortfolioItem(ctx context.Context, title, description string, aiTags []string, onDelta func(chunk string) error) error
	EvaluateOrderQuality(ctx context.Context, order *models.Order, requirements []models.OrderRequirement) (*models.OrderQualityEvaluation, error)
	StreamEvaluateOrderQuality(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, onDelta func(chunk string) error, onComplete func(evaluation *models.OrderQualityEvaluation) error) error
	FindSuitableFreelancers(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, freelancerProfiles []*models.Profile, freelancerPortfolios map[uuid.UUID][]models.PortfolioItemForAI) ([]models.SuitableFreelancer, error)
	StreamFindSuitableFreelancers(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, freelancerProfiles []*models.Profile, freelancerPortfolios map[uuid.UUID][]models.PortfolioItemForAI, onDelta func(chunk string) error, onComplete func(data []models.SuitableFreelancer) error) error
	StreamRecommendRelevantOrders(ctx context.Context, freelancerProfile *models.Profile, portfolioItems []models.PortfolioItemForAI, orders []models.Order, onDelta func(chunk string) error, onComplete func(recommendedOrders []models.RecommendedOrder, generalExplanation string) error) error
	StreamRecommendPriceAndTimeline(ctx context.Context, order *models.Order, requirements []models.OrderRequirement, freelancerProfile *models.Profile, otherProposals []*models.Proposal, onDelta func(chunk string) error, onComplete func(recommendation *models.PriceTimelineRecommendation) error) error
	AIChatAssistant(ctx context.Context, userMessage string, userRole string, contextData map[string]interface{}) (string, error)
	StreamAIChatAssistant(ctx context.Context, userMessage string, userRole string, contextData map[string]interface{}, onDelta func(chunk string) error) error
}

// WSNotifier интерфейс для отправки WebSocket уведомлений.
type WSNotifier interface {
	BroadcastToUser(userID uuid.UUID, event string, data interface{}) error
}

// OrderService содержит бизнес-логику работы с заказами.
type OrderService struct {
	repo      OrderRepository
	profile   ProfileRepository
	portfolio PortfolioRepositoryForAI
	users     UserRepositoryForAI
	ai        AIHelper
	hub       WSNotifier
}

// NewOrderService создаёт новый сервис заказов.
func NewOrderService(repo OrderRepository, profile ProfileRepository, portfolio PortfolioRepositoryForAI, users UserRepositoryForAI, ai AIHelper) *OrderService {
	return &OrderService{
		repo:      repo,
		profile:   profile,
		portfolio: portfolio,
		users:     users,
		ai:        ai,
	}
}

// SetHub устанавливает WebSocket hub для отправки уведомлений.
func (s *OrderService) SetHub(hub WSNotifier) {
	s.hub = hub
}

// CreateOrderInput описывает входные данные.
type CreateOrderInput struct {
	ClientID      uuid.UUID
	Title         string
	Description   string
	BudgetMin     *float64
	BudgetMax     *float64
	DeadlineAt    *time.Time
	Requirements  []models.OrderRequirement
	AttachmentIDs []uuid.UUID
}

// UpdateOrderInput описывает входные данные для обновления заказа.
type UpdateOrderInput struct {
	OrderID       uuid.UUID
	ClientID      uuid.UUID
	Title         string
	Description   string
	BudgetMin     *float64
	BudgetMax     *float64
	Status        string
	DeadlineAt    *time.Time
	Requirements  []models.OrderRequirement
	AttachmentIDs []uuid.UUID
}

// ProposalInput описывает отклик.
type ProposalInput struct {
	OrderID      uuid.UUID
	FreelancerID uuid.UUID
	CoverLetter  string
	Amount       *float64
}

// CreateOrder создаёт заказ и возвращает его.
func (s *OrderService) CreateOrder(ctx context.Context, in CreateOrderInput) (*models.Order, error) {
	// Валидация входных данных
	if in.Title == "" {
		return nil, fmt.Errorf("order service: заголовок заказа не может быть пустым")
	}
	if in.Description == "" {
		return nil, fmt.Errorf("order service: описание заказа не может быть пустым")
	}
	if in.BudgetMin != nil && in.BudgetMax != nil && *in.BudgetMin > *in.BudgetMax {
		return nil, fmt.Errorf("order service: минимальный бюджет не может быть больше максимального")
	}
	if in.DeadlineAt != nil && in.DeadlineAt.Before(time.Now()) {
		return nil, fmt.Errorf("order service: дедлайн не может быть в прошлом")
	}

	order := &models.Order{
		ClientID:    in.ClientID,
		Title:       in.Title,
		Description: in.Description,
		Status:      models.OrderStatusPublished,
		BudgetMin:   in.BudgetMin,
		BudgetMax:   in.BudgetMax,
		DeadlineAt:  in.DeadlineAt,
	}

	if s.ai != nil {
		if summary, err := s.ai.SummarizeOrder(ctx, in.Title, in.Description); err == nil {
			order.AISummary = &summary
		}
	}

	if err := s.repo.Create(ctx, order, in.Requirements, in.AttachmentIDs); err != nil {
		return nil, err
	}

	return order, nil
}

// ListOrders возвращает список заказов с фильтрацией и поиском.
func (s *OrderService) ListOrders(ctx context.Context, params repository.ListFilterParams) (*repository.ListResult, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 20
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	return s.repo.List(ctx, params)
}

// UpdateOrder обновляет существующий заказ.
func (s *OrderService) UpdateOrder(ctx context.Context, in UpdateOrderInput) (*models.Order, error) {
	existing, err := s.repo.GetByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}

	if existing.ClientID != in.ClientID {
		return nil, fmt.Errorf("order service: у вас нет прав на изменение заказа")
	}

	// Валидация статуса
	if in.Status != "" {
		if _, ok := models.ValidOrderStatuses[in.Status]; !ok {
			return nil, fmt.Errorf("order service: некорректный статус заказа")
		}
		// Проверка на возможность изменения статуса
		if existing.Status == models.OrderStatusCompleted && in.Status != models.OrderStatusCompleted {
			return nil, fmt.Errorf("order service: нельзя изменить статус завершённого заказа")
		}
		if existing.Status == models.OrderStatusCancelled && in.Status != models.OrderStatusCancelled {
			return nil, fmt.Errorf("order service: нельзя изменить статус отменённого заказа")
		}
	}

	// Валидация бюджета
	if in.BudgetMin != nil && in.BudgetMax != nil && *in.BudgetMin > *in.BudgetMax {
		return nil, fmt.Errorf("order service: минимальный бюджет не может быть больше максимального")
	}

	// Валидация дедлайна
	if in.DeadlineAt != nil && in.DeadlineAt.Before(time.Now()) {
		return nil, fmt.Errorf("order service: дедлайн не может быть в прошлом")
	}

	// Валидация входных данных
	if in.Title == "" {
		return nil, fmt.Errorf("order service: заголовок заказа не может быть пустым")
	}
	if in.Description == "" {
		return nil, fmt.Errorf("order service: описание заказа не может быть пустым")
	}

	needsResummary := existing.Title != in.Title || existing.Description != in.Description

	existing.Title = in.Title
	existing.Description = in.Description
	existing.BudgetMin = in.BudgetMin
	existing.BudgetMax = in.BudgetMax
	if in.Status != "" {
		existing.Status = in.Status
	}
	existing.DeadlineAt = in.DeadlineAt

	if s.ai != nil && needsResummary {
		if summary, err := s.ai.SummarizeOrder(ctx, existing.Title, existing.Description); err == nil {
			existing.AISummary = &summary
		}
	}

	if err := s.repo.Update(ctx, existing, in.Requirements, in.AttachmentIDs); err != nil {
		return nil, err
	}

	return existing, nil
}

// CreateProposal создаёт отклик и может сформировать чат.
func (s *OrderService) CreateProposal(ctx context.Context, in ProposalInput) (*models.Proposal, error) {
	// Валидация входных данных
	if in.CoverLetter == "" {
		return nil, fmt.Errorf("order service: сопроводительное письмо не может быть пустым")
	}

	order, err := s.repo.GetByID(ctx, in.OrderID)
	if err != nil {
		return nil, fmt.Errorf("order service: не найден заказ: %w", err)
	}

	// Проверка, что заказ доступен для предложений
	if order.Status != models.OrderStatusPublished {
		return nil, fmt.Errorf("order service: нельзя создать предложение для заказа со статусом %s", order.Status)
	}

	// Проверка, что фрилансер не является клиентом
	if order.ClientID == in.FreelancerID {
		return nil, fmt.Errorf("order service: нельзя создать предложение на свой заказ")
	}

	// Проверка на дублирование предложений
	existingProposals, err := s.repo.ListProposals(ctx, in.OrderID)
	if err == nil {
		for _, p := range existingProposals {
			if p.FreelancerID == in.FreelancerID && p.Status != models.ProposalStatusRejected {
				return nil, fmt.Errorf("order service: вы уже отправили предложение на этот заказ")
			}
		}
	}

	proposal := &models.Proposal{
		OrderID:        in.OrderID,
		FreelancerID:   in.FreelancerID,
		CoverLetter:    in.CoverLetter,
		ProposedAmount: in.Amount,
		Status:         models.ProposalStatusPending,
	}

	// Не генерируем feedback для исполнителя при создании отклика
	// Анализ для заказчика будет генерироваться при получении списка откликов

	if err := s.repo.CreateProposal(ctx, proposal); err != nil {
		return nil, err
	}

	conv := &models.Conversation{
		OrderID:      &order.ID,
		ClientID:     order.ClientID,
		FreelancerID: in.FreelancerID,
	}

	// Пытаемся создать conversation, но не критично, если он уже существует
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		// Если conversation уже существует, это не ошибка
		// Проверяем, существует ли conversation
		existingConv, checkErr := s.repo.GetConversationByParticipants(ctx, order.ID, order.ClientID, in.FreelancerID)
		if checkErr != nil || existingConv == nil {
			// Если conversation не существует и не удалось создать, возвращаем ошибку
			return nil, fmt.Errorf("order service: не удалось создать диалог: %w", err)
		}
		// Conversation существует, продолжаем
	}

	return proposal, nil
}

// BestRecommendation содержит рекомендацию лучшего исполнителя.
type BestRecommendation struct {
	ProposalID    *uuid.UUID `json:"proposal_id,omitempty"`
	Justification string     `json:"justification,omitempty"`
}

// ListProposalsResult содержит список откликов и рекомендацию лучшего.
type ListProposalsResult struct {
	Proposals          []models.Proposal   `json:"proposals"`
	BestRecommendation *BestRecommendation `json:"best_recommendation,omitempty"`
}

// ListProposals возвращает отклики по заказу.
// Если вызывается заказчиком, возвращает кэшированные AI анализы и запускает асинхронную регенерацию при необходимости.
func (s *OrderService) ListProposals(ctx context.Context, orderID uuid.UUID, clientID *uuid.UUID) (*ListProposalsResult, error) {
	proposals, err := s.repo.ListProposals(ctx, orderID)
	if err != nil {
		return nil, err
	}

	result := &ListProposalsResult{
		Proposals: proposals,
	}

	// Если вызывается заказчиком и есть AI сервис
	if clientID != nil && s.ai != nil && s.profile != nil && len(proposals) > 0 {
		order, err := s.repo.GetByID(ctx, orderID)
		if err == nil && order != nil && order.ClientID == *clientID {
			// Используем кэшированные данные если они есть
			if order.BestRecommendationProposalID != nil && order.BestRecommendationJustification != nil {
				result.BestRecommendation = &BestRecommendation{
					ProposalID:    order.BestRecommendationProposalID,
					Justification: *order.BestRecommendationJustification,
				}
			}

			// Проверяем, нужно ли регенерировать анализ
			needsRegeneration := s.needsAIRegeneration(ctx, orderID, order)

			if needsRegeneration {
				// Запускаем асинхронную генерацию в фоне
				go func() {
					bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer cancel()
					s.generateAIAnalysisAsync(bgCtx, orderID, *clientID, order, proposals)
				}()
			}
		}
	}

	return result, nil
}

// needsAIRegeneration проверяет, нужно ли регенерировать AI анализ.
func (s *OrderService) needsAIRegeneration(ctx context.Context, orderID uuid.UUID, order *models.Order) bool {
	// Если анализа еще не было, нужно сгенерировать
	if order.AIAnalysisUpdatedAt == nil {
		return true
	}

	// Проверяем, были ли изменения в откликах после последней генерации
	proposalsLastUpdate, err := s.repo.GetProposalsLastUpdateTime(ctx, orderID)
	if err != nil || proposalsLastUpdate == nil {
		return true
	}

	// Если отклики обновлялись после генерации анализа, нужно регенерировать
	if proposalsLastUpdate.After(*order.AIAnalysisUpdatedAt) {
		return true
	}

	// Проверяем, был ли обновлен заказ после генерации анализа
	if order.UpdatedAt.After(*order.AIAnalysisUpdatedAt) {
		return true
	}

	return false
}

// generateAIAnalysisAsync генерирует AI анализ асинхронно в фоне.
func (s *OrderService) generateAIAnalysisAsync(ctx context.Context, orderID uuid.UUID, clientID uuid.UUID, order *models.Order, proposals []models.Proposal) {
	// Получаем требования заказа
	requirements, err := s.repo.ListRequirements(ctx, orderID)
	if err != nil {
		requirements = []models.OrderRequirement{}
	}

	// Собираем профили всех исполнителей
	freelancerProfiles := make(map[uuid.UUID]*models.Profile)
	proposalPointers := make([]*models.Proposal, len(proposals))

	// Генерируем анализ для каждого отклика
	for i := range proposals {
		proposalPointers[i] = &proposals[i]

		// Получаем профиль исполнителя
		freelancerProfile, err := s.profile.GetProfile(ctx, proposals[i].FreelancerID)
		if err != nil || freelancerProfile == nil {
			// Если профиль не найден, создаём минимальный
			freelancerProfile = &models.Profile{
				UserID:          proposals[i].FreelancerID,
				DisplayName:     "Исполнитель",
				ExperienceLevel: "middle",
				Skills:          []string{},
			}
		}
		freelancerProfiles[proposals[i].FreelancerID] = freelancerProfile

		// Формируем список других откликов для сравнения
		otherProposals := make([]*models.Proposal, 0)
		for j := range proposals {
			if i != j {
				otherProposals = append(otherProposals, &proposals[j])
			}
		}

		// Формируем список "других" откликов для сравнительного анализа
		if proposals[i].AIFeedback == nil || *proposals[i].AIFeedback == "" {
			// Готовим данные портфолио исполнителя для AI (если есть репозиторий)
			var portfolioForAI []models.PortfolioItemForAI
			if s.portfolio != nil {
				if items, err := s.portfolio.List(ctx, proposals[i].FreelancerID); err == nil {
					portfolioForAI = make([]models.PortfolioItemForAI, len(items))
					for idx, it := range items {
						var description string
						if it.Description != nil {
							description = *it.Description
						}
						portfolioForAI[idx] = models.PortfolioItemForAI{
							Title:       it.Title,
							Description: description,
							AITags:      it.AITags,
						}
					}
				}
			}

			// Генерируем анализ для заказчика только если его еще нет
			if analysis, err := s.ai.ProposalAnalysisForClient(ctx, order, &proposals[i], freelancerProfile, requirements, portfolioForAI, otherProposals); err == nil && analysis != "" {
				// Сохраняем в кэш
				_ = s.repo.UpdateProposalAIFeedback(ctx, proposals[i].ID, analysis)
			}
		}
	}

	// Генерируем (или пересчитываем) рекомендацию лучшего исполнителя,
	// если есть хотя бы 2 отклика. Решение о необходимости регенерации
	// принимается выше в needsAIRegeneration по полю AIAnalysisUpdatedAt.
	if len(proposals) >= 2 {
		if bestProposalID, justification, err := s.ai.RecommendBestProposal(ctx, order, proposalPointers, freelancerProfiles, requirements); err == nil && bestProposalID != nil {
			// Сохраняем в кэш
			_ = s.repo.UpdateBestRecommendation(ctx, orderID, bestProposalID, justification)

			// Отправляем уведомление через WebSocket
			if s.hub != nil {
				_ = s.hub.BroadcastToUser(clientID, "proposals.ai_analysis_ready", map[string]interface{}{
					"order_id": orderID,
					"message":  "AI анализ откликов готов",
				})
			}
		}
	}
}

// GetMyProposalForOrder возвращает предложение пользователя для конкретного заказа.
func (s *OrderService) GetMyProposalForOrder(ctx context.Context, orderID, freelancerID uuid.UUID) (*models.Proposal, error) {
	return s.repo.GetMyProposalForOrder(ctx, orderID, freelancerID)
}

// GetProposalFeedback возвращает рекомендации по улучшению отклика для исполнителя.
func (s *OrderService) GetProposalFeedback(ctx context.Context, orderID uuid.UUID, freelancerID uuid.UUID) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return "", err
	}

	proposal, err := s.repo.GetMyProposalForOrder(ctx, orderID, freelancerID)
	if err != nil {
		return "", err
	}

	// Проверяем, что отклик принадлежит пользователю
	if proposal.FreelancerID != freelancerID {
		return "", fmt.Errorf("order service: у вас нет доступа к этому отклику")
	}

	feedback, err := s.ai.ProposalFeedback(ctx, order, proposal.CoverLetter)
	if err != nil {
		return "", err
	}

	return feedback, nil
}

// StreamProposalFeedback возвращает рекомендации по улучшению отклика потоково.
// Используется для стриминга ответа AI в реальном времени.
func (s *OrderService) StreamProposalFeedback(
	ctx context.Context,
	orderID uuid.UUID,
	freelancerID uuid.UUID,
	onDelta func(chunk string) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	proposal, err := s.repo.GetMyProposalForOrder(ctx, orderID, freelancerID)
	if err != nil {
		return err
	}

	// Проверяем, что отклик принадлежит пользователю
	if proposal.FreelancerID != freelancerID {
		return fmt.Errorf("order service: у вас нет доступа к этому отклику")
	}

	return s.ai.StreamProposalFeedback(ctx, order, proposal.CoverLetter, onDelta)
}

// GetOrder возвращает заказ по идентификатору.
func (s *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	return s.repo.GetByID(ctx, id)
}

// GetOrderWithDetails возвращает заказ со всеми связанными данными (требования и вложения).
func (s *OrderService) GetOrderWithDetails(ctx context.Context, id uuid.UUID) (*models.Order, []models.OrderRequirement, []models.OrderAttachment, error) {
	return s.repo.GetByIDWithDetails(ctx, id)
}

// ListMyOrders возвращает заказы пользователя (как заказчика и как исполнителя).
func (s *OrderService) ListMyOrders(ctx context.Context, userID uuid.UUID) ([]models.Order, []models.Order, error) {
	return s.repo.ListMyOrders(ctx, userID)
}

// ListRequirements возвращает список требований к заказу.
func (s *OrderService) ListRequirements(ctx context.Context, orderID uuid.UUID) ([]models.OrderRequirement, error) {
	return s.repo.ListRequirements(ctx, orderID)
}

// ListAttachments возвращает список вложений заказа.
func (s *OrderService) ListAttachments(ctx context.Context, orderID uuid.UUID) ([]models.OrderAttachment, error) {
	return s.repo.ListAttachments(ctx, orderID)
}

// UpdateProposalStatus обновляет статус отклика.
func (s *OrderService) UpdateProposalStatus(ctx context.Context, actorID uuid.UUID, proposalID uuid.UUID, status string) (*models.Proposal, *models.Conversation, error) {
	proposal, err := s.repo.GetProposalByID(ctx, proposalID)
	if err != nil {
		return nil, nil, err
	}

	order, err := s.repo.GetByID(ctx, proposal.OrderID)
	if err != nil {
		return nil, nil, err
	}

	if order.ClientID != actorID {
		return nil, nil, fmt.Errorf("order service: у вас нет прав изменять статус отклика")
	}

	// Валидация статуса
	if _, ok := models.ValidProposalStatuses[status]; !ok {
		return nil, nil, fmt.Errorf("order service: некорректный статус отклика")
	}

	// Проверка, что заказ ещё не завершён или отменён
	if order.Status == models.OrderStatusCompleted || order.Status == models.OrderStatusCancelled {
		return nil, nil, fmt.Errorf("order service: нельзя изменить статус предложения для завершённого или отменённого заказа")
	}

	updatedProposal, err := s.repo.UpdateProposalStatus(ctx, proposalID, status)
	if err != nil {
		return nil, nil, err
	}

	var conversation *models.Conversation

	if status == models.ProposalStatusAccepted {
		// Автоматически меняем статус заказа на in_progress
		if order.Status == models.OrderStatusPublished {
			order.Status = models.OrderStatusInProgress
			// Обновляем заказ без изменения других полей
			err = s.repo.Update(ctx, order, []models.OrderRequirement{}, []uuid.UUID{})
			if err != nil {
				// Логируем ошибку, но не прерываем процесс
				if logger.Log != nil {
					logger.Log.WithFields(map[string]interface{}{
						"order_id": order.ID,
						"error":    err.Error(),
					}).Warn("order service: не удалось обновить статус заказа")
				}
			}
		}

		conversation, err = s.repo.GetConversationByParticipants(ctx, proposal.OrderID, order.ClientID, proposal.FreelancerID)
		if err != nil {
			if errors.Is(err, repository.ErrConversationNotFound) {
				orderID := proposal.OrderID
				conversation = &models.Conversation{
					OrderID:      &orderID,
					ClientID:     order.ClientID,
					FreelancerID: proposal.FreelancerID,
				}
				if err := s.repo.CreateConversation(ctx, conversation); err != nil {
					return updatedProposal, nil, err
				}
			} else {
				return updatedProposal, nil, err
			}
		}
	}

	return updatedProposal, conversation, nil
}

// GetConversation возвращает существующий чат между клиентом и исполнителем.
func (s *OrderService) GetConversation(ctx context.Context, orderID uuid.UUID, clientID, freelancerID uuid.UUID) (*models.Conversation, error) {
	return s.repo.GetConversationByParticipants(ctx, orderID, clientID, freelancerID)
}

// GetConversationByID возвращает чат по идентификатору.
func (s *OrderService) GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	return s.repo.GetConversationByID(ctx, id)
}

// GetOrderChat возвращает чат для заказа (только если есть accepted proposal).
// Для заказчика возвращает чат с принятым исполнителем.
// Для исполнителя возвращает чат, если его предложение принято.
func (s *OrderService) GetOrderChat(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*models.Conversation, *models.Proposal, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}

	// Получаем accepted proposal для этого заказа
	proposals, err := s.repo.ListProposals(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}

	var acceptedProposal *models.Proposal
	for i := range proposals {
		if proposals[i].Status == models.ProposalStatusAccepted {
			acceptedProposal = &proposals[i]
			break
		}
	}

	if acceptedProposal == nil {
		return nil, nil, fmt.Errorf("order service: для этого заказа нет принятого исполнителя")
	}

	// Проверяем доступ
	if order.ClientID != userID && acceptedProposal.FreelancerID != userID {
		return nil, nil, fmt.Errorf("order service: у вас нет доступа к этому чату")
	}

	// Получаем или создаем чат
	conversation, err := s.repo.GetConversationByParticipants(ctx, orderID, order.ClientID, acceptedProposal.FreelancerID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			// Создаем чат, если его нет
			conversation = &models.Conversation{
				OrderID:      &orderID,
				ClientID:     order.ClientID,
				FreelancerID: acceptedProposal.FreelancerID,
			}
			if err := s.repo.CreateConversation(ctx, conversation); err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	return conversation, acceptedProposal, nil
}

// ListMyConversations возвращает все чаты пользователя.
func (s *OrderService) ListMyConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	return s.repo.ListMyConversations(ctx, userID)
}

// GetLastMessageForConversation возвращает последнее сообщение в чате.
func (s *OrderService) GetLastMessageForConversation(ctx context.Context, conversationID uuid.UUID) (*models.Message, error) {
	return s.repo.GetLastMessageForConversation(ctx, conversationID)
}

// ListMessages возвращает сообщения в чате с пагинацией.
func (s *OrderService) ListMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListMessages(ctx, conversationID, limit, offset)
}

// SendMessage добавляет сообщение в чат.
func (s *OrderService) SendMessage(ctx context.Context, conversationID, authorID uuid.UUID, content string) (*models.Message, *models.Conversation, error) {
	// Валидация входных данных
	if content == "" {
		return nil, nil, fmt.Errorf("order service: текст сообщения не может быть пустым")
	}
	if len(content) > 5000 {
		return nil, nil, fmt.Errorf("order service: сообщение слишком длинное (максимум 5000 символов)")
	}

	conversation, err := s.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}

	var authorType string
	switch {
	case conversation.ClientID == authorID:
		authorType = "client"
	case conversation.FreelancerID == authorID:
		authorType = "freelancer"
	default:
		return nil, nil, fmt.Errorf("order service: у вас нет доступа к этому чату")
	}

	message := &models.Message{
		ConversationID: conversationID,
		AuthorType:     authorType,
		AuthorID:       &authorID,
		Content:        content,
		AIMetadata:     json.RawMessage("null"),
	}

	if err := s.repo.AddMessage(ctx, message); err != nil {
		return nil, nil, err
	}

	return message, conversation, nil
}

// UpdateMessage обновляет содержимое сообщения.
func (s *OrderService) UpdateMessage(ctx context.Context, messageID uuid.UUID, authorID uuid.UUID, newContent string) (*models.Message, error) {
	// Валидация входных данных
	if newContent == "" {
		return nil, fmt.Errorf("order service: текст сообщения не может быть пустым")
	}
	if len(newContent) > 5000 {
		return nil, fmt.Errorf("order service: сообщение слишком длинное (максимум 5000 символов)")
	}

	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Проверка прав: только автор может редактировать
	if message.AuthorID == nil || *message.AuthorID != authorID {
		return nil, fmt.Errorf("order service: у вас нет прав на редактирование этого сообщения")
	}

	// Проверка, что сообщение не удалено
	if message.Content == "[Сообщение удалено]" {
		return nil, fmt.Errorf("order service: нельзя редактировать удалённое сообщение")
	}

	if err := s.repo.UpdateMessage(ctx, messageID, newContent); err != nil {
		return nil, err
	}

	// Получаем обновлённое сообщение
	updatedMessage, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	return updatedMessage, nil
}

// DeleteMessage удаляет сообщение.
func (s *OrderService) DeleteMessage(ctx context.Context, messageID uuid.UUID, authorID uuid.UUID) error {
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Проверка прав: только автор может удалять
	if message.AuthorID == nil || *message.AuthorID != authorID {
		return fmt.Errorf("order service: у вас нет прав на удаление этого сообщения")
	}

	// Проверка, что сообщение ещё не удалено
	if message.Content == "[Сообщение удалено]" {
		return fmt.Errorf("order service: сообщение уже удалено")
	}

	return s.repo.DeleteMessage(ctx, messageID)
}

// DeleteOrder удаляет заказ.
func (s *OrderService) DeleteOrder(ctx context.Context, orderID uuid.UUID, clientID uuid.UUID) error {
	// Проверяем, что заказ существует и принадлежит клиенту
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.ClientID != clientID {
		return fmt.Errorf("order service: у вас нет прав на удаление этого заказа")
	}

	// Проверяем, можно ли удалить заказ (не должен быть в статусе in_progress или completed)
	if order.Status == models.OrderStatusInProgress {
		return fmt.Errorf("order service: нельзя удалить заказ в процессе выполнения")
	}

	return s.repo.Delete(ctx, orderID, clientID)
}

// GenerateOrderDescription генерирует описание заказа с помощью AI.
func (s *OrderService) GenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}
	return s.ai.GenerateOrderDescription(ctx, title, briefDescription, skills)
}

// StreamGenerateOrderDescription генерирует описание заказа потоково через AI.
func (s *OrderService) StreamGenerateOrderDescription(ctx context.Context, title, briefDescription string, skills []string, onDelta func(chunk string) error) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}
	return s.ai.StreamGenerateOrderDescription(ctx, title, briefDescription, skills, onDelta)
}

// GenerateProposal генерирует отклик на заказ с помощью AI.
func (s *OrderService) GenerateProposal(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, overrideSkills []string, overrideExperience string, overrideBio string, portfolioItems []models.PortfolioItemForAI) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return "", err
	}

	// Получаем требования заказа
	requirements, err := s.repo.ListRequirements(ctx, orderID)
	if err != nil {
		// Не критично, если не удалось получить требования
		requirements = []models.OrderRequirement{}
	}

	// Получаем профиль пользователя для использования его данных
	var userSkills []string
	var userExperience string
	var userBio string

	if s.profile != nil {
		profile, err := s.profile.GetProfile(ctx, userID)
		if err == nil && profile != nil {
			// Используем данные из профиля, если они есть
			userSkills = profile.Skills
			userExperience = profile.ExperienceLevel
			if profile.Bio != nil {
				userBio = *profile.Bio
			}
		}
	}

	// Переданные параметры имеют приоритет (переопределяют данные профиля)
	if len(overrideSkills) > 0 {
		userSkills = overrideSkills
	}
	if overrideExperience != "" {
		userExperience = overrideExperience
	}
	if overrideBio != "" {
		userBio = overrideBio
	}

	// Используем bio как дополнительный контекст для AI
	experienceStr := userExperience
	if userBio != "" {
		if experienceStr != "" {
			experienceStr += ". " + userBio
		} else {
			experienceStr = userBio
		}
	}

	// Преобразуем service.PortfolioItemForAI в структуру для AI клиента
	// Оба типа имеют одинаковую структуру, поэтому можем просто передать как interface{}
	// и преобразовать в AI клиенте
	aiItems := make([]struct {
		Title       string
		Description string
		AITags      []string
	}, len(portfolioItems))
	for i, item := range portfolioItems {
		aiItems[i] = struct {
			Title       string
			Description string
			AITags      []string
		}{
			Title:       item.Title,
			Description: item.Description,
			AITags:      item.AITags,
		}
	}

	return s.ai.GenerateProposal(ctx, order, requirements, userSkills, experienceStr, aiItems)
}

// StreamGenerateProposal генерирует отклик на заказ с помощью AI потоково.
func (s *OrderService) StreamGenerateProposal(
	ctx context.Context,
	orderID uuid.UUID,
	userID uuid.UUID,
	overrideSkills []string,
	overrideExperience string,
	overrideBio string,
	portfolioItems []models.PortfolioItemForAI,
	onDelta func(chunk string) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Получаем требования заказа
	requirements, err := s.repo.ListRequirements(ctx, orderID)
	if err != nil {
		requirements = []models.OrderRequirement{}
	}

	// Получаем профиль пользователя для использования его данных
	var userSkills []string
	var userExperience string
	var userBio string

	if s.profile != nil {
		profile, err := s.profile.GetProfile(ctx, userID)
		if err == nil && profile != nil {
			userSkills = profile.Skills
			userExperience = profile.ExperienceLevel
			if profile.Bio != nil {
				userBio = *profile.Bio
			}
		}
	}

	if len(overrideSkills) > 0 {
		userSkills = overrideSkills
	}
	if overrideExperience != "" {
		userExperience = overrideExperience
	}
	if overrideBio != "" {
		userBio = overrideBio
	}

	experienceStr := userExperience
	if userBio != "" {
		if experienceStr != "" {
			experienceStr += ". " + userBio
		} else {
			experienceStr = userBio
		}
	}

	aiItems := make([]struct {
		Title       string
		Description string
		AITags      []string
	}, len(portfolioItems))
	for i, item := range portfolioItems {
		aiItems[i] = struct {
			Title       string
			Description string
			AITags      []string
		}{
			Title:       item.Title,
			Description: item.Description,
			AITags:      item.AITags,
		}
	}

	return s.ai.StreamGenerateProposal(ctx, order, requirements, userSkills, experienceStr, aiItems, onDelta)
}

// ImproveOrderDescription улучшает описание заказа с помощью AI.
func (s *OrderService) ImproveOrderDescription(ctx context.Context, title, description string) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}
	return s.ai.ImproveOrderDescription(ctx, title, description)
}

// StreamImproveOrderDescription улучшает описание заказа потоково через AI.
func (s *OrderService) StreamImproveOrderDescription(ctx context.Context, title, description string, onDelta func(chunk string) error) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}
	return s.ai.StreamImproveOrderDescription(ctx, title, description, onDelta)
}

// RegenerateOrderSummary регенерирует AI summary для заказа.
func (s *OrderService) RegenerateOrderSummary(ctx context.Context, orderID uuid.UUID, clientID uuid.UUID) (*models.Order, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("order service: AI сервис недоступен")
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.ClientID != clientID {
		return nil, fmt.Errorf("order service: у вас нет прав на изменение этого заказа")
	}

	// Генерируем новый summary
	summary, err := s.ai.SummarizeOrder(ctx, order.Title, order.Description)
	if err != nil {
		return nil, fmt.Errorf("order service: не удалось сгенерировать summary: %w", err)
	}

	// Обновляем только ai_summary в базе
	if err := s.repo.UpdateAISummary(ctx, orderID, clientID, summary); err != nil {
		return nil, err
	}

	order.AISummary = &summary

	return order, nil
}

// StreamRegenerateOrderSummary регенерирует AI summary для заказа потоково.
func (s *OrderService) StreamRegenerateOrderSummary(
	ctx context.Context,
	orderID uuid.UUID,
	clientID uuid.UUID,
	onDelta func(chunk string) error,
) (*models.Order, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("order service: AI сервис недоступен")
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.ClientID != clientID {
		return nil, fmt.Errorf("order service: у вас нет прав на изменение этого заказа")
	}

	// Стримим новый summary
	var finalSummary string
	err = s.ai.StreamSummarizeOrder(ctx, order.Title, order.Description, func(chunk string) error {
		finalSummary += chunk
		return onDelta(chunk)
	})
	if err != nil {
		return nil, fmt.Errorf("order service: не удалось сгенерировать summary (stream): %w", err)
	}

	// После завершения сохраняем полный summary в базе
	if finalSummary != "" {
		if err := s.repo.UpdateAISummary(ctx, orderID, clientID, finalSummary); err != nil {
			return nil, err
		}
		order.AISummary = &finalSummary
	}

	return order, nil
}

// SummarizeConversation создаёт резюме переписки в чате.
func (s *OrderService) SummarizeConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (*models.ChatSummary, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("order service: AI сервис недоступен")
	}

	// Получаем чат
	conversation, err := s.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Проверяем права доступа
	if conversation.ClientID != userID && conversation.FreelancerID != userID {
		return nil, fmt.Errorf("order service: у вас нет доступа к этому чату")
	}

	// Получаем все сообщения
	messages, err := s.repo.ListMessages(ctx, conversationID, 1000, 0)
	if err != nil {
		return nil, err
	}

	// Получаем название заказа
	orderTitle := "Чат"
	if conversation.OrderID != nil {
		order, err := s.repo.GetByID(ctx, *conversation.OrderID)
		if err == nil && order != nil {
			orderTitle = order.Title
		}
	}

	return s.ai.SummarizeConversation(ctx, messages, orderTitle)
}

// StreamSummarizeConversation создаёт резюме переписки потоково.
func (s *OrderService) StreamSummarizeConversation(
	ctx context.Context,
	conversationID uuid.UUID,
	userID uuid.UUID,
	onDelta func(chunk string) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	conversation, err := s.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return err
	}

	if conversation.ClientID != userID && conversation.FreelancerID != userID {
		return fmt.Errorf("order service: у вас нет доступа к этому чату")
	}

	messages, err := s.repo.ListMessages(ctx, conversationID, 1000, 0)
	if err != nil {
		return err
	}

	orderTitle := "Чат"
	if conversation.OrderID != nil {
		order, err := s.repo.GetByID(ctx, *conversation.OrderID)
		if err == nil && order != nil {
			orderTitle = order.Title
		}
	}

	return s.ai.StreamSummarizeConversation(ctx, messages, orderTitle, onDelta)
}

// RecommendRelevantOrders рекомендует подходящие заказы для фрилансера.
// УЛУЧШЕНО: Теперь исключает заказы, на которые фрилансер уже откликнулся.
func (s *OrderService) RecommendRelevantOrders(ctx context.Context, freelancerID uuid.UUID, limit int) ([]models.RecommendedOrder, string, error) {
	if s.ai == nil {
		return nil, "", fmt.Errorf("order service: AI сервис недоступен")
	}

	// Получаем профиль фрилансера
	profile, err := s.profile.GetProfile(ctx, freelancerID)
	if err != nil {
		return nil, "", err
	}

	// Получаем портфолио
	portfolioItems, err := s.portfolio.List(ctx, freelancerID)
	if err != nil {
		return nil, "", err
	}

	// Преобразуем портфолио в формат для AI
	aiPortfolio := make([]models.PortfolioItemForAI, 0, len(portfolioItems))
	for _, item := range portfolioItems {
		desc := ""
		if item.Description != nil {
			desc = *item.Description
		}
		aiPortfolio = append(aiPortfolio, models.PortfolioItemForAI{
			Title:       item.Title,
			Description: desc,
			AITags:      item.AITags,
		})
	}

	// Получаем список всех опубликованных заказов (берем больше для лучшего выбора)
	searchLimit := limit * 5 // Берем в 5 раз больше заказов для анализа
	if searchLimit > 200 {
		searchLimit = 200 // Но не более 200
	}

	params := repository.ListFilterParams{
		Status: models.OrderStatusPublished,
		Limit:  searchLimit,
		Offset: 0,
	}
	result, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, "", err
	}

	if len(result.Orders) == 0 {
		return []models.RecommendedOrder{}, "Нет доступных заказов", nil
	}

	// Получаем список заказов, на которые фрилансер уже откликнулся
	myProposals, err := s.repo.ListMyProposals(ctx, freelancerID)
	if err != nil {
		// Если не удалось получить, продолжаем без фильтрации
		myProposals = []models.Proposal{}
	}

	// Собираем ID заказов, на которые уже откликнулся
	alreadyRespondedOrders := make(map[uuid.UUID]bool)
	for _, proposal := range myProposals {
		alreadyRespondedOrders[proposal.OrderID] = true
	}

	// Фильтруем заказы: исключаем те, на которые уже откликнулся
	filteredOrders := make([]models.Order, 0, len(result.Orders))
	for _, order := range result.Orders {
		// Пропускаем заказы, на которые уже откликнулся
		if alreadyRespondedOrders[order.ID] {
			continue
		}
		// Пропускаем заказы, где уже выбран исполнитель
		if order.Status == models.OrderStatusInProgress || order.Status == models.OrderStatusCompleted {
			continue
		}
		filteredOrders = append(filteredOrders, order)
	}

	if len(filteredOrders) == 0 {
		return []models.RecommendedOrder{}, "Нет новых заказов для рекомендации", nil
	}

	// Передаем отфильтрованные заказы в AI для анализа
	return s.ai.RecommendRelevantOrders(ctx, profile, aiPortfolio, filteredOrders)
}

// StreamRecommendRelevantOrders рекомендует подходящие заказы для фрилансера потоково.
func (s *OrderService) StreamRecommendRelevantOrders(
	ctx context.Context,
	freelancerID uuid.UUID,
	limit int,
	onDelta func(chunk string) error,
	onComplete func(recommendedOrders []models.RecommendedOrder, generalExplanation string) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	// Получаем профиль фрилансера
	profile, err := s.profile.GetProfile(ctx, freelancerID)
	if err != nil {
		return err
	}

	// Получаем портфолио
	portfolioItems, err := s.portfolio.List(ctx, freelancerID)
	if err != nil {
		return err
	}

	// Преобразуем портфолио в формат для AI
	aiPortfolio := make([]models.PortfolioItemForAI, 0, len(portfolioItems))
	for _, item := range portfolioItems {
		desc := ""
		if item.Description != nil {
			desc = *item.Description
		}
		aiPortfolio = append(aiPortfolio, models.PortfolioItemForAI{
			Title:       item.Title,
			Description: desc,
			AITags:      item.AITags,
		})
	}

	// Получаем список всех опубликованных заказов
	searchLimit := limit * 5
	if searchLimit > 200 {
		searchLimit = 200
	}

	params := repository.ListFilterParams{
		Status: models.OrderStatusPublished,
		Limit:  searchLimit,
		Offset: 0,
	}
	result, err := s.repo.List(ctx, params)
	if err != nil {
		return err
	}

	// Получаем список заказов, на которые фрилансер уже откликнулся
	myProposals, err := s.repo.ListMyProposals(ctx, freelancerID)
	if err != nil {
		// Если не удалось получить, продолжаем без фильтрации
		myProposals = []models.Proposal{}
	}

	// Собираем ID заказов, на которые уже откликнулся
	alreadyRespondedOrders := make(map[uuid.UUID]bool)
	for _, proposal := range myProposals {
		alreadyRespondedOrders[proposal.OrderID] = true
	}

	// Фильтруем заказы: исключаем те, на которые уже откликнулся
	filteredOrders := make([]models.Order, 0, len(result.Orders))
	for _, order := range result.Orders {
		// Пропускаем заказы, на которые уже откликнулся
		if alreadyRespondedOrders[order.ID] {
			continue
		}
		// Пропускаем заказы, где уже выбран исполнитель
		if order.Status == models.OrderStatusInProgress || order.Status == models.OrderStatusCompleted {
			continue
		}
		filteredOrders = append(filteredOrders, order)
	}

	if len(filteredOrders) == 0 {
		return onComplete([]models.RecommendedOrder{}, "Нет новых заказов для рекомендации")
	}

	return s.ai.StreamRecommendRelevantOrders(ctx, profile, aiPortfolio, filteredOrders, onDelta, onComplete)
}

// RecommendPriceAndTimeline рекомендует цену и сроки для отклика.
func (s *OrderService) RecommendPriceAndTimeline(
	ctx context.Context,
	orderID uuid.UUID,
	freelancerID uuid.UUID,
) (*models.PriceTimelineRecommendation, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("order service: AI сервис недоступен")
	}

	// Получаем заказ
	order, requirements, _, err := s.repo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Получаем профиль фрилансера
	profile, err := s.profile.GetProfile(ctx, freelancerID)
	if err != nil {
		return nil, err
	}

	// Получаем другие отклики
	proposals, err := s.repo.ListProposals(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Фильтруем отклики других фрилансеров
	otherProposals := make([]*models.Proposal, 0)
	for i := range proposals {
		if proposals[i].FreelancerID != freelancerID {
			otherProposals = append(otherProposals, &proposals[i])
		}
	}

	return s.ai.RecommendPriceAndTimeline(ctx, order, requirements, profile, otherProposals)
}

// StreamRecommendPriceAndTimeline рекомендует цену и сроки для отклика потоково.
func (s *OrderService) StreamRecommendPriceAndTimeline(
	ctx context.Context,
	orderID uuid.UUID,
	freelancerID uuid.UUID,
	onDelta func(chunk string) error,
	onComplete func(recommendation *models.PriceTimelineRecommendation) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	// Получаем заказ
	order, requirements, _, err := s.repo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return err
	}

	// Получаем профиль фрилансера
	profile, err := s.profile.GetProfile(ctx, freelancerID)
	if err != nil {
		return err
	}

	// Получаем другие отклики
	proposals, err := s.repo.ListProposals(ctx, orderID)
	if err != nil {
		return err
	}

	// Фильтруем отклики других фрилансеров
	otherProposals := make([]*models.Proposal, 0)
	for i := range proposals {
		if proposals[i].FreelancerID != freelancerID {
			otherProposals = append(otherProposals, &proposals[i])
		}
	}

	return s.ai.StreamRecommendPriceAndTimeline(ctx, order, requirements, profile, otherProposals, onDelta, onComplete)
}

// EvaluateOrderQuality оценивает качество заказа.
func (s *OrderService) EvaluateOrderQuality(ctx context.Context, orderID uuid.UUID, clientID uuid.UUID) (*models.OrderQualityEvaluation, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("order service: AI сервис недоступен")
	}

	order, requirements, _, err := s.repo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.ClientID != clientID {
		return nil, fmt.Errorf("order service: у вас нет прав на оценку этого заказа")
	}

	return s.ai.EvaluateOrderQuality(ctx, order, requirements)
}

// StreamEvaluateOrderQuality оценивает качество заказа потоково.
func (s *OrderService) StreamEvaluateOrderQuality(
	ctx context.Context,
	orderID uuid.UUID,
	clientID uuid.UUID,
	onDelta func(chunk string) error,
	onComplete func(evaluation *models.OrderQualityEvaluation) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	order, requirements, _, err := s.repo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return err
	}

	if order.ClientID != clientID {
		return fmt.Errorf("order service: у вас нет прав на оценку этого заказа")
	}

	return s.ai.StreamEvaluateOrderQuality(ctx, order, requirements, onDelta, onComplete)
}

// FindSuitableFreelancers находит подходящих фрилансеров для заказа.
// ИСПРАВЛЕНО: Теперь ищет среди ВСЕХ фрилансеров на платформе, а не только тех, кто уже откликнулся.
func (s *OrderService) FindSuitableFreelancers(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, userRole string, limit int) ([]models.SuitableFreelancer, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("order service: AI сервис недоступен")
	}

	if s.users == nil {
		return nil, fmt.Errorf("order service: UserRepository недоступен")
	}

	order, requirements, _, err := s.repo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order service: не удалось получить заказ: %w", err)
	}

	// Проверка прав: только владелец заказа или admin могут искать исполнителей
	// Нормализуем роль для сравнения (на случай разных регистров)
	isAdmin := strings.ToLower(userRole) == "admin"
	isOwner := order.ClientID == userID

	if !isAdmin && !isOwner {
		return nil, fmt.Errorf("order service: у вас нет прав на поиск исполнителей для этого заказа (заказ ID: %s, владелец: %s, ваш ID: %s, ваша роль: %s)",
			orderID, order.ClientID, userID, userRole)
	}

	// Получаем список фрилансеров, которые УЖЕ откликнулись на этот заказ (чтобы исключить их опционально)
	proposals, err := s.repo.ListProposals(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Собираем ID фрилансеров, которые уже откликнулись
	alreadyResponded := make(map[uuid.UUID]bool)
	for _, p := range proposals {
		alreadyResponded[p.FreelancerID] = true
	}

	// Получаем ВСЕХ активных фрилансеров с платформы
	// Используем больший лимит, чтобы AI мог выбрать лучших
	searchLimit := limit * 3 // Берем в 3 раза больше для лучшего выбора
	if searchLimit > 100 {
		searchLimit = 100 // Но не более 100
	}

	freelancerUsers, err := s.users.ListFreelancers(ctx, searchLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("order service: не удалось получить список фрилансеров: %w", err)
	}

	if len(freelancerUsers) == 0 {
		return []models.SuitableFreelancer{}, nil
	}

	// Получаем профили и портфолио всех фрилансеров
	freelancerProfiles := make([]*models.Profile, 0, len(freelancerUsers))
	freelancerPortfolios := make(map[uuid.UUID][]models.PortfolioItemForAI)

	for _, user := range freelancerUsers {
		// Пропускаем тех, кто уже откликнулся (опционально - можно убрать эту проверку)
		// if alreadyResponded[user.ID] {
		// 	continue
		// }

		profile, err := s.profile.GetProfile(ctx, user.ID)
		if err != nil {
			continue // Пропускаем, если нет профиля
		}
		freelancerProfiles = append(freelancerProfiles, profile)

		portfolioItems, err := s.portfolio.List(ctx, user.ID)
		if err == nil {
			aiPortfolio := make([]models.PortfolioItemForAI, 0, len(portfolioItems))
			for _, item := range portfolioItems {
				desc := ""
				if item.Description != nil {
					desc = *item.Description
				}
				aiPortfolio = append(aiPortfolio, models.PortfolioItemForAI{
					Title:       item.Title,
					Description: desc,
					AITags:      item.AITags,
				})
			}
			freelancerPortfolios[user.ID] = aiPortfolio
		}
	}

	if len(freelancerProfiles) == 0 {
		return []models.SuitableFreelancer{}, nil
	}

	// Передаем в AI для анализа и выбора лучших
	return s.ai.FindSuitableFreelancers(ctx, order, requirements, freelancerProfiles, freelancerPortfolios)
}

// StreamFindSuitableFreelancers находит подходящих фрилансеров для заказа потоково.
func (s *OrderService) StreamFindSuitableFreelancers(
	ctx context.Context,
	orderID uuid.UUID,
	userID uuid.UUID,
	userRole string,
	limit int,
	onDelta func(chunk string) error,
	onComplete func(data []models.SuitableFreelancer) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	if s.users == nil {
		return fmt.Errorf("order service: UserRepository недоступен")
	}

	order, requirements, _, err := s.repo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order service: не удалось получить заказ: %w", err)
	}

	// Проверка прав: только владелец заказа или admin могут искать исполнителей
	// Нормализуем роль для сравнения (на случай разных регистров)
	isAdmin := strings.ToLower(userRole) == "admin"
	isOwner := order.ClientID == userID

	if !isAdmin && !isOwner {
		return fmt.Errorf("order service: у вас нет прав на поиск исполнителей для этого заказа (заказ ID: %s, владелец: %s, ваш ID: %s, ваша роль: %s)",
			orderID, order.ClientID, userID, userRole)
	}

	// Получаем список фрилансеров, которые УЖЕ откликнулись на этот заказ
	proposals, err := s.repo.ListProposals(ctx, orderID)
	if err != nil {
		return err
	}

	// Собираем ID фрилансеров, которые уже откликнулись
	alreadyResponded := make(map[uuid.UUID]bool)
	for _, p := range proposals {
		alreadyResponded[p.FreelancerID] = true
	}

	// Получаем ВСЕХ активных фрилансеров с платформы
	searchLimit := limit * 3
	if searchLimit > 100 {
		searchLimit = 100
	}

	freelancerUsers, err := s.users.ListFreelancers(ctx, searchLimit, 0)
	if err != nil {
		return fmt.Errorf("order service: не удалось получить список фрилансеров: %w", err)
	}

	if len(freelancerUsers) == 0 {
		return onComplete([]models.SuitableFreelancer{})
	}

	// Получаем профили и портфолио всех фрилансеров
	freelancerProfiles := make([]*models.Profile, 0, len(freelancerUsers))
	freelancerPortfolios := make(map[uuid.UUID][]models.PortfolioItemForAI)

	for _, user := range freelancerUsers {
		profile, err := s.profile.GetProfile(ctx, user.ID)
		if err != nil {
			continue
		}
		freelancerProfiles = append(freelancerProfiles, profile)

		portfolioItems, err := s.portfolio.List(ctx, user.ID)
		if err == nil {
			aiPortfolio := make([]models.PortfolioItemForAI, 0, len(portfolioItems))
			for _, item := range portfolioItems {
				desc := ""
				if item.Description != nil {
					desc = *item.Description
				}
				aiPortfolio = append(aiPortfolio, models.PortfolioItemForAI{
					Title:       item.Title,
					Description: desc,
					AITags:      item.AITags,
				})
			}
			freelancerPortfolios[user.ID] = aiPortfolio
		}
	}

	if len(freelancerProfiles) == 0 {
		return onComplete([]models.SuitableFreelancer{})
	}

	return s.ai.StreamFindSuitableFreelancers(ctx, order, requirements, freelancerProfiles, freelancerPortfolios, onDelta, onComplete)
}

// AIChatAssistant обрабатывает запросы к AI помощнику.
func (s *OrderService) AIChatAssistant(
	ctx context.Context,
	userID uuid.UUID,
	userMessage string,
	userRole string,
	contextData map[string]interface{},
) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}

	return s.ai.AIChatAssistant(ctx, userMessage, userRole, contextData)
}

// StreamAIChatAssistant обрабатывает запросы к AI помощнику потоково.
func (s *OrderService) StreamAIChatAssistant(
	ctx context.Context,
	userID uuid.UUID,
	userMessage string,
	userRole string,
	contextData map[string]interface{},
	onDelta func(chunk string) error,
) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}

	return s.ai.StreamAIChatAssistant(ctx, userMessage, userRole, contextData, onDelta)
}

// ImproveProfile улучшает описание профиля с помощью AI.
func (s *OrderService) ImproveProfile(ctx context.Context, currentBio string, skills []string, experienceLevel string) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}

	return s.ai.ImproveProfile(ctx, currentBio, skills, experienceLevel)
}

// StreamImproveProfile улучшает описание профиля потоково через AI.
func (s *OrderService) StreamImproveProfile(ctx context.Context, currentBio string, skills []string, experienceLevel string, onDelta func(chunk string) error) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}
	return s.ai.StreamImproveProfile(ctx, currentBio, skills, experienceLevel, onDelta)
}

// ImprovePortfolioItem улучшает описание работы в портфолио с помощью AI.
func (s *OrderService) ImprovePortfolioItem(ctx context.Context, title, description string, aiTags []string) (string, error) {
	if s.ai == nil {
		return "", fmt.Errorf("order service: AI сервис недоступен")
	}

	return s.ai.ImprovePortfolioItem(ctx, title, description, aiTags)
}

// StreamImprovePortfolioItem улучшает описание работы в портфолио потоково через AI.
func (s *OrderService) StreamImprovePortfolioItem(ctx context.Context, title, description string, aiTags []string, onDelta func(chunk string) error) error {
	if s.ai == nil {
		return fmt.Errorf("order service: AI сервис недоступен")
	}
	return s.ai.StreamImprovePortfolioItem(ctx, title, description, aiTags, onDelta)
}
