# API Документация для Frontend разработчика

## Обзор

Бэкенд фриланс-платформы с AI-ассистентом.

- **Base URL**: `http://localhost:8080/api`
- **Формат данных**: JSON
- **Аутентификация**: JWT Bearer Token

## Содержание

1. [Аутентификация](#1-аутентификация)
2. [Профиль пользователя](#2-профиль-пользователя)
3. [Заказы](#3-заказы)
4. [Отклики (Proposals)](#4-отклики-proposals)
5. [Чаты и сообщения](#5-чаты-и-сообщения)
6. [AI функции](#6-ai-функции)
7. [Портфолио](#7-портфолио)
8. [Уведомления](#8-уведомления)
9. [Медиа файлы](#9-медиа-файлы)
10. [WebSocket](#10-websocket)
11. [Dashboard](#11-dashboard)
12. [Статистика](#12-статистика)
12.5. [Каталог (категории и навыки)](#125-каталог-категории-и-навыки)
13. [Платежи и Escrow](#13-платежи-и-escrow-защищённая-сделка)
14. [Отзывы](#14-отзывы)
15. [Вывод средств](#15-вывод-средств-withdrawals)
16. [Избранное](#16-избранное-favorites)
17. [Споры](#17-споры-disputes)
18. [Жалобы](#18-жалобы-reports)
19. [Верификация](#19-верификация)
20. [Шаблоны откликов](#20-шаблоны-откликов)
21. [Поиск фрилансеров](#21-поиск-фрилансеров)
22. [Seed данные (только development)](#22-seed-данные-только-development)

---

## Общие принципы

### Заголовки запросов

```
Content-Type: application/json
Authorization: Bearer <access_token>
```

### Формат ошибок

```json
{
  "error": "Описание ошибки"
}
```

### HTTP коды ответов

| Код | Описание |
|-----|----------|
| 200 | Успешный запрос |
| 201 | Ресурс создан |
| 400 | Ошибка валидации |
| 401 | Не авторизован |
| 403 | Доступ запрещён |
| 404 | Не найдено |
| 500 | Ошибка сервера |

---

## 1. Аутентификация

### 1.1 Регистрация

```
POST /api/auth/register
```

**Тело запроса:**
```json
{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "username": "johndoe",
  "role": "freelancer",
  "display_name": "John Doe"
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| email | string | ✅ | Email (уникальный) |
| password | string | ✅ | Пароль (мин. 8 символов, буквы + цифры) |
| username | string | ❌ | Логин (3-30 символов, a-z, 0-9, _) |
| role | string | ❌ | `client` или `freelancer` (по умолчанию `client`) |
| display_name | string | ❌ | Отображаемое имя |

**Ответ (201):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "role": "freelancer",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z"
  },
  "profile": {
    "user_id": "uuid",
    "display_name": "John Doe",
    "experience_level": "junior",
    "skills": []
  },
  "tokens": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_at": "2024-01-01T00:15:00Z"
  }
}
```

### 1.2 Вход

```
POST /api/auth/login
```

**Тело запроса:**
```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**Ответ (200):** Аналогичен регистрации

### 1.3 Обновление токена

```
POST /api/auth/refresh
```

**Тело запроса:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Ответ (200):**
```json
{
  "tokens": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_at": "2024-01-01T00:15:00Z"
  }
}
```

### 1.4 Список сессий

```
GET /api/auth/sessions
Authorization: Bearer <token>
```

**Ответ (200):**
```json
[
  {
    "id": "uuid",
    "user_agent": "Mozilla/5.0...",
    "ip_address": "192.168.1.1",
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2024-01-31T00:00:00Z"
  }
]
```

### 1.5 Удалить сессию

```
DELETE /api/auth/sessions/:id
```

### 1.6 Удалить все сессии кроме текущей

```
DELETE /api/auth/sessions
```

---

## 2. Профиль пользователя

### 2.1 Получить свой профиль

```
GET /api/profile
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "role": "freelancer"
  },
  "profile": {
    "user_id": "uuid",
    "display_name": "John Doe",
    "bio": "Опытный разработчик",
    "hourly_rate": 50.0,
    "experience_level": "senior",
    "skills": ["React", "Node.js", "TypeScript"],
    "location": "Москва",
    "photo_id": "uuid",
    "ai_summary": "AI-сгенерированное описание"
  },
  "stats": {
    "total_orders": 15,
    "completed_orders": 12,
    "average_rating": 4.8,
    "total_reviews": 10
  }
}
```

### 2.2 Обновить профиль

```
PUT /api/profile
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "display_name": "John Doe",
  "bio": "Опытный full-stack разработчик",
  "hourly_rate": 75.0,
  "experience_level": "senior",
  "skills": ["React", "Node.js", "TypeScript", "Go"],
  "location": "Москва",
  "photo_id": "uuid-фото",
  "phone": "+7 999 123-45-67",
  "telegram": "@johndoe",
  "website": "https://johndoe.dev",
  "company_name": "ООО Разработка",
  "inn": "1234567890"
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| display_name | string | Отображаемое имя (обязательно) |
| bio | string | О себе |
| hourly_rate | number | Ставка в час (₽) |
| experience_level | string | `junior`, `middle`, `senior` |
| skills | string[] | Массив навыков |
| location | string | Местоположение |
| photo_id | string | UUID загруженного фото |
| phone | string | Телефон |
| telegram | string | Telegram username |
| website | string | Личный сайт |
| company_name | string | Название компании (для юр. лиц) |
| inn | string | ИНН (для юр. лиц) |

### 2.3 Изменить роль

```
PUT /api/users/me/role
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "role": "freelancer"
}
```

### 2.4 Получить профиль другого пользователя

```
GET /api/users/:id
```

**Ответ (200):**
```json
{
  "user": {
    "id": "uuid",
    "username": "johndoe",
    "role": "freelancer"
  },
  "profile": {
    "display_name": "John Doe",
    "bio": "...",
    "skills": ["React", "Node.js"],
    "experience_level": "senior"
  },
  "stats": {
    "completed_orders": 12,
    "average_rating": 4.8,
    "total_reviews": 10
  }
}
```

---

## 3. Заказы

### 3.1 Создать заказ

```
POST /api/orders
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Разработка мобильного приложения",
  "description": "Нужно разработать iOS и Android приложение...",
  "category_id": "uuid-категории",
  "budget_min": 50000,
  "budget_max": 100000,
  "deadline_at": "2024-06-01T00:00:00Z",
  "requirements": [
    {"skill": "Swift", "level": "senior"},
    {"skill": "Kotlin", "level": "senior"},
    {"skill": "Firebase", "level": "middle"}
  ],
  "attachment_ids": ["uuid-1", "uuid-2"]
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| title | string | ✅ | Заголовок (5-200 символов) |
| description | string | ✅ | Описание (20-10000 символов) |
| category_id | string | ❌ | UUID категории (из /catalog/categories) |
| budget_min | number | ❌ | Минимальный бюджет (₽) |
| budget_max | number | ❌ | Максимальный бюджет (₽) |
| deadline_at | string | ❌ | Дедлайн (ISO 8601) |
| requirements | array | ❌ | Требуемые навыки (из /catalog/skills) |
| attachment_ids | string[] | ❌ | UUID загруженных файлов |

**Ответ (201):**
```json
{
  "id": "uuid",
  "client_id": "uuid",
  "category_id": "uuid",
  "title": "Разработка мобильного приложения",
  "description": "...",
  "budget_min": 50000,
  "budget_max": 100000,
  "status": "draft",
  "deadline_at": "2024-06-01T00:00:00Z",
  "ai_summary": "AI-сгенерированное резюме заказа",
  "created_at": "2024-01-01T00:00:00Z",
  "requirements": [...],
  "attachments": [...]
}
```

### 3.2 Список заказов (публичный)

```
GET /api/orders?status=published&limit=20&offset=0&search=react&category_id=uuid
```

**Query параметры:**

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| status | string | published | Фильтр по статусу |
| category_id | string | - | Фильтр по категории |
| limit | int | 20 | Количество (макс. 100) |
| offset | int | 0 | Смещение |
| search | string | - | Поиск по заголовку/описанию |
| skills | string | - | Фильтр по навыкам (через запятую) |
| budget_min | number | - | Мин. бюджет (₽) |
| budget_max | number | - | Макс. бюджет (₽) |

**Ответ (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "category_id": "uuid",
      "title": "...",
      "description": "...",
      "status": "published",
      "budget_min": 50000,
      "budget_max": 100000,
      "proposals_count": 5,
      "created_at": "..."
    }
  ],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

### 3.3 Мои заказы

```
GET /api/orders/my
Authorization: Bearer <token>
```

### 3.4 Получить заказ

```
GET /api/orders/:id
```

**Ответ (200):**
```json
{
  "id": "uuid",
  "client_id": "uuid",
  "title": "...",
  "description": "...",
  "status": "published",
  "budget_min": 50000,
  "budget_max": 100000,
  "deadline_at": "2024-06-01T00:00:00Z",
  "ai_summary": "...",
  "best_recommendation_proposal_id": "uuid",
  "best_recommendation_justification": "Почему этот исполнитель лучший",
  "created_at": "...",
  "updated_at": "...",
  "requirements": [
    {"id": "uuid", "skill": "Swift", "level": "senior"}
  ],
  "attachments": [
    {"id": "uuid", "media": {"id": "uuid", "url": "/media/...", "filename": "..."}}
  ]
}
```

### 3.5 Обновить заказ

```
PUT /api/orders/:id
Authorization: Bearer <token>
```

Тело запроса аналогично созданию.

### 3.6 Удалить заказ

```
DELETE /api/orders/:id
Authorization: Bearer <token>
```

### Статусы заказов

| Статус | Описание |
|--------|----------|
| `draft` | Черновик |
| `published` | Опубликован |
| `in_progress` | В работе |
| `pending_completion` | Ожидает подтверждения завершения |
| `completed` | Завершён |
| `cancelled` | Отменён |



---

## 4. Отклики (Proposals)

### 4.1 Создать отклик

```
POST /api/orders/:id/proposals
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "cover_letter": "Здравствуйте! Имею 5 лет опыта в мобильной разработке...",
  "amount": 75000
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| cover_letter | string | ✅ | Сопроводительное письмо |
| amount | number | ❌ | Предлагаемая сумма |

**Ответ (201):**
```json
{
  "id": "uuid",
  "order_id": "uuid",
  "freelancer_id": "uuid",
  "cover_letter": "...",
  "proposed_amount": 75000,
  "status": "pending",
  "ai_feedback": "AI советы по улучшению отклика",
  "created_at": "..."
}
```

### 4.2 Список откликов на заказ

```
GET /api/orders/:id/proposals
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "proposals": [
    {
      "id": "uuid",
      "freelancer_id": "uuid",
      "cover_letter": "...",
      "proposed_amount": 75000,
      "status": "pending",
      "created_at": "...",
      "freelancer": {
        "display_name": "John Doe",
        "skills": ["Swift", "Kotlin"],
        "experience_level": "senior",
        "photo_id": "uuid"
      }
    }
  ],
  "best_recommendation_proposal_id": "uuid",
  "recommendation_justification": "Этот исполнитель лучше всего подходит потому что..."
}
```

### 4.3 Мой отклик на заказ

```
GET /api/orders/:id/my-proposal
Authorization: Bearer <token>
```

### 4.4 Мои отклики

```
GET /api/proposals/my
Authorization: Bearer <token>
```

**Ответ (200):**
```json
[
  {
    "id": "uuid",
    "cover_letter": "...",
    "proposed_amount": 75000,
    "status": "pending",
    "order": {
      "id": "uuid",
      "title": "Разработка приложения",
      "status": "published",
      "client_id": "uuid"
    }
  }
]
```

### 4.5 Изменить статус отклика

```
PUT /api/orders/:orderId/proposals/:proposalId/status
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "status": "accepted"
}
```

| Статус | Кто может | Описание |
|--------|-----------|----------|
| `accepted` | Клиент | Принять отклик |
| `rejected` | Клиент | Отклонить |
| `withdrawn` | Фрилансер | Отозвать свой отклик |

**Ответ (200):**
```json
{
  "proposal": {...},
  "conversation": {
    "id": "uuid",
    "order_id": "uuid",
    "client_id": "uuid",
    "freelancer_id": "uuid"
  },
  "order": {
    "id": "uuid",
    "title": "...",
    "status": "in_progress"
  }
}
```

### 4.6 Отметить заказ выполненным (фрилансер)

```
POST /api/orders/:id/complete-by-freelancer
Authorization: Bearer <token>
```

---

## 5. Чаты и сообщения

### 5.1 Мои чаты

```
GET /api/conversations/my
Authorization: Bearer <token>
```

**Ответ (200):**
```json
[
  {
    "id": "uuid",
    "order_id": "uuid",
    "client_id": "uuid",
    "freelancer_id": "uuid",
    "created_at": "...",
    "order_title": "Разработка приложения",
    "other_participant": {
      "user_id": "uuid",
      "display_name": "John Doe",
      "photo_id": "uuid"
    },
    "last_message": {
      "id": "uuid",
      "content": "Привет!",
      "created_at": "..."
    },
    "unread_count": 3
  }
]
```

### 5.2 Получить/создать чат

```
GET /api/orders/:orderId/conversations/:participantId
Authorization: Bearer <token>
```

Создаёт чат если не существует.

### 5.3 Сообщения чата

```
GET /api/conversations/:conversationId/messages?limit=50&offset=0
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "messages": [
    {
      "id": "uuid",
      "conversation_id": "uuid",
      "author_type": "user",
      "author_id": "uuid",
      "content": "Привет! Готов обсудить проект.",
      "parent_message_id": null,
      "created_at": "...",
      "attachments": [
        {"id": "uuid", "media": {"url": "/media/...", "filename": "doc.pdf"}}
      ],
      "reactions": [
        {"id": "uuid", "user_id": "uuid", "emoji": "👍"}
      ]
    }
  ],
  "conversation_id": "uuid",
  "order_title": "Разработка приложения",
  "other_participant": {
    "user_id": "uuid",
    "display_name": "John Doe"
  }
}
```

### 5.4 Отправить сообщение

```
POST /api/conversations/:conversationId/messages
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "content": "Привет! Готов начать работу.",
  "parent_message_id": "uuid-родительского-сообщения",
  "attachment_ids": ["uuid-1", "uuid-2"]
}
```

### 5.5 Редактировать сообщение

```
PUT /api/conversations/:conversationId/messages/:messageId
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "content": "Исправленный текст"
}
```

### 5.6 Удалить сообщение

```
DELETE /api/conversations/:conversationId/messages/:messageId
Authorization: Bearer <token>
```

### 5.7 Добавить реакцию

```
POST /api/conversations/:conversationId/messages/:messageId/reactions
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "emoji": "👍"
}
```

### 5.8 Удалить реакцию

```
DELETE /api/conversations/:conversationId/messages/:messageId/reactions
Authorization: Bearer <token>
```

---

## 6. AI функции

Все AI endpoints поддерживают два режима:
- **Обычный**: возвращает JSON
- **Streaming**: возвращает Server-Sent Events (SSE)

### 6.1 Генерация описания заказа

```
POST /api/ai/orders/description
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Разработка мобильного приложения",
  "brief": "Приложение для доставки еды",
  "skills": ["Swift", "Kotlin", "Firebase"]
}
```

**Ответ (200):**
```json
{
  "description": "Сгенерированное описание заказа..."
}
```

**Streaming версия:**
```
POST /api/ai/orders/description/stream
```

Возвращает SSE:
```
data: {"delta": "Сгенери"}
data: {"delta": "рованное"}
data: {"delta": " описание..."}
data: [DONE]
```

### 6.2 Улучшение описания заказа

```
POST /api/ai/orders/improve
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Разработка приложения",
  "description": "Нужно сделать приложение"
}
```

### 6.3 Генерация предложений для заказа

```
POST /api/ai/orders/suggestions
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Разработка приложения",
  "description": "Текущее описание..."
}
```

**Ответ:**
```json
{
  "suggestions": [
    "Добавьте информацию о целевой аудитории",
    "Укажите предпочтительный стек технологий"
  ]
}
```

### 6.4 Генерация навыков для заказа

```
POST /api/ai/orders/skills
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Разработка мобильного приложения",
  "description": "iOS и Android приложение для доставки еды"
}
```

**Ответ:**
```json
{
  "skills": [
    {"skill": "Swift", "level": "senior"},
    {"skill": "Kotlin", "level": "senior"},
    {"skill": "Firebase", "level": "middle"}
  ]
}
```

### 6.5 Рекомендация бюджета

```
POST /api/ai/orders/budget
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Разработка приложения",
  "description": "...",
  "requirements": [{"skill": "Swift", "level": "senior"}]
}
```

**Ответ:**
```json
{
  "budget_min": 50000,
  "budget_max": 100000,
  "explanation": "Обоснование рекомендации..."
}
```

### 6.6 Генерация отклика

```
POST /api/ai/orders/:id/proposal
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "cover_letter": "Сгенерированный текст отклика..."
}
```

### 6.7 Фидбек по отклику

```
GET /api/ai/orders/:id/proposals/feedback
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "feedback": [
    "Укажите конкретные примеры из портфолио",
    "Добавьте предполагаемые сроки"
  ]
}
```

### 6.8 Рекомендованные заказы для фрилансера

```
GET /api/ai/orders/recommended
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "orders": [
    {
      "order_id": "uuid",
      "match_score": 9.5,
      "explanation": "Идеально подходит по навыкам Swift и Kotlin"
    }
  ],
  "explanation": "Общее объяснение рекомендаций"
}
```

### 6.9 Рекомендация цены и сроков

```
GET /api/ai/orders/:id/price-timeline
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "recommended_amount": 75000,
  "min_amount": 60000,
  "max_amount": 90000,
  "recommended_days": 30,
  "min_days": 21,
  "max_days": 45,
  "explanation": "Обоснование..."
}
```

### 6.10 Оценка качества заказа

```
GET /api/ai/orders/:id/quality
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "score": 8,
  "strengths": ["Чёткое описание", "Реалистичный бюджет"],
  "weaknesses": ["Нет дедлайна"],
  "recommendations": ["Добавьте срок выполнения"]
}
```

### 6.11 Поиск подходящих фрилансеров

```
GET /api/ai/orders/:id/suitable-freelancers
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "freelancers": [
    {
      "user_id": "uuid",
      "match_score": 9.5,
      "explanation": "Senior разработчик с опытом в Swift и Kotlin"
    }
  ]
}
```

### 6.12 Резюме переписки

```
GET /api/ai/conversations/:conversationId/summary
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "summary": "Краткое резюме переписки...",
  "next_steps": ["Обсудить детали ТЗ", "Согласовать сроки"],
  "agreements": ["Бюджет 75000 руб"],
  "open_questions": ["Какой дизайн предпочтителен?"]
}
```

### 6.13 Улучшение профиля

```
POST /api/ai/profile/improve
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "bio": "Разработчик с опытом 5 лет",
  "skills": ["React", "Node.js"],
  "level": "senior"
}
```

**Ответ:**
```json
{
  "improved_bio": "Улучшенное описание профиля..."
}
```

### 6.14 Улучшение портфолио

```
POST /api/ai/portfolio/improve
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Мобильное приложение",
  "description": "Разработал приложение",
  "tags": ["iOS", "Swift"]
}
```

### 6.15 AI Ассистент (чат)

```
POST /api/ai/assistant
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "message": "Как улучшить мой профиль?",
  "context": "profile"
}
```

**Ответ:**
```json
{
  "response": "Ответ AI ассистента..."
}
```

### 6.16 Приветственное сообщение

```
POST /api/ai/welcome-message
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "order_id": "uuid",
  "freelancer_id": "uuid"
}
```


---

## 7. Портфолио

### 7.1 Мои работы

```
GET /api/portfolio
Authorization: Bearer <token>
```

**Ответ (200):**
```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "title": "Мобильное приложение для доставки",
    "description": "Разработал iOS приложение...",
    "url": "https://apps.apple.com/...",
    "tags": ["iOS", "Swift", "Firebase"],
    "ai_tags": ["mobile", "delivery", "e-commerce"],
    "created_at": "...",
    "media": [
      {"id": "uuid", "url": "/media/...", "filename": "screenshot.png"}
    ]
  }
]
```

### 7.2 Создать работу

```
POST /api/portfolio
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "title": "Мобильное приложение",
  "description": "Описание проекта...",
  "url": "https://example.com",
  "media_ids": ["uuid-1", "uuid-2"],
  "tags": ["iOS", "Swift"]
}
```

### 7.3 Получить работу

```
GET /api/portfolio/:id
Authorization: Bearer <token>
```

### 7.4 Обновить работу

```
PUT /api/portfolio/:id
Authorization: Bearer <token>
```

### 7.5 Удалить работу

```
DELETE /api/portfolio/:id
Authorization: Bearer <token>
```

### 7.6 Портфолио другого пользователя

```
GET /api/users/:userId/portfolio
```

---

## 8. Уведомления

### 8.1 Список уведомлений

```
GET /api/notifications?limit=20&offset=0
Authorization: Bearer <token>
```

**Ответ (200):**
```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "payload": {
      "type": "new_proposal",
      "order_id": "uuid",
      "order_title": "Разработка приложения",
      "freelancer_name": "John Doe",
      "message": "Новый отклик на ваш заказ"
    },
    "is_read": false,
    "created_at": "..."
  }
]
```

### Типы уведомлений

| Тип | Описание |
|-----|----------|
| `new_proposal` | Новый отклик на заказ |
| `proposal_accepted` | Ваш отклик принят |
| `proposal_rejected` | Ваш отклик отклонён |
| `new_message` | Новое сообщение в чате |
| `order_completed` | Заказ завершён |
| `order_cancelled` | Заказ отменён |

### 8.2 Количество непрочитанных

```
GET /api/notifications/unread/count
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "count": 5
}
```

### 8.3 Получить уведомление

```
GET /api/notifications/:id
Authorization: Bearer <token>
```

### 8.4 Отметить прочитанным

```
PUT /api/notifications/:id/read
Authorization: Bearer <token>
```

### 8.5 Отметить все прочитанными

```
PUT /api/notifications/read-all
Authorization: Bearer <token>
```

### 8.6 Удалить уведомление

```
DELETE /api/notifications/:id
Authorization: Bearer <token>
```

---

## 9. Медиа файлы

### 9.1 Загрузить фото/файл

```
POST /api/media/photos
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**Form data:**
- `file`: файл (макс. 10MB)

**Ответ (201):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "filename": "photo.jpg",
  "content_type": "image/jpeg",
  "size": 1024000,
  "url": "/media/uuid/photo.jpg",
  "created_at": "..."
}
```

### 9.2 Удалить файл

```
DELETE /api/media/:id
Authorization: Bearer <token>
```

### 9.3 Получить файл

```
GET /media/:userId/:filename
```

Публичный доступ к загруженным файлам.

---

## 10. WebSocket

### Подключение

```
GET /api/ws?token=<access_token>
```

или через заголовок:
```
GET /api/ws
Authorization: Bearer <token>
```

### Формат сообщений

**Входящие (от сервера):**
```json
{
  "type": "notification",
  "payload": {
    "type": "new_message",
    "conversation_id": "uuid",
    "message": {...}
  }
}
```

**Типы событий:**

| Тип | Описание |
|-----|----------|
| `notification` | Уведомление |
| `new_message` | Новое сообщение в чате |
| `message_updated` | Сообщение отредактировано |
| `message_deleted` | Сообщение удалено |
| `typing` | Пользователь печатает |
| `proposal_status_changed` | Статус отклика изменён |
| `order_status_changed` | Статус заказа изменён |

---

## 11. Dashboard

### 11.1 Данные дашборда

```
GET /api/dashboard/data
Authorization: Bearer <token>
```

**Ответ для клиента:**
```json
{
  "stats": {
    "total_orders": 10,
    "active_orders": 3,
    "completed_orders": 5,
    "total_spent": 250000
  },
  "recent_orders": [...],
  "recent_proposals": [...],
  "notifications_count": 5
}
```

**Ответ для фрилансера:**
```json
{
  "stats": {
    "total_proposals": 20,
    "accepted_proposals": 8,
    "completed_orders": 6,
    "total_earned": 180000,
    "average_rating": 4.8
  },
  "recommended_orders": [
    {
      "order_id": "uuid",
      "match_score": 9.5,
      "explanation": "Подходит по навыкам"
    }
  ],
  "active_orders": [...],
  "recent_messages": [...]
}
```

### 11.2 Инвалидация кэша

```
POST /api/dashboard/cache/invalidate
Authorization: Bearer <token>
```

---

## 12. Статистика

### 12.1 Моя статистика

```
GET /api/stats
Authorization: Bearer <token>
```

**Ответ:**
```json
{
  "total_orders": 15,
  "completed_orders": 12,
  "in_progress_orders": 2,
  "total_proposals": 30,
  "accepted_proposals": 12,
  "average_rating": 4.8,
  "total_reviews": 10,
  "total_earned": 500000,
  "total_spent": 0
}
```

---

## 12.5 Каталог (категории и навыки)

Публичные эндпоинты для получения списка категорий и навыков.

### 12.5.1 Список категорий

```
GET /api/catalog/categories
```

**Ответ (200):**
```json
{
  "categories": [
    {
      "id": "uuid",
      "slug": "web-development",
      "name": "Веб-разработка",
      "description": "Создание сайтов и веб-приложений",
      "icon": "🌐",
      "parent_id": null,
      "sort_order": 1,
      "is_active": true,
      "children": [
        {
          "id": "uuid",
          "slug": "frontend",
          "name": "Frontend разработка",
          "parent_id": "uuid",
          "sort_order": 1
        },
        {
          "id": "uuid",
          "slug": "backend",
          "name": "Backend разработка",
          "parent_id": "uuid",
          "sort_order": 2
        }
      ]
    },
    {
      "id": "uuid",
      "slug": "mobile-development",
      "name": "Мобильная разработка",
      "description": "iOS и Android приложения",
      "icon": "📱",
      "sort_order": 2,
      "children": []
    }
  ]
}
```

### 12.5.2 Получить категорию

```
GET /api/catalog/categories/:slug
```

**Ответ (200):**
```json
{
  "category": {
    "id": "uuid",
    "slug": "web-development",
    "name": "Веб-разработка",
    "description": "Создание сайтов и веб-приложений",
    "icon": "🌐",
    "children": [...]
  },
  "skills": [
    {"id": "uuid", "slug": "javascript", "name": "JavaScript"},
    {"id": "uuid", "slug": "react", "name": "React"},
    {"id": "uuid", "slug": "nodejs", "name": "Node.js"}
  ]
}
```

### 12.5.3 Список навыков

```
GET /api/catalog/skills?category_id=uuid
```

**Query параметры:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| category_id | string | Фильтр по категории (опционально) |

**Ответ (200):**
```json
{
  "skills": [
    {"id": "uuid", "slug": "javascript", "name": "JavaScript", "category_id": "uuid"},
    {"id": "uuid", "slug": "typescript", "name": "TypeScript", "category_id": "uuid"},
    {"id": "uuid", "slug": "react", "name": "React", "category_id": "uuid"}
  ]
}
```

### Доступные категории

| Slug | Название | Иконка |
|------|----------|--------|
| web-development | Веб-разработка | 🌐 |
| mobile-development | Мобильная разработка | 📱 |
| design | Дизайн | 🎨 |
| marketing | Маркетинг | 📈 |
| writing | Копирайтинг | ✍️ |
| video | Видео и анимация | 🎬 |
| data | Данные и аналитика | 📊 |
| admin | Администрирование | ⚙️ |
| other | Другое | 📦 |

---

## 13. Платежи и Escrow (Защищённая сделка)

Система защищённой оплаты гарантирует безопасную передачу средств между заказчиком и исполнителем. Все суммы указываются в рублях (₽).

### Как работает Escrow:
1. Заказчик пополняет баланс
2. При принятии отклика создаётся escrow - средства замораживаются
3. После завершения заказа средства переводятся фрилансеру
4. При отмене заказа средства возвращаются заказчику

### 13.1 Получить баланс

```
GET /api/payments/balance
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "user_id": "uuid",
  "available": 50000.00,
  "frozen": 25000.00,
  "updated_at": "2024-01-01T00:00:00Z"
}
```

| Поле | Описание |
|------|----------|
| available | Доступные средства (₽) |
| frozen | Замороженные в escrow (₽) |

### 13.2 Пополнить баланс

```
POST /api/payments/deposit
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "amount": 10000.00
}
```

**Ответ (200):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "type": "deposit",
  "amount": 10000.00,
  "status": "completed",
  "description": "Пополнение баланса",
  "created_at": "2024-01-01T00:00:00Z",
  "completed_at": "2024-01-01T00:00:00Z"
}
```

### 13.3 Создать Escrow (защищённую сделку)

```
POST /api/payments/escrow
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "order_id": "uuid",
  "freelancer_id": "uuid",
  "amount": 25000.00
}
```

**Ответ (201):**
```json
{
  "id": "uuid",
  "order_id": "uuid",
  "client_id": "uuid",
  "freelancer_id": "uuid",
  "amount": 25000.00,
  "status": "held",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 13.4 Получить Escrow по заказу

```
GET /api/payments/escrow/:orderId
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "id": "uuid",
  "order_id": "uuid",
  "client_id": "uuid",
  "freelancer_id": "uuid",
  "amount": 25000.00,
  "status": "held",
  "created_at": "2024-01-01T00:00:00Z",
  "released_at": null
}
```

### Статусы Escrow

| Статус | Описание |
|--------|----------|
| `held` | Средства заморожены |
| `released` | Средства переведены фрилансеру |
| `refunded` | Средства возвращены заказчику |
| `disputed` | Спор (требует разрешения) |

### 13.5 История транзакций

```
GET /api/payments/transactions?limit=20&offset=0
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "transactions": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "order_id": "uuid",
      "type": "escrow_release",
      "amount": 25000.00,
      "status": "completed",
      "description": "Получение оплаты за заказ",
      "created_at": "2024-01-01T00:00:00Z",
      "completed_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Типы транзакций

| Тип | Описание |
|-----|----------|
| `deposit` | Пополнение баланса |
| `withdrawal` | Вывод средств |
| `escrow_hold` | Заморозка для escrow |
| `escrow_release` | Получение оплаты |
| `escrow_refund` | Возврат средств |

---

## 14. Отзывы

Отзывы можно оставить только после завершения заказа. Каждый участник (заказчик и фрилансер) может оставить один отзыв о другом участнике.

### 14.1 Создать отзыв

```
POST /api/orders/:id/reviews
Authorization: Bearer <token>
```

**Тело запроса:**
```json
{
  "rating": 5,
  "comment": "Отличная работа! Рекомендую."
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| rating | int | ✅ | Оценка от 1 до 5 |
| comment | string | ❌ | Текстовый комментарий |

**Ответ (201):**
```json
{
  "id": "uuid",
  "order_id": "uuid",
  "reviewer_id": "uuid",
  "reviewed_id": "uuid",
  "rating": 5,
  "comment": "Отличная работа! Рекомендую.",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### 14.2 Отзывы по заказу

```
GET /api/orders/:id/reviews
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "reviews": [
    {
      "id": "uuid",
      "order_id": "uuid",
      "reviewer_id": "uuid",
      "reviewed_id": "uuid",
      "rating": 5,
      "comment": "Отличная работа!",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### 14.3 Отзывы о пользователе (публичный)

```
GET /api/users/:id/reviews?limit=20&offset=0
```

**Ответ (200):**
```json
{
  "reviews": [...],
  "average_rating": 4.8,
  "total_reviews": 15
}
```

### 14.4 Проверить возможность оставить отзыв

```
GET /api/orders/:id/can-review
Authorization: Bearer <token>
```

**Ответ (200):**
```json
{
  "can_review": true
}
```

---

## Приложение A: Модели данных

### User
```typescript
interface User {
  id: string;
  email: string;
  username: string;
  role: 'client' | 'freelancer' | 'admin';
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}
```

### Profile
```typescript
interface Profile {
  user_id: string;
  display_name: string;
  bio?: string;
  hourly_rate?: number;
  experience_level: 'junior' | 'middle' | 'senior';
  skills: string[];
  location?: string;
  photo_id?: string;
  ai_summary?: string;
  phone?: string;
  telegram?: string;
  website?: string;
  company_name?: string;
  inn?: string;
}
```

### Order
```typescript
interface Order {
  id: string;
  client_id: string;
  freelancer_id?: string;
  category_id?: string;
  title: string;
  description: string;
  budget_min?: number;
  budget_max?: number;
  final_amount?: number;
  status: 'draft' | 'published' | 'in_progress' | 'pending_completion' | 'completed' | 'cancelled';
  deadline_at?: string;
  ai_summary?: string;
  created_at: string;
  updated_at: string;
  category?: Category;
}
```

### Proposal
```typescript
interface Proposal {
  id: string;
  order_id: string;
  freelancer_id: string;
  cover_letter: string;
  proposed_amount?: number;
  status: 'pending' | 'accepted' | 'rejected' | 'withdrawn';
  ai_feedback?: string;
  created_at: string;
  updated_at: string;
}
```

### Message
```typescript
interface Message {
  id: string;
  conversation_id: string;
  author_type: 'user' | 'system' | 'ai';
  author_id?: string;
  content: string;
  parent_message_id?: string;
  created_at: string;
  attachments?: MessageAttachment[];
  reactions?: MessageReaction[];
}
```

### Notification
```typescript
interface Notification {
  id: string;
  user_id: string;
  payload: {
    type: string;
    [key: string]: any;
  };
  is_read: boolean;
  created_at: string;
}
```

### Category
```typescript
interface Category {
  id: string;
  slug: string;
  name: string;
  description?: string;
  icon?: string;
  parent_id?: string;
  sort_order: number;
  is_active: boolean;
  children?: Category[];
}
```

### Skill
```typescript
interface Skill {
  id: string;
  slug: string;
  name: string;
  category_id?: string;
  is_active: boolean;
}
```

### UserBalance
```typescript
interface UserBalance {
  user_id: string;
  available: number;
  frozen: number;
  updated_at: string;
}
```

### Escrow
```typescript
interface Escrow {
  id: string;
  order_id: string;
  client_id: string;
  freelancer_id: string;
  amount: number;
  status: 'held' | 'released' | 'refunded' | 'disputed';
  created_at: string;
  released_at?: string;
}
```

### Transaction
```typescript
interface Transaction {
  id: string;
  user_id: string;
  order_id?: string;
  type: 'deposit' | 'withdrawal' | 'escrow_hold' | 'escrow_release' | 'escrow_refund';
  amount: number;
  status: 'pending' | 'completed' | 'failed' | 'cancelled';
  description?: string;
  created_at: string;
  completed_at?: string;
}
```

### Review
```typescript
interface Review {
  id: string;
  order_id: string;
  reviewer_id: string;
  reviewed_id: string;
  rating: number;
  comment?: string;
  created_at: string;
  updated_at: string;
}
```

---

## Приложение B: Коды ошибок

| Код | Сообщение | Описание |
|-----|-----------|----------|
| 400 | "email уже используется" | Email занят |
| 400 | "неверный формат email" | Невалидный email |
| 400 | "пароль должен содержать минимум 8 символов" | Слабый пароль |
| 400 | "недостаточно средств на балансе" | Не хватает денег для escrow |
| 400 | "рейтинг должен быть от 1 до 5" | Некорректный рейтинг |
| 400 | "отзыв можно оставить только после завершения заказа" | Заказ не завершён |
| 400 | "вы уже оставили отзыв на этот заказ" | Дубликат отзыва |
| 401 | "неверный email или пароль" | Ошибка входа |
| 401 | "токен истёк" | Нужно обновить токен |
| 403 | "нет доступа к этому ресурсу" | Недостаточно прав |
| 404 | "заказ не найден" | Ресурс не существует |
| 404 | "escrow не найден" | Escrow не существует |
| 409 | "вы уже откликнулись на этот заказ" | Дубликат отклика |

---

## Приложение C: Примеры использования

### Полный флоу создания заказа

```javascript
// 1. Загрузить вложения
const formData = new FormData();
formData.append('file', file);
const { data: media } = await api.post('/media/photos', formData);

// 2. Получить AI рекомендации по навыкам
const { data: skills } = await api.post('/ai/orders/skills', {
  title: 'Разработка приложения',
  description: 'iOS и Android приложение для доставки'
});

// 3. Получить рекомендацию бюджета
const { data: budget } = await api.post('/ai/orders/budget', {
  title: 'Разработка приложения',
  description: '...',
  requirements: skills.skills
});

// 4. Создать заказ
const { data: order } = await api.post('/orders', {
  title: 'Разработка приложения',
  description: '...',
  budget_min: budget.budget_min,
  budget_max: budget.budget_max,
  requirements: skills.skills,
  attachment_ids: [media.id]
});
```

### Флоу защищённой оплаты (Escrow)

```javascript
// 1. Заказчик пополняет баланс
await api.post('/payments/deposit', { amount: 50000 });

// 2. При принятии отклика создаём escrow
const { data: escrow } = await api.post('/payments/escrow', {
  order_id: orderId,
  freelancer_id: freelancerId,
  amount: 25000
});

// 3. После завершения заказа escrow автоматически освобождается
// Фрилансер получает средства на баланс

// 4. Проверить баланс
const { data: balance } = await api.get('/payments/balance');
console.log(`Доступно: ${balance.available}₽, Заморожено: ${balance.frozen}₽`);
```

### Оставить отзыв после завершения заказа

```javascript
// 1. Проверить, можно ли оставить отзыв
const { data: { can_review } } = await api.get(`/orders/${orderId}/can-review`);

if (can_review) {
  // 2. Оставить отзыв
  await api.post(`/orders/${orderId}/reviews`, {
    rating: 5,
    comment: 'Отличная работа! Всё сделано в срок и качественно.'
  });
}

// 3. Получить отзывы о пользователе
const { data } = await api.get(`/users/${userId}/reviews`);
console.log(`Средний рейтинг: ${data.average_rating}, Отзывов: ${data.total_reviews}`);
```

### Работа со streaming AI

```javascript
const eventSource = new EventSource(
  '/api/ai/orders/description/stream',
  {
    headers: { 'Authorization': `Bearer ${token}` },
    method: 'POST',
    body: JSON.stringify({ title, brief, skills })
  }
);

let fullText = '';
eventSource.onmessage = (event) => {
  if (event.data === '[DONE]') {
    eventSource.close();
    return;
  }
  const { delta } = JSON.parse(event.data);
  fullText += delta;
  updateUI(fullText);
};
```

### WebSocket подключение

```javascript
const ws = new WebSocket(`ws://localhost:8080/api/ws?token=${accessToken}`);

ws.onmessage = (event) => {
  const { type, payload } = JSON.parse(event.data);
  
  switch (type) {
    case 'new_message':
      addMessageToChat(payload);
      break;
    case 'notification':
      showNotification(payload);
      break;
    case 'typing':
      showTypingIndicator(payload.user_id);
      break;
  }
};
```

---

## 15. Вывод средств (Withdrawals)

### 15.1 Создать заявку на вывод

```
POST /api/withdrawals
```

**Тело запроса:**
```json
{
  "amount": 5000,
  "card_last4": "1234",
  "bank_name": "Сбербанк"
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| amount | number | ✅ | Сумма вывода (мин. 100₽) |
| card_last4 | string | ✅ | Последние 4 цифры карты |
| bank_name | string | ✅ | Название банка |

**Ответ (201):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "amount": 5000,
  "status": "pending",
  "card_last4": "1234",
  "bank_name": "Сбербанк",
  "created_at": "2024-12-03T00:00:00Z"
}
```

### 15.2 Список заявок на вывод

```
GET /api/withdrawals?limit=20&offset=0
```

**Ответ (200):** Массив объектов Withdrawal

---

## 16. Избранное (Favorites)

### 16.1 Добавить в избранное

```
POST /api/favorites
```

**Тело запроса:**
```json
{
  "target_type": "order",
  "target_id": "uuid"
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| target_type | string | `order` или `freelancer` |
| target_id | string | UUID заказа или фрилансера |

### 16.2 Список избранного

```
GET /api/favorites?type=order&limit=20&offset=0
```

| Параметр | Описание |
|----------|----------|
| type | Фильтр по типу: `order`, `freelancer` |

### 16.3 Проверить, в избранном ли

```
GET /api/favorites/:type/:id
```

**Ответ (200):**
```json
{
  "is_favorite": true
}
```

### 16.4 Удалить из избранного

```
DELETE /api/favorites/:type/:id
```

---

## 17. Споры (Disputes)

### 17.1 Открыть спор

```
POST /api/orders/:id/dispute
```

**Тело запроса:**
```json
{
  "reason": "Работа не соответствует ТЗ"
}
```

**Условия:**
- Escrow должен быть в статусе `held`
- Только участники сделки могут открыть спор

**Ответ (201):**
```json
{
  "id": "uuid",
  "escrow_id": "uuid",
  "order_id": "uuid",
  "initiator_id": "uuid",
  "reason": "Работа не соответствует ТЗ",
  "status": "open",
  "created_at": "2024-12-03T00:00:00Z"
}
```

### 17.2 Получить спор по заказу

```
GET /api/orders/:id/dispute
```

### 17.3 Список моих споров

```
GET /api/disputes?limit=20&offset=0
```

---

## 18. Жалобы (Reports)

### 18.1 Подать жалобу

```
POST /api/reports
```

**Тело запроса:**
```json
{
  "target_type": "user",
  "target_id": "uuid",
  "reason": "spam",
  "description": "Пользователь рассылает спам в сообщениях"
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| target_type | string | `user`, `order`, `message`, `review` |
| target_id | string | UUID объекта жалобы |
| reason | string | Причина жалобы |
| description | string | Подробное описание (опционально) |

### 18.2 Мои жалобы

```
GET /api/reports?limit=20&offset=0
```

---

## 19. Верификация

### 19.1 Отправить код на email

```
POST /api/verification/email/send
```

**Ответ (200):**
```json
{
  "message": "code sent",
  "code": "123456"
}
```

> ⚠️ В продакшене код не возвращается, только отправляется на email

### 19.2 Отправить код на телефон

```
POST /api/verification/phone/send
```

### 19.3 Подтвердить код

```
POST /api/verification/verify
```

**Тело запроса:**
```json
{
  "type": "email",
  "code": "123456"
}
```

**Ответ (200):**
```json
{
  "verified": true
}
```

### 19.4 Статус верификации

```
GET /api/verification/status
```

**Ответ (200):**
```json
{
  "email_verified": true,
  "phone_verified": false,
  "identity_verified": false
}
```

---

## 20. Шаблоны откликов

### 20.1 Создать шаблон

```
POST /api/proposal-templates
```

**Тело запроса:**
```json
{
  "title": "Стандартный отклик",
  "content": "Здравствуйте! Готов выполнить ваш заказ..."
}
```

### 20.2 Список шаблонов

```
GET /api/proposal-templates
```

### 20.3 Обновить шаблон

```
PUT /api/proposal-templates/:id
```

### 20.4 Удалить шаблон

```
DELETE /api/proposal-templates/:id
```

---

## 21. Поиск фрилансеров

### 21.1 Поиск

```
GET /api/freelancers/search
```

**Query параметры:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| q | string | Поиск по имени, bio, username |
| skills | string | Навыки через запятую: `react,typescript` |
| min_hourly_rate | number | Минимальная ставка |
| max_hourly_rate | number | Максимальная ставка |
| experience_level | string | `junior`, `middle`, `senior` |
| location | string | Локация |
| min_rating | number | Минимальный рейтинг (1-5) |
| limit | number | Лимит (по умолчанию 20) |
| offset | number | Смещение |

**Пример:**
```
GET /api/freelancers/search?skills=react,node&min_rating=4&experience_level=senior
```

**Ответ (200):**
```json
[
  {
    "id": "uuid",
    "username": "johndoe",
    "display_name": "John Doe",
    "bio": "Full-stack разработчик",
    "hourly_rate": 2500,
    "experience_level": "senior",
    "skills": ["react", "node", "typescript"],
    "location": "Москва",
    "photo_id": "uuid",
    "avg_rating": 4.8,
    "review_count": 15,
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

---

## 22. Seed данные (только development)

> ⚠️ Эти эндпоинты доступны только в режиме разработки (APP_ENV=development)

### 22.1 Базовый seed

```
GET /api/seed?num_users=50&num_orders=100
POST /api/seed
```

**Тело запроса (POST):**
```json
{
  "num_users": 50,
  "num_orders": 100
}
```

**Ответ (200):**
```json
{
  "message": "Seed data generated successfully",
  "num_users": 50,
  "num_orders": 100,
  "accounts": [
    {
      "email": "aleksandr.ivanov42@gmail.com",
      "username": "Aleksandr_Ivanov_123",
      "password": "Password123",
      "role": "freelancer"
    }
  ]
}
```

### 22.2 Реалистичный seed

Генерирует данные, имитирующие реальную активность пользователей:
- 15 пользователей (5 клиентов, 10 фрилансеров)
- Профили с реалистичными данными
- 20 заказов в разных статусах
- Отклики на заказы (2-4 на каждый)
- Принятые отклики и работа в процессе
- Завершённые заказы с отзывами
- Пополненные балансы клиентов
- Избранные заказы и фрилансеры
- Шаблоны откликов для фрилансеров

```
GET /api/seed/realistic
POST /api/seed/realistic
```

**Ответ (200):**
```json
{
  "message": "Realistic seed data generated successfully",
  "accounts": [...],
  "orders_created": 20,
  "proposals_created": 45,
  "reviews_created": 12
}
```

---

## Приложение A: Модели данных (дополнение)

### Withdrawal
```typescript
interface Withdrawal {
  id: string;
  user_id: string;
  amount: number;
  status: 'pending' | 'processing' | 'completed' | 'rejected';
  card_last4?: string;
  bank_name?: string;
  rejection_reason?: string;
  created_at: string;
  processed_at?: string;
}
```

### Favorite
```typescript
interface Favorite {
  id: string;
  user_id: string;
  target_type: 'order' | 'freelancer';
  target_id: string;
  created_at: string;
}
```

### Dispute
```typescript
interface Dispute {
  id: string;
  escrow_id: string;
  order_id: string;
  initiator_id: string;
  reason: string;
  status: 'open' | 'under_review' | 'resolved_client' | 'resolved_freelancer' | 'cancelled';
  resolution?: string;
  resolved_by?: string;
  created_at: string;
  resolved_at?: string;
}
```

### Report
```typescript
interface Report {
  id: string;
  reporter_id: string;
  target_type: 'user' | 'order' | 'message' | 'review';
  target_id: string;
  reason: string;
  description?: string;
  status: 'pending' | 'reviewed' | 'action_taken' | 'dismissed';
  reviewed_by?: string;
  reviewed_at?: string;
  created_at: string;
}
```

### ProposalTemplate
```typescript
interface ProposalTemplate {
  id: string;
  user_id: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}
```

### FreelancerSearchResult
```typescript
interface FreelancerSearchResult {
  id: string;
  username: string;
  display_name?: string;
  bio?: string;
  hourly_rate?: number;
  experience_level?: string;
  skills?: string[];
  location?: string;
  photo_id?: string;
  avg_rating: number;
  review_count: number;
  created_at: string;
}
```

---

*Документация актуальна на декабрь 2024*
