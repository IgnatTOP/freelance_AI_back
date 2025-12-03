package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// ExtendedSeedService генерирует реалистичные данные для тестирования.
type ExtendedSeedService struct {
	userRepo             *repository.UserRepository
	orderRepo            *repository.OrderRepository
	paymentRepo          *repository.PaymentRepository
	reviewRepo           *repository.ReviewRepository
	favoriteRepo         *repository.FavoriteRepository
	proposalTemplateRepo *repository.ProposalTemplateRepository
}

func NewExtendedSeedService(
	userRepo *repository.UserRepository,
	orderRepo *repository.OrderRepository,
	paymentRepo *repository.PaymentRepository,
	reviewRepo *repository.ReviewRepository,
	favoriteRepo *repository.FavoriteRepository,
	proposalTemplateRepo *repository.ProposalTemplateRepository,
) *ExtendedSeedService {
	return &ExtendedSeedService{
		userRepo:             userRepo,
		orderRepo:            orderRepo,
		paymentRepo:          paymentRepo,
		reviewRepo:           reviewRepo,
		favoriteRepo:         favoriteRepo,
		proposalTemplateRepo: proposalTemplateRepo,
	}
}

type ExtendedSeedResult struct {
	Accounts         []SeedAccountInfo `json:"accounts"`
	OrdersCreated    int               `json:"orders_created"`
	ProposalsCreated int               `json:"proposals_created"`
	ReviewsCreated   int               `json:"reviews_created"`
	MessagesCreated  int               `json:"messages_created"`
}

var (
	extRussianFirstNames = []string{
		"Александр", "Дмитрий", "Максим", "Сергей", "Андрей", "Алексей", "Артём", "Илья",
		"Иван", "Михаил", "Никита", "Роман", "Егор", "Павел", "Владимир", "Константин",
		"Анна", "Мария", "Елена", "Ольга", "Татьяна", "Наталья", "Ирина", "Светлана",
		"Екатерина", "Юлия", "Анастасия", "Дарья", "Виктория", "Полина", "София", "Алиса",
	}
	extRussianLastNames = []string{
		"Иванов", "Петров", "Смирнов", "Козлов", "Соколов", "Попов", "Лебедев", "Новиков",
		"Морозов", "Волков", "Соловьёв", "Васильев", "Зайцев", "Павлов", "Семёнов", "Голубев",
	}
	extRussianCities = []string{
		"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань", "Нижний Новгород",
		"Челябинск", "Самара", "Омск", "Ростов-на-Дону", "Уфа", "Красноярск", "Воронеж",
	}
	extTechSkills = []string{
		"JavaScript", "TypeScript", "React", "Vue.js", "Angular", "Node.js", "Python", "Go",
		"Java", "PHP", "PostgreSQL", "MySQL", "MongoDB", "Docker", "AWS", "Git",
	}
	extDesignSkills   = []string{"Figma", "Adobe XD", "Photoshop", "UI/UX Design", "Web Design"}
	extMarketingSkills = []string{"SEO", "SMM", "Контекстная реклама", "Копирайтинг"}
)


var freelancerBios = []string{
	"Full-stack разработчик с 5+ годами опыта. Специализируюсь на React и Node.js. Создаю качественные веб-приложения под ключ.",
	"Backend разработчик, работаю с Go и Python. Опыт построения высоконагруженных систем и микросервисной архитектуры.",
	"Frontend разработчик, эксперт в React/TypeScript. Создаю современные, быстрые и отзывчивые интерфейсы.",
	"Мобильный разработчик (iOS/Android). Работаю с React Native и Flutter. Более 20 опубликованных приложений.",
	"DevOps инженер. Настройка CI/CD, Docker, Kubernetes, AWS. Автоматизация всех процессов разработки.",
	"UI/UX дизайнер с опытом 7 лет. Создаю интуитивные интерфейсы, которые любят пользователи.",
	"Data Scientist. Машинное обучение, анализ данных, Python. Помогу извлечь ценность из ваших данных.",
	"WordPress разработчик. Создание сайтов, интернет-магазинов, кастомных плагинов и тем.",
	"QA инженер. Ручное и автоматизированное тестирование. Selenium, Cypress, Jest.",
	"SEO специалист. Продвижение сайтов в поисковых системах, аудит, оптимизация.",
}

