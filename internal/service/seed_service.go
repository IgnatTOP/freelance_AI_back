package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ignatzorin/freelance-backend/internal/models"
	"github.com/ignatzorin/freelance-backend/internal/repository"
)

// SeedService генерирует фейковые данные для тестирования.
type SeedService struct {
	userRepo  *repository.UserRepository
	orderRepo *repository.OrderRepository
}

// NewSeedService создаёт новый сервис для генерации данных.
func NewSeedService(userRepo *repository.UserRepository, orderRepo *repository.OrderRepository) *SeedService {
	return &SeedService{
		userRepo:  userRepo,
		orderRepo: orderRepo,
	}
}

// SeedData генерирует фейковые профили и заказы.
func (s *SeedService) SeedData(ctx context.Context, numUsers int, numOrders int) error {
	rand.Seed(time.Now().UnixNano())

	// Генерируем пользователей
	users, err := s.generateUsers(ctx, numUsers)
	if err != nil {
		return fmt.Errorf("seed service: failed to generate users: %w", err)
	}

	// Разделяем на клиентов и фрилансеров
	var clients []*models.User
	var freelancers []*models.User
	for _, user := range users {
		if user.Role == "client" {
			clients = append(clients, user)
		} else {
			freelancers = append(freelancers, user)
		}
	}

	// Генерируем заказы
	if err := s.generateOrders(ctx, clients, numOrders); err != nil {
		return fmt.Errorf("seed service: failed to generate orders: %w", err)
	}

	return nil
}