var clientBios = []string{
	"Владелец интернет-магазина. Ищу надёжных исполнителей для развития бизнеса.",
	"Стартап в сфере EdTech. Создаём платформу онлайн-обучения.",
	"Маркетинговое агентство. Регулярно ищем фрилансеров для проектов клиентов.",
	"IT компания. Аутсорсим часть задач на фриланс.",
	"Предприниматель. Развиваю несколько онлайн-проектов.",
}

var orderTitles = []string{
	"Разработка интернет-магазина на React",
	"Создание мобильного приложения для доставки",
	"Дизайн лендинга для стартапа",
	"Разработка CRM системы",
	"Настройка CI/CD для проекта",
	"Создание Telegram бота",
	"Редизайн корпоративного сайта",
	"Разработка REST API",
	"Интеграция платёжной системы",
	"Создание админ-панели",
	"Оптимизация производительности сайта",
	"Разработка системы бронирования",
	"Создание дашборда аналитики",
	"Миграция на новый сервер",
	"Разработка чат-бота поддержки",
}

var orderDescriptions = []string{
	"Нужен современный интернет-магазин с каталогом товаров, корзиной, личным кабинетом и интеграцией с платёжными системами. Дизайн уже есть в Figma.",
	"Требуется разработать мобильное приложение для iOS и Android. Основной функционал: авторизация, каталог, корзина, отслеживание заказа на карте.",
	"Нужен продающий лендинг для нового продукта. Требуется адаптивный дизайн, анимации, форма заявки с интеграцией в CRM.",
	"Разработка CRM для отдела продаж: управление клиентами, сделками, задачами. Интеграция с телефонией и почтой.",
	"Настроить автоматическую сборку, тестирование и деплой проекта. Стек: Node.js, PostgreSQL, Docker.",
	"Создать Telegram бота для автоматизации работы с клиентами. Приём заявок, ответы на FAQ, уведомления.",
	"Полный редизайн корпоративного сайта. Нужен современный вид, улучшенная навигация, мобильная версия.",
	"Разработать REST API для мобильного приложения. Авторизация JWT, CRUD операции, документация Swagger.",
	"Интегрировать Stripe и ЮKassa в существующий сайт. Обработка платежей, подписки, возвраты.",
	"Создать админ-панель для управления контентом сайта. Редактирование страниц, загрузка медиа, статистика.",
}

var proposalTexts = []string{
	"Здравствуйте! Внимательно изучил ваше ТЗ. Имею большой опыт в подобных проектах. Готов приступить сразу после обсуждения деталей.",
	"Добрый день! Заинтересовал ваш проект. Работал над похожими задачами, могу показать примеры. Предлагаю созвониться для обсуждения.",
	"Приветствую! Это именно тот тип проектов, в которых я специализируюсь. Гарантирую качественный результат в срок.",
	"Здравствуйте! Готов взяться за ваш проект. Есть релевантный опыт и все необходимые навыки. Давайте обсудим детали.",
	"Добрый день! Проект интересный, готов предложить оптимальное решение. Работаю по договору, даю гарантию на код.",
}

var reviewComments = []string{
	"Отличная работа! Всё сделано качественно и в срок. Рекомендую!",
	"Профессиональный подход, хорошая коммуникация. Буду обращаться ещё.",
	"Выполнил всё по ТЗ, оперативно вносил правки. Спасибо!",
	"Хороший специалист, разбирается в своём деле. Результатом доволен.",
	"Работа выполнена на высоком уровне. Всегда на связи, отвечает быстро.",
	"Рекомендую данного исполнителя. Качественно, быстро, по адекватной цене.",
	"Всё супер! Проект сдан раньше срока, качество отличное.",
	"Приятно работать с профессионалом. Обязательно обращусь снова.",
}

var chatMessages = []string{
	"Здравствуйте! Готов обсудить детали проекта.",
	"Добрый день! Когда удобно созвониться?",
	"Отправил макеты на согласование, посмотрите пожалуйста.",
	"Сделал первую версию, жду обратную связь.",
	"Внёс правки по вашим комментариям.",
	"Как продвигается работа?",
	"Всё идёт по плану, завтра покажу результат.",
	"Отлично, спасибо за оперативность!",
	"Есть пара вопросов по ТЗ, можем обсудить?",
	"Да, конечно, слушаю.",
	"Проект готов, можете проверять.",
	"Проверил, всё отлично! Принимаю работу.",
}

var proposalTemplates = []struct {
	Title   string
	Content string
}{
	{"Стандартный отклик", "Здравствуйте!\n\nВнимательно изучил ваш проект. Имею релевантный опыт и готов приступить к работе.\n\nПредлагаю обсудить детали в личных сообщениях.\n\nС уважением"},
	{"Для срочных проектов", "Добрый день!\n\nВижу, что проект срочный. Могу начать работу сегодня и уложиться в ваши сроки.\n\nДавайте обсудим детали."},
	{"С портфолио", "Здравствуйте!\n\nЗаинтересовал ваш проект. Работал над похожими задачами, вот примеры моих работ: [ссылки]\n\nГотов обсудить детали и ответить на вопросы."},
}

func (s *ExtendedSeedService) SeedRealisticData(ctx context.Context) (*ExtendedSeedResult, error) {
	rand.Seed(time.Now().UnixNano())
	result := &ExtendedSeedResult{}

	// 1. Создаём пользователей
	users, accounts, err := s.createUsers(ctx, 15)
	if err != nil {
		return nil, fmt.Errorf("create users: %w", err)
	}
	result.Accounts = accounts

	var clients, freelancers []*models.User
	for _, u := range users {
		if u.Role == "client" {
			clients = append(clients, u)
		} else {
			freelancers = append(freelancers, u)
		}
	}

	// 2. Пополняем балансы клиентов
	for _, client := range clients {
		amount := float64(rand.Intn(100000) + 50000)
		s.paymentRepo.Deposit(ctx, client.ID, amount, "Пополнение баланса")
	}

	// 3. Создаём шаблоны откликов для фрилансеров
	for _, f := range freelancers {
		for _, tmpl := range proposalTemplates {
			t := &models.ProposalTemplate{UserID: f.ID, Title: tmpl.Title, Content: tmpl.Content}
			s.proposalTemplateRepo.Create(ctx, t)
		}
	}

	// 4. Создаём заказы
	orders, err := s.createOrders(ctx, clients, 20)
	if err != nil {
		return nil, fmt.Errorf("create orders: %w", err)
	}
	result.OrdersCreated = len(orders)

	// 5. Создаём отклики и принимаем некоторые
	proposalCount, err := s.createProposalsAndAccept(ctx, orders, freelancers)
	if err != nil {
		return nil, fmt.Errorf("create proposals: %w", err)
	}
	result.ProposalsCreated = proposalCount

	// 6. Завершаем часть заказов и создаём отзывы
	reviewCount, err := s.completeOrdersAndReview(ctx, orders, freelancers)
	if err != nil {
		return nil, fmt.Errorf("complete orders: %w", err)
	}
	result.ReviewsCreated = reviewCount

	// 7. Добавляем избранное
	s.createFavorites(ctx, users, orders, freelancers)

	return result, nil
}