// generateUsers создаёт фейковых пользователей с профилями.
func (s *SeedService) generateUsers(ctx context.Context, count int) ([]*models.User, error) {
	firstNames := []string{
		"Александр", "Дмитрий", "Максим", "Сергей", "Андрей", "Алексей", "Артём", "Илья",
		"Иван", "Михаил", "Никита", "Роман", "Егор", "Павел", "Владимир", "Константин",
		"Анна", "Мария", "Елена", "Ольга", "Татьяна", "Наталья", "Ирина", "Светлана",
		"Екатерина", "Юлия", "Анастасия", "Дарья", "Виктория", "Полина", "София", "Алиса",
	}
	lastNames := []string{
		"Иванов", "Петров", "Смирнов", "Козлов", "Соколов", "Попов", "Лебедев", "Новиков",
		"Морозов", "Волков", "Соловьёв", "Васильев", "Зайцев", "Павлов", "Семёнов", "Голубев",
		"Виноградов", "Богданов", "Воробьёв", "Фёдоров", "Михайлов", "Белов", "Тарасов", "Беляев",
	}
	domains := []string{"gmail.com", "yandex.ru", "mail.ru", "outlook.com", "yahoo.com"}

	skills := []string{
		"JavaScript", "TypeScript", "React", "Vue.js", "Angular", "Node.js", "Python", "Go",
		"Java", "C++", "C#", "PHP", "Ruby", "Swift", "Kotlin", "Dart", "Flutter", "React Native",
		"HTML", "CSS", "SASS", "LESS", "Webpack", "Vite", "Docker", "Kubernetes", "AWS", "Azure",
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "GraphQL", "REST API", "Git", "CI/CD",
		"Figma", "Adobe XD", "Photoshop", "Illustrator", "UI/UX Design", "Mobile Design",
		"SEO", "Marketing", "Content Writing", "Translation", "Data Analysis", "Machine Learning",
	}

	locations := []string{
		"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань", "Нижний Новгород",
		"Челябинск", "Самара", "Омск", "Ростов-на-Дону", "Уфа", "Красноярск", "Воронеж",
		"Пермь", "Волгоград", "Краснодар", "Саратов", "Тюмень", "Тольятти", "Ижевск",
	}

	bios := []string{
		"Опытный разработчик с более чем 5 годами опыта в веб-разработке. Специализируюсь на создании современных и отзывчивых веб-приложений.",
		"Full-stack разработчик, увлечённый созданием качественных продуктов. Работаю с современными технологиями и фреймворками.",
		"Дизайнер интерфейсов с фокусом на пользовательский опыт. Создаю интуитивные и красивые интерфейсы для веб и мобильных приложений.",
		"Backend разработчик, специализирующийся на высоконагруженных системах. Опыт работы с микросервисной архитектурой.",
		"Frontend разработчик с глубокими знаниями React и TypeScript. Создаю производительные и масштабируемые приложения.",
		"Мобильный разработчик с опытом создания нативных и кроссплатформенных приложений. Работаю с iOS и Android.",
		"DevOps инженер, специализирующийся на автоматизации и развёртывании. Опыт работы с облачными платформами.",
		"Data Scientist с опытом работы с большими данными и машинным обучением. Создаю модели для решения бизнес-задач.",
		"QA инженер с опытом автоматизации тестирования. Обеспечиваю качество программного обеспечения на всех этапах разработки.",
		"Контент-менеджер и копирайтер. Создаю качественный контент для веб-сайтов, блогов и социальных сетей.",
	}

	experienceLevels := []string{
		models.ExperienceLevelJunior,
		models.ExperienceLevelMiddle,
		models.ExperienceLevelSenior,
	}

	var users []*models.User
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)

	for i := 0; i < count; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		username := fmt.Sprintf("%s_%s_%d", firstName, lastName, rand.Intn(10000))
		email := fmt.Sprintf("%s.%s.%d@%s",
			toLatin(firstName), toLatin(lastName), rand.Intn(10000), domains[rand.Intn(len(domains))])

		var role string
		if i%3 == 0 { // 1/3 клиентов, 2/3 фрилансеров
			role = "client"
		} else {
			role = "freelancer"
		}

		user := &models.User{
			Email:        email,
			Username:     username,
			PasswordHash: string(passwordHash),
			Role:         role,
			IsActive:     true,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Создаём профиль
		displayName := fmt.Sprintf("%s %s", firstName, lastName)
		numSkills := rand.Intn(8) + 3 // 3-10 навыков
		userSkills := make([]string, 0, numSkills)
		skillMap := make(map[string]bool)
		for len(userSkills) < numSkills {
			skill := skills[rand.Intn(len(skills))]
			if !skillMap[skill] {
				userSkills = append(userSkills, skill)
				skillMap[skill] = true
			}
		}

		bio := bios[rand.Intn(len(bios))]
		location := locations[rand.Intn(len(locations))]
		experienceLevel := experienceLevels[rand.Intn(len(experienceLevels))]

		var hourlyRate *float64
		if role == "freelancer" {
			rate := float64(rand.Intn(5000)+500) / 100.0 // 5-55 USD
			hourlyRate = &rate
		}

		profile := &models.Profile{
			UserID:          user.ID,
			DisplayName:     displayName,
			Bio:             &bio,
			HourlyRate:      hourlyRate,
			ExperienceLevel: experienceLevel,
			Skills:          userSkills,
			Location:        &location,
		}

		if err := s.userRepo.UpsertProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to create profile: %w", err)
		}

		users = append(users, user)
	}

	return users, nil
}