func (s *ExtendedSeedService) createUsers(ctx context.Context, count int) ([]*models.User, []SeedAccountInfo, error) {
	var users []*models.User
	var accounts []SeedAccountInfo
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	domains := []string{"gmail.com", "yandex.ru", "mail.ru"}

	for i := 0; i < count; i++ {
		firstName := extRussianFirstNames[rand.Intn(len(extRussianFirstNames))]
		lastName := extRussianLastNames[rand.Intn(len(extRussianLastNames))]
		username := fmt.Sprintf("%s_%s_%d", toLatin(firstName), toLatin(lastName), rand.Intn(1000))
		email := fmt.Sprintf("%s.%s%d@%s", toLatin(firstName), toLatin(lastName), rand.Intn(100), domains[rand.Intn(len(domains))])

		role := "freelancer"
		if i < 5 { // первые 5 - клиенты
			role = "client"
		}

		user := &models.User{
			Email:        email,
			Username:     username,
			PasswordHash: string(passwordHash),
			Role:         role,
			IsActive:     true,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			continue // пропускаем дубликаты
		}

		// Профиль
		displayName := fmt.Sprintf("%s %s", firstName, lastName)
		location := extRussianCities[rand.Intn(len(extRussianCities))]
		var bio string
		var skills []string
		var hourlyRate *float64

		if role == "freelancer" {
			bio = freelancerBios[rand.Intn(len(freelancerBios))]
			skills = s.randomSkills(5, 10)
			rate := float64(1000 + rand.Intn(4000))
			hourlyRate = &rate
		} else {
			bio = clientBios[rand.Intn(len(clientBios))]
		}

		expLevels := []string{"junior", "middle", "senior"}
		profile := &models.Profile{
			UserID:          user.ID,
			DisplayName:     displayName,
			Bio:             &bio,
			Location:        &location,
			Skills:          skills,
			HourlyRate:      hourlyRate,
			ExperienceLevel: expLevels[rand.Intn(len(expLevels))],
		}
		s.userRepo.UpsertProfile(ctx, profile)

		users = append(users, user)
		accounts = append(accounts, SeedAccountInfo{
			Email:    email,
			Username: username,
			Password: "Password123",
			Role:     role,
		})
	}
	return users, accounts, nil
}

func (s *ExtendedSeedService) randomSkills(min, max int) []string {
	allSkills := append(append(extTechSkills, extDesignSkills...), extMarketingSkills...)
	count := min + rand.Intn(max-min+1)
	if count > len(allSkills) {
		count = len(allSkills)
	}
	rand.Shuffle(len(allSkills), func(i, j int) { allSkills[i], allSkills[j] = allSkills[j], allSkills[i] })
	return allSkills[:count]
}

func (s *ExtendedSeedService) createOrders(ctx context.Context, clients []*models.User, count int) ([]*models.Order, error) {
	var orders []*models.Order
	statuses := []string{models.OrderStatusPublished, models.OrderStatusPublished, models.OrderStatusPublished, models.OrderStatusInProgress, models.OrderStatusCompleted}

	for i := 0; i < count; i++ {
		client := clients[rand.Intn(len(clients))]
		title := orderTitles[rand.Intn(len(orderTitles))]
		desc := orderDescriptions[rand.Intn(len(orderDescriptions))]

		budgetMin := float64(10000 + rand.Intn(40000))
		budgetMax := budgetMin + float64(rand.Intn(30000))
		deadline := time.Now().Add(time.Duration(7+rand.Intn(30)) * 24 * time.Hour)

		order := &models.Order{
			ClientID:    client.ID,
			Title:       title,
			Description: desc,
			BudgetMin:   &budgetMin,
			BudgetMax:   &budgetMax,
			Status:      statuses[rand.Intn(len(statuses))],
			DeadlineAt:  &deadline,
		}

		skills := s.randomSkills(2, 5)
		var reqs []models.OrderRequirement
		for _, sk := range skills {
			reqs = append(reqs, models.OrderRequirement{Skill: sk, Level: "middle"})
		}

		if err := s.orderRepo.Create(ctx, order, reqs, nil); err != nil {
			continue
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *ExtendedSeedService) createProposalsAndAccept(ctx context.Context, orders []*models.Order, freelancers []*models.User) (int, error) {
	count := 0
	for _, order := range orders {
		if order.Status == models.OrderStatusDraft {
			continue
		}
		// 2-4 отклика на заказ
		numProposals := 2 + rand.Intn(3)
		usedFreelancers := make(map[uuid.UUID]bool)

		for j := 0; j < numProposals && j < len(freelancers); j++ {
			f := freelancers[rand.Intn(len(freelancers))]
			if usedFreelancers[f.ID] || f.ID == order.ClientID {
				continue
			}
			usedFreelancers[f.ID] = true

			price := *order.BudgetMin + float64(rand.Intn(int(*order.BudgetMax-*order.BudgetMin)))
			text := proposalTexts[rand.Intn(len(proposalTexts))]

			proposal := &models.Proposal{
				OrderID:        order.ID,
				FreelancerID:   f.ID,
				ProposedAmount: &price,
				CoverLetter:    text,
				Status:         models.ProposalStatusPending,
			}
			if err := s.orderRepo.CreateProposal(ctx, proposal); err != nil {
				continue
			}
			count++

			// Принимаем первый отклик для заказов in_progress
			if j == 0 && order.Status == models.OrderStatusInProgress {
				s.orderRepo.UpdateProposalStatus(ctx, proposal.ID, models.ProposalStatusAccepted)
				order.FreelancerID = &f.ID
			}
		}
	}
	return count, nil
}

func (s *ExtendedSeedService) completeOrdersAndReview(ctx context.Context, orders []*models.Order, freelancers []*models.User) (int, error) {
	reviewCount := 0
	for _, order := range orders {
		if order.Status != models.OrderStatusCompleted {
			continue
		}
		// Находим принятый отклик
		proposals, _ := s.orderRepo.ListProposals(ctx, order.ID)
		var acceptedFreelancer uuid.UUID
		for _, p := range proposals {
			if p.Status == models.ProposalStatusAccepted {
				acceptedFreelancer = p.FreelancerID
				break
			}
		}
		if acceptedFreelancer == uuid.Nil && len(freelancers) > 0 {
			// Принимаем случайный отклик
			f := freelancers[rand.Intn(len(freelancers))]
			acceptedFreelancer = f.ID
		}
		if acceptedFreelancer == uuid.Nil {
			continue
		}

		// Отзыв от клиента
		rating := 4 + rand.Intn(2) // 4-5
		comment := reviewComments[rand.Intn(len(reviewComments))]
		clientReview := &models.Review{
			OrderID:    order.ID,
			ReviewerID: order.ClientID,
			ReviewedID: acceptedFreelancer,
			Rating:     rating,
			Comment:    &comment,
		}
		if err := s.reviewRepo.Create(ctx, clientReview); err == nil {
			reviewCount++
		}

		// Отзыв от фрилансера
		comment2 := "Приятный заказчик, чёткое ТЗ, быстрая оплата. Рекомендую!"
		freelancerReview := &models.Review{
			OrderID:    order.ID,
			ReviewerID: acceptedFreelancer,
			ReviewedID: order.ClientID,
			Rating:     5,
			Comment:    &comment2,
		}
		if err := s.reviewRepo.Create(ctx, freelancerReview); err == nil {
			reviewCount++
		}
	}
	return reviewCount, nil
}

func (s *ExtendedSeedService) createFavorites(ctx context.Context, users []*models.User, orders []*models.Order, freelancers []*models.User) {
	for _, u := range users {
		// Добавляем 2-3 заказа в избранное
		for i := 0; i < 2+rand.Intn(2); i++ {
			if len(orders) > 0 {
				order := orders[rand.Intn(len(orders))]
				s.favoriteRepo.Add(ctx, u.ID, models.FavoriteTypeOrder, order.ID)
			}
		}
		// Клиенты добавляют фрилансеров в избранное
		if u.Role == "client" && len(freelancers) > 0 {
			f := freelancers[rand.Intn(len(freelancers))]
			s.favoriteRepo.Add(ctx, u.ID, models.FavoriteTypeFreelancer, f.ID)
		}
	}
}