// generateOrders создаёт фейковые заказы.
func (s *SeedService) generateOrders(ctx context.Context, clients []*models.User, count int) error {
	if len(clients) == 0 {
		return fmt.Errorf("no clients available to create orders")
	}

	titles := []string{
		"Разработка веб-сайта для интернет-магазина",
		"Создание мобильного приложения для доставки еды",
		"Дизайн логотипа и фирменного стиля",
		"Настройка и оптимизация базы данных",
		"Разработка REST API для мобильного приложения",
		"Создание landing page для стартапа",
		"Интеграция платежной системы",
		"Разработка админ-панели для управления контентом",
		"Создание чат-бота для поддержки клиентов",
		"Разработка дашборда для аналитики",
		"Миграция сайта на новый фреймворк",
		"Оптимизация производительности веб-приложения",
		"Создание системы бронирования",
		"Разработка плагина для WordPress",
		"Настройка CI/CD pipeline",
		"Создание системы уведомлений",
		"Разработка мобильного приложения для фитнеса",
		"Дизайн пользовательского интерфейса",
		"Создание системы управления задачами",
		"Разработка API для интеграции с внешними сервисами",
		"Создание онлайн-калькулятора",
		"Разработка системы рейтингов и отзывов",
		"Создание системы поиска с фильтрами",
		"Разработка модуля электронной коммерции",
		"Создание системы аналитики и отчетности",
		"Разработка системы авторизации и безопасности",
		"Создание мобильного приложения для социальной сети",
		"Дизайн и разработка корпоративного сайта",
		"Создание системы управления документами",
		"Разработка системы онлайн-обучения",
	}

	descriptions := []string{
		"Требуется разработка современного веб-сайта с адаптивным дизайном. Необходимо реализовать каталог товаров, корзину, систему оплаты и личный кабинет пользователя.",
		"Нужно создать мобильное приложение для iOS и Android с использованием React Native. Приложение должно включать авторизацию, карту с геолокацией и систему уведомлений.",
		"Требуется разработать фирменный стиль компании: логотип, цветовая палитра, типографика. Также нужны шаблоны для визиток, бланков и презентаций.",
		"Необходимо оптимизировать существующую базу данных PostgreSQL, улучшить производительность запросов и настроить репликацию.",
		"Требуется разработать REST API для мобильного приложения. API должно поддерживать авторизацию, работу с пользователями, заказами и уведомлениями.",
		"Нужно создать одностраничный сайт (landing page) для продвижения нового продукта. Требуется современный дизайн, анимации и интеграция с формами обратной связи.",
		"Требуется интегрировать платежную систему (Stripe/PayPal) в существующее веб-приложение. Необходимо обработать все сценарии оплаты и возвратов.",
		"Нужно разработать административную панель для управления контентом сайта. Требуется система ролей, логирование действий и экспорт данных.",
		"Требуется создать интеллектуального чат-бота для поддержки клиентов. Бот должен отвечать на частые вопросы и переключать на живого оператора при необходимости.",
		"Необходимо разработать дашборд для визуализации аналитических данных. Требуются графики, таблицы и возможность фильтрации по различным параметрам.",
		"Требуется мигрировать существующий сайт с устаревшего фреймворка на современный стек (React/Vue). Необходимо сохранить весь функционал и улучшить производительность.",
		"Нужно оптимизировать производительность веб-приложения: уменьшить время загрузки, оптимизировать запросы к базе данных и улучшить кэширование.",
		"Требуется разработать систему бронирования для отеля/ресторана. Необходимо реализовать календарь, выбор времени, уведомления и интеграцию с платежной системой.",
		"Нужно создать кастомный плагин для WordPress с расширенным функционалом. Плагин должен быть оптимизирован и соответствовать стандартам WordPress.",
		"Требуется настроить CI/CD pipeline для автоматической сборки, тестирования и развертывания приложения. Использовать GitHub Actions или GitLab CI.",
		"Необходимо разработать систему уведомлений для веб и мобильных приложений. Поддержка email, SMS, push-уведомлений и in-app уведомлений.",
		"Требуется создать мобильное приложение для фитнеса с трекингом тренировок, прогрессом и социальными функциями. Необходима интеграция с фитнес-браслетами.",
		"Нужно разработать пользовательский интерфейс для веб-приложения. Требуется создать дизайн-систему, компоненты и адаптивную верстку.",
		"Требуется создать систему управления задачами (task management) с досками, списками, метками и возможностью совместной работы.",
		"Необходимо разработать API для интеграции с внешними сервисами (CRM, аналитика, платежи). Требуется обработка webhooks и синхронизация данных.",
		"Требуется создать онлайн-калькулятор для расчета стоимости услуг. Калькулятор должен быть интерактивным с возможностью сохранения результатов.",
		"Нужно разработать систему рейтингов и отзывов для платформы. Требуется модерация, фильтрация и агрегация оценок.",
		"Требуется создать систему поиска с расширенными фильтрами, сортировкой и автодополнением. Необходима оптимизация для больших объемов данных.",
		"Нужно разработать модуль электронной коммерции с каталогом товаров, корзиной, оформлением заказа и управлением инвентарем.",
		"Требуется создать систему аналитики и отчетности с дашбордами, экспортом данных и настраиваемыми метриками.",
		"Необходимо разработать систему авторизации и безопасности с поддержкой OAuth, двухфакторной аутентификации и управления сессиями.",
		"Требуется создать мобильное приложение для социальной сети с лентой новостей, мессенджером и профилями пользователей.",
		"Нужно разработать корпоративный сайт с информацией о компании, услугах, портфолио и контактной формой.",
		"Требуется создать систему управления документами с версионированием, совместным редактированием и правами доступа.",
		"Необходимо разработать систему онлайн-обучения с видеолекциями, тестами, прогрессом обучения и сертификатами.",
	}

	skills := []string{
		"JavaScript", "TypeScript", "React", "Vue.js", "Angular", "Node.js", "Python", "Go",
		"Java", "C++", "C#", "PHP", "Ruby", "Swift", "Kotlin", "Dart", "Flutter", "React Native",
		"HTML", "CSS", "SASS", "LESS", "Webpack", "Vite", "Docker", "Kubernetes", "AWS", "Azure",
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "GraphQL", "REST API", "Git", "CI/CD",
		"Figma", "Adobe XD", "Photoshop", "Illustrator", "UI/UX Design", "Mobile Design",
	}

	statuses := []string{
		models.OrderStatusDraft,
		models.OrderStatusPublished,
		models.OrderStatusInProgress,
		models.OrderStatusCompleted,
		models.OrderStatusCancelled,
	}

	levels := []string{
		models.ExperienceLevelJunior,
		models.ExperienceLevelMiddle,
		models.ExperienceLevelSenior,
	}

	for i := 0; i < count; i++ {
		client := clients[rand.Intn(len(clients))]
		title := titles[rand.Intn(len(titles))]
		description := descriptions[rand.Intn(len(descriptions))]

		var budgetMin, budgetMax *float64
		if rand.Float32() > 0.2 { // 80% заказов с бюджетом
			min := float64(rand.Intn(50000)+10000) / 100.0      // 100-600 USD
			max := min + float64(rand.Intn(100000)+20000)/100.0 // +200-1200 USD
			budgetMin = &min
			budgetMax = &max
		}

		status := statuses[rand.Intn(len(statuses))]
		// Больше опубликованных заказов
		if rand.Float32() > 0.3 {
			status = models.OrderStatusPublished
		}

		var deadlineAt *time.Time
		if rand.Float32() > 0.3 { // 70% заказов с дедлайном
			days := time.Duration(rand.Intn(60)+7) * 24 * time.Hour
			deadline := time.Now().Add(days)
			deadlineAt = &deadline
		}

		order := &models.Order{
			ClientID:    client.ID,
			Title:       title,
			Description: description,
			BudgetMin:   budgetMin,
			BudgetMax:   budgetMax,
			Status:      status,
			DeadlineAt:  deadlineAt,
		}

		// Генерируем требования
		numRequirements := rand.Intn(5) + 1 // 1-5 требований
		requirements := make([]models.OrderRequirement, 0, numRequirements)
		skillMap := make(map[string]bool)
		for len(requirements) < numRequirements {
			skill := skills[rand.Intn(len(skills))]
			if !skillMap[skill] {
				level := levels[rand.Intn(len(levels))]
				requirements = append(requirements, models.OrderRequirement{
					Skill: skill,
					Level: level,
				})
				skillMap[skill] = true
			}
		}

		if err := s.orderRepo.Create(ctx, order, requirements, nil); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}
	}

	return nil
}

// toLatin транслитерирует русские имена в латиницу для email.
func toLatin(s string) string {
	translit := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
		'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
		'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
		'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
		'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo",
		'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
		'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
		'Ф': "F", 'Х': "H", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Sch",
		'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	result := ""
	for _, r := range s {
		if val, ok := translit[r]; ok {
			result += val
		} else {
			result += string(r)
		}
	}
	return result
}
