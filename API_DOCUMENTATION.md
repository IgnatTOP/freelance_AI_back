# API Документация - Freelance Platform Backend

## Содержание

1. [Общая информация](#общая-информация)
2. [Аутентификация](#аутентификация)
3. [Модели данных](#модели-данных)
4. [Публичные эндпоинты](#публичные-эндпоинты)
5. [Профиль](#профиль)
6. [Заказы](#заказы)
7. [Предложения](#предложения)
8. [Чаты и сообщения](#чаты-и-сообщения)
9. [Портфолио](#портфолио)
10. [Медиа файлы](#медиа-файлы)
11. [Уведомления](#уведомления)
12. [Статистика](#статистика)
13. [AI эндпоинты](#ai-эндпоинты)
14. [WebSocket](#websocket)
15. [Коды ошибок](#коды-ошибок)

---

## Общая информация

### Базовый URL

```
http://localhost:8080/api
```

### Статические файлы (медиа)

```
http://localhost:8080/media/{file_path}
```

### Формат данных

- Все запросы и ответы: `application/json`
- Даты в формате RFC3339: `2024-01-15T10:30:00Z`
- UUID v4 для всех идентификаторов

### Заголовки

```http
Content-Type: application/json
Authorization: Bearer <access_token>  # для защищённых эндпоинтов
```

### Rate Limiting

| Эндпоинты | Лимит | Период |
|-----------|-------|--------|
| `/auth/*` | 5 запросов | 1 минута |
| Остальные | 10 запросов | 1 минута |

### Роли пользователей

| Роль | Описание |
|------|----------|
| `client` | Заказчик. Создаёт заказы, управляет предложениями |
| `freelancer` | Исполнитель. Откликается на заказы, выполняет работу |

---

## Аутентификация

Система использует JWT токены (access + refresh).

| Токен | TTL | Назначение |
|-------|-----|------------|
| Access Token | 15 минут | Доступ к API |
| Refresh Token | 30 дней | Обновление access токена |

### POST /auth/register

Регистрация нового пользователя.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "username": "john_doe",
  "role": "freelancer",
  "display_name": "John Doe"
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| email | string | ✅ | Email (уникальный) |
| password | string | ✅ | Пароль (мин. 6 символов) |
| username | string | ❌ | Username (уникальный) |
| role | string | ❌ | `client` или `freelancer` (default: `freelancer`) |
| display_name | string | ❌ | Отображаемое имя |

**Response: 201 Created**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "john_doe",
    "role": "freelancer",
    "is_active": true,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "display_name": "John Doe",
    "experience_level": "junior",
    "skills": [],
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

---

### POST /auth/login

Авторизация пользователя.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response: 200 OK**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "john_doe",
    "role": "freelancer",
    "is_active": true,
    "last_login_at": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-10T08:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "display_name": "John Doe",
    "bio": "Experienced developer",
    "hourly_rate": 50.00,
    "experience_level": "middle",
    "skills": ["JavaScript", "Vue.js", "Go"],
    "location": "Moscow",
    "photo_id": "660e8400-e29b-41d4-a716-446655440001",
    "updated_at": "2024-01-14T15:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

---

### POST /auth/refresh

Обновление access токена.

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response: 200 OK**
```json
{
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

---

### GET /auth/sessions 🔒

Получить список активных сессий.

**Response: 200 OK**
```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_agent": "Mozilla/5.0...",
    "ip_address": "192.168.1.1",
    "expires_at": "2024-02-14T10:30:00Z",
    "created_at": "2024-01-15T10:30:00Z"
  }
]
```

---

### DELETE /auth/sessions/:id 🔒

Удалить конкретную сессию.

**Response: 200 OK**
```json
{
  "message": "сессия успешно удалена"
}
```

---

### DELETE /auth/sessions 🔒

Удалить все сессии кроме текущей.

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response: 200 OK**
```json
{
  "message": "все сессии кроме текущей успешно удалены"
}
```

---

## Модели данных

### User
```typescript
interface User {
  id: string;                // UUID
  email: string;
  username: string;
  role: "client" | "freelancer";
  is_active: boolean;
  last_login_at?: string;    // RFC3339
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
  experience_level: "junior" | "middle" | "senior";
  skills: string[];
  location?: string;
  photo_id?: string;         // UUID медиафайла
  ai_summary?: string;
  updated_at: string;
}
```

### Order
```typescript
interface Order {
  id: string;
  client_id: string;
  title: string;
  description: string;
  budget_min?: number;
  budget_max?: number;
  status: "draft" | "published" | "in_progress" | "completed" | "cancelled";
  deadline_at?: string;
  ai_summary?: string;
  best_recommendation_proposal_id?: string;
  best_recommendation_justification?: string;
  created_at: string;
  updated_at: string;
  attachments?: OrderAttachment[];
}

interface OrderRequirement {
  id: string;
  order_id: string;
  skill: string;
  level: "junior" | "middle" | "senior";
}

interface OrderAttachment {
  id: string;
  order_id: string;
  media_id: string;
  created_at: string;
  media?: MediaFile;
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
  status: "pending" | "shortlisted" | "accepted" | "rejected";
  ai_feedback?: string;
  created_at: string;
  updated_at: string;
}
```

### Conversation & Message
```typescript
interface Conversation {
  id: string;
  order_id?: string;
  client_id: string;
  freelancer_id: string;
  created_at: string;
}

interface Message {
  id: string;
  conversation_id: string;
  author_type: "client" | "freelancer" | "system" | "assistant";
  author_id?: string;
  content: string;
  ai_metadata?: object;
  created_at: string;
}
```

### MediaFile
```typescript
interface MediaFile {
  id: string;
  user_id?: string;
  file_path: string;
  file_type: string;         // MIME type
  file_size: number;         // bytes
  is_public: boolean;
  created_at: string;
}
```

### PortfolioItem
```typescript
interface PortfolioItem {
  id: string;
  user_id: string;
  title: string;
  description?: string;
  cover_media_id?: string;
  ai_tags: string[];
  external_link?: string;
  created_at: string;
}
```

### Notification
```typescript
interface Notification {
  id: string;
  user_id: string;
  payload: object;
  is_read: boolean;
  created_at: string;
}
```

### Review
```typescript
interface Review {
  id: string;
  order_id: string;
  reviewer_id: string;
  reviewed_id: string;
  rating: number;            // 1-5
  comment?: string;
  created_at: string;
  updated_at: string;
}
```

---

## Публичные эндпоинты

### GET /health

Проверка работоспособности сервера.

**Response: 200 OK**
```json
{
  "status": "ok"
}
```

---

### GET /orders

Получить список заказов с фильтрацией и пагинацией.

**Query Parameters:**

| Параметр | Тип | Default | Описание |
|----------|-----|---------|----------|
| status | string | - | `open`/`published`, `in_progress`, `completed`, `cancelled`, `draft` |
| search | string | - | Поиск по заголовку и описанию |
| skills | string | - | Навыки через запятую: `JavaScript,Go` |
| budget_min | number | - | Минимальный бюджет |
| budget_max | number | - | Максимальный бюджет |
| sort_by | string | `created_at` | Поле сортировки |
| sort_order | string | `desc` | `asc` или `desc` |
| limit | number | 20 | Количество записей |
| offset | number | 0 | Смещение |

**Response: 200 OK**
```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "client_id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Разработка веб-приложения",
      "description": "Нужно разработать SPA...",
      "budget_min": 50000,
      "budget_max": 100000,
      "status": "published",
      "deadline_at": "2024-02-15T00:00:00Z",
      "ai_summary": "Проект веб-разработки...",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "total": 45,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

---

### GET /orders/:id

Получить заказ по ID с требованиями и вложениями.

**Response: 200 OK**
```json
{
  "id": "880e8400-e29b-41d4-a716-446655440000",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Разработка веб-приложения",
  "description": "Полное описание...",
  "budget_min": 50000,
  "budget_max": 100000,
  "status": "published",
  "deadline_at": "2024-02-15T00:00:00Z",
  "ai_summary": "Краткое резюме от AI...",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "requirements": [
    {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "order_id": "880e8400-e29b-41d4-a716-446655440000",
      "skill": "Vue.js",
      "level": "middle"
    },
    {
      "id": "990e8400-e29b-41d4-a716-446655440001",
      "order_id": "880e8400-e29b-41d4-a716-446655440000",
      "skill": "Go",
      "level": "senior"
    }
  ],
  "attachments": [
    {
      "id": "aa0e8400-e29b-41d4-a716-446655440000",
      "order_id": "880e8400-e29b-41d4-a716-446655440000",
      "media_id": "bb0e8400-e29b-41d4-a716-446655440000",
      "created_at": "2024-01-15T10:30:00Z",
      "media": {
        "id": "bb0e8400-e29b-41d4-a716-446655440000",
        "file_path": "photos/2024/01/image.png",
        "file_type": "image/png",
        "file_size": 102400,
        "is_public": true,
        "created_at": "2024-01-15T10:30:00Z"
      }
    }
  ]
}
```

---

### GET /users/:id

Получить публичный профиль пользователя.

**Response: 200 OK**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "role": "freelancer"
  },
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "display_name": "John Doe",
    "bio": "Опытный веб-разработчик...",
    "hourly_rate": 50.00,
    "experience_level": "senior",
    "skills": ["JavaScript", "Vue.js", "Go"],
    "location": "Москва",
    "photo_id": "660e8400-e29b-41d4-a716-446655440001",
    "ai_summary": "Высококвалифицированный специалист...",
    "updated_at": "2024-01-14T15:00:00Z"
  },
  "stats": {
    "total_orders": 25,
    "completed_orders": 20,
    "average_rating": 4.8,
    "total_reviews": 18
  },
  "reviews": [
    {
      "id": "cc0e8400-e29b-41d4-a716-446655440000",
      "order_id": "880e8400-e29b-41d4-a716-446655440000",
      "reviewer_id": "dd0e8400-e29b-41d4-a716-446655440000",
      "reviewed_id": "550e8400-e29b-41d4-a716-446655440000",
      "rating": 5,
      "comment": "Отличная работа!",
      "created_at": "2024-01-10T12:00:00Z"
    }
  ],
  "completed_orders": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440001",
      "title": "Разработка лендинга",
      "status": "completed"
    }
  ]
}
```

---

### GET /users/:id/portfolio

Получить портфолио пользователя.

**Response: 200 OK**
```json
[
  {
    "id": "ee0e8400-e29b-41d4-a716-446655440000",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "E-commerce платформа",
    "description": "Интернет-магазин...",
    "cover_media_id": "ff0e8400-e29b-41d4-a716-446655440000",
    "ai_tags": ["Vue.js", "E-commerce"],
    "external_link": "https://example.com",
    "created_at": "2024-01-05T10:00:00Z",
    "media": [
      {
        "id": "ff0e8400-e29b-41d4-a716-446655440000",
        "file_path": "photos/2024/01/portfolio1.png",
        "file_type": "image/png",
        "file_size": 204800,
        "is_public": true,
        "created_at": "2024-01-05T10:00:00Z"
      }
    ]
  }
]
```

---

## Профиль

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### GET /profile 🔒

Получить профиль текущего пользователя.

**Response: 200 OK**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "display_name": "John Doe",
  "bio": "Опытный веб-разработчик",
  "hourly_rate": 50.00,
  "experience_level": "middle",
  "skills": ["JavaScript", "Vue.js", "Go"],
  "location": "Москва",
  "photo_id": "660e8400-e29b-41d4-a716-446655440001",
  "ai_summary": "Квалифицированный разработчик...",
  "updated_at": "2024-01-14T15:00:00Z"
}
```

---

### PUT /profile 🔒

Обновить профиль текущего пользователя.

**Request Body:**
```json
{
  "display_name": "John Doe",
  "bio": "Опытный веб-разработчик с 5-летним стажем",
  "hourly_rate": 75.00,
  "experience_level": "senior",
  "skills": ["JavaScript", "TypeScript", "Vue.js", "Go"],
  "location": "Москва, Россия",
  "photo_id": "660e8400-e29b-41d4-a716-446655440001",
  "ai_summary": "Full-stack разработчик..."
}
```

| Поле | Тип | Обязательно | Ограничения |
|------|-----|-------------|-------------|
| display_name | string | ✅ | 2-100 символов |
| bio | string | ❌ | макс. 1000 символов |
| hourly_rate | number | ❌ | >= 0 |
| experience_level | string | ❌ | `junior`, `middle`, `senior` |
| skills | string[] | ❌ | - |
| location | string | ❌ | - |
| photo_id | string | ❌ | UUID загруженного медиафайла |
| ai_summary | string | ❌ | - |

**Response: 200 OK** — возвращает обновлённый профиль

---

### PUT /users/me/role 🔒

Изменить роль пользователя.

**Request Body:**
```json
{
  "role": "client"
}
```

**Response: 200 OK**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "john_doe",
  "role": "client",
  "is_active": true,
  "created_at": "2024-01-10T08:00:00Z",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

## Заказы

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### POST /orders 🔒

Создать новый заказ. **Только для роли `client`**.

**Request Body:**
```json
{
  "title": "Разработка веб-приложения",
  "description": "Требуется разработать SPA приложение...",
  "budget_min": 50000,
  "budget_max": 100000,
  "deadline_at": "2024-02-15T00:00:00Z",
  "requirements": [
    { "skill": "Vue.js", "level": "middle" },
    { "skill": "Go", "level": "senior" }
  ],
  "attachment_ids": ["bb0e8400-e29b-41d4-a716-446655440000"]
}
```

| Поле | Тип | Обязательно | Ограничения |
|------|-----|-------------|-------------|
| title | string | ✅ | 3-200 символов |
| description | string | ✅ | 10-5000 символов |
| budget_min | number | ❌ | - |
| budget_max | number | ❌ | - |
| deadline_at | string | ❌ | RFC3339 |
| requirements | array | ❌ | - |
| requirements[].skill | string | ✅ | - |
| requirements[].level | string | ❌ | `junior`, `middle` (default), `senior` |
| attachment_ids | string[] | ❌ | UUID загруженных файлов |

**Response: 201 Created** — возвращает созданный заказ

---

### GET /orders/my 🔒

Получить свои заказы.

**Response: 200 OK**
```json
{
  "as_client": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "title": "Мой заказ",
      "status": "published"
    }
  ],
  "as_freelancer": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440001",
      "title": "Заказ где я исполнитель",
      "status": "in_progress"
    }
  ]
}
```

---

### PUT /orders/:id 🔒

Обновить заказ. **Только владелец**.

**Request Body:**
```json
{
  "title": "Обновлённый заголовок",
  "description": "Обновлённое описание...",
  "budget_min": 60000,
  "budget_max": 120000,
  "deadline_at": "2024-03-01T00:00:00Z",
  "status": "published",
  "requirements": [
    { "skill": "Vue.js", "level": "senior" }
  ],
  "attachment_ids": []
}
```

**Response: 200 OK**
```json
{
  "id": "880e8400-e29b-41d4-a716-446655440000",
  "title": "Обновлённый заголовок",
  "requirements": [...],
  "attachments": [...]
}
```

---

### DELETE /orders/:id 🔒

Удалить заказ. **Только владелец**.

**Response: 200 OK**
```json
{
  "message": "заказ успешно удалён"
}
```

---

## Предложения

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### POST /orders/:id/proposals 🔒

Создать предложение. **Только для роли `freelancer`**.

**Request Body:**
```json
{
  "cover_letter": "Здравствуйте! Готов взяться за проект...",
  "amount": 75000
}
```

| Поле | Тип | Обязательно | Ограничения |
|------|-----|-------------|-------------|
| cover_letter | string | ✅ | 10-2000 символов |
| amount | number | ❌ | Предлагаемая сумма |

**Response: 201 Created**
```json
{
  "id": "110e8400-e29b-41d4-a716-446655440000",
  "order_id": "880e8400-e29b-41d4-a716-446655440000",
  "freelancer_id": "550e8400-e29b-41d4-a716-446655440000",
  "cover_letter": "Здравствуйте!...",
  "proposed_amount": 75000,
  "status": "pending",
  "created_at": "2024-01-15T12:00:00Z",
  "updated_at": "2024-01-15T12:00:00Z"
}
```

---

### GET /orders/:id/proposals 🔒

Получить предложения на заказ.

- **Владелец заказа** — видит все предложения + AI рекомендацию
- **Фрилансер** — видит только если подал предложение

**Response для владельца: 200 OK**
```json
{
  "proposals": [
    {
      "id": "110e8400-e29b-41d4-a716-446655440000",
      "order_id": "880e8400-e29b-41d4-a716-446655440000",
      "freelancer_id": "550e8400-e29b-41d4-a716-446655440000",
      "cover_letter": "...",
      "proposed_amount": 75000,
      "status": "pending",
      "ai_feedback": "Краткая AI-сводка по отклику...",
      "created_at": "2024-01-15T12:00:00Z"
    }
  ],
  "best_recommendation": {
    "proposal_id": "110e8400-e29b-41d4-a716-446655440000",
    "justification": "Данный исполнитель имеет наибольший опыт..."
  }
}
```

---

### GET /orders/:id/my-proposal 🔒

Получить своё предложение на заказ.

**Response: 200 OK**
```json
{
  "id": "110e8400-e29b-41d4-a716-446655440000",
  "order_id": "880e8400-e29b-41d4-a716-446655440000",
  "freelancer_id": "550e8400-e29b-41d4-a716-446655440000",
  "cover_letter": "...",
  "proposed_amount": 75000,
  "status": "pending",
  "created_at": "2024-01-15T12:00:00Z"
}
```

---

### PUT /orders/:id/proposals/:proposalId/status 🔒

Изменить статус предложения. **Только владелец заказа**.

**Request Body:**
```json
{
  "status": "accepted"
}
```

| Статус | Описание |
|--------|----------|
| `pending` | Ожидает рассмотрения |
| `shortlisted` | В шорт-листе |
| `accepted` | Принято |
| `rejected` | Отклонено |

**Response: 200 OK**
```json
{
  "proposal": { "id": "...", "status": "accepted" },
  "conversation": { "id": "...", "order_id": "...", "client_id": "...", "freelancer_id": "..." },
  "order": { "id": "...", "title": "..." }
}
```

---

### GET /proposals/my 🔒

Получить все свои предложения.

**Response: 200 OK**
```json
[
  {
    "id": "110e8400-e29b-41d4-a716-446655440000",
    "order_id": "880e8400-e29b-41d4-a716-446655440000",
    "cover_letter": "...",
    "proposed_amount": 75000,
    "status": "pending",
    "order": { "id": "...", "title": "...", "status": "published" }
  }
]
```

---

## Чаты и сообщения

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### GET /orders/:id/conversations/:participantId 🔒

Получить/создать чат с участником по заказу.

**Response: 200 OK**
```json
{
  "conversation": {
    "id": "220e8400-e29b-41d4-a716-446655440000",
    "order_id": "880e8400-e29b-41d4-a716-446655440000",
    "client_id": "...",
    "freelancer_id": "..."
  },
  "messages": [
    {
      "id": "...",
      "author_type": "client",
      "author_id": "...",
      "content": "Привет!",
      "created_at": "2024-01-15T12:00:00Z"
    }
  ]
}
```

---

### GET /conversations/my 🔒

Получить все свои чаты.

**Response: 200 OK**
```json
[
  {
    "id": "220e8400-e29b-41d4-a716-446655440000",
    "order_id": "880e8400-e29b-41d4-a716-446655440000",
    "order_title": "Разработка веб-приложения",
    "other_user": { "id": "...", "display_name": "John Doe", "photo_id": "..." },
    "last_message": { "content": "Последнее сообщение", "created_at": "..." }
  }
]
```

---

### GET /conversations/:conversationId/messages 🔒

Получить сообщения чата.

**Query Parameters:** `limit` (default: 50), `offset` (default: 0)

**Response: 200 OK**
```json
{
  "conversation": { "id": "...", "order_id": "...", "client_id": "...", "freelancer_id": "..." },
  "messages": [...],
  "order": { "id": "...", "title": "..." },
  "other_user": { "id": "...", "display_name": "...", "photo_id": "..." }
}
```

---

### POST /conversations/:conversationId/messages 🔒

Отправить сообщение.

**Request Body:**
```json
{ "content": "Текст сообщения" }
```

**Response: 201 Created**
```json
{
  "message": { "id": "...", "author_type": "client", "content": "...", "created_at": "..." }
}
```

---

### PUT /conversations/:conversationId/messages/:messageId 🔒

Редактировать сообщение. **Только автор**.

**Request Body:** `{ "content": "Новый текст" }`

---

### DELETE /conversations/:conversationId/messages/:messageId 🔒

Удалить сообщение. **Только автор**.

---

## Портфолио

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### GET /portfolio 🔒

Получить своё портфолио.

**Response: 200 OK**
```json
[
  {
    "id": "ee0e8400-e29b-41d4-a716-446655440000",
    "title": "Проект 1",
    "description": "Описание",
    "cover_media_id": "...",
    "ai_tags": ["web", "react"],
    "external_link": "https://example.com",
    "created_at": "2024-01-05T10:00:00Z"
  }
]
```

---

### POST /portfolio 🔒

Создать работу в портфолио.

**Request Body:**
```json
{
  "title": "Проект 1",
  "description": "Описание проекта",
  "cover_media_id": "uuid",
  "ai_tags": ["web", "react"],
  "external_link": "https://example.com",
  "media_ids": ["uuid1", "uuid2"]
}
```

---

### GET /portfolio/:id 🔒

Получить работу из портфолио.

**Response: 200 OK**
```json
{
  "id": "...",
  "title": "...",
  "description": "...",
  "media": [{ "id": "...", "file_path": "...", "file_type": "image/png" }]
}
```

---

### PUT /portfolio/:id 🔒

Обновить работу. **Только владелец**.

---

### DELETE /portfolio/:id 🔒

Удалить работу. **Только владелец**.

---

## Медиа файлы

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### POST /media/photos 🔒

Загрузить изображение.

**Content-Type:** `multipart/form-data`

**Form Data:** `file` - файл изображения

**Поддерживаемые форматы:** JPEG, PNG, GIF, WebP, SVG

**Максимальный размер:** 15 MB

**Response: 201 Created**
```json
{
  "id": "bb0e8400-e29b-41d4-a716-446655440000",
  "file_path": "photos/2024/01/image.png",
  "file_type": "image/png",
  "file_size": 102400,
  "is_public": true
}
```

**Доступ к файлу:** `http://localhost:8080/media/{file_path}`

---

### DELETE /media/:id 🔒

Удалить медиа файл. **Только владелец**.

**Response: 204 No Content**

---

## Уведомления

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

### GET /notifications 🔒

Получить уведомления.

**Query Parameters:**
- `limit` (default: 20)
- `offset` (default: 0)
- `unread_only` (`true`/`false`)

**Response: 200 OK**
```json
[
  {
    "id": "...",
    "payload": { "type": "proposal.new", "message": "Получено новое предложение" },
    "is_read": false,
    "created_at": "2024-01-15T12:00:00Z"
  }
]
```

---

### GET /notifications/unread/count 🔒

**Response:** `{ "count": 5 }`

---

### GET /notifications/:id 🔒

Получить уведомление по ID.

---

### PUT /notifications/:id/read 🔒

Отметить как прочитанное.

---

### PUT /notifications/read-all 🔒

Отметить все как прочитанные.

---

### DELETE /notifications/:id 🔒

Удалить уведомление.

---

## Статистика

### GET /stats 🔒

Получить свою статистику.

**Response: 200 OK**
```json
{
  "orders": {
    "total": 10,
    "open": 3,
    "in_progress": 2,
    "completed": 5,
    "total_proposals": 15
  },
  "proposals": {
    "total": 20,
    "pending": 5,
    "accepted": 10,
    "rejected": 5
  },
  "balance": 0,
  "average_rating": 0.0
}
```

---

## AI эндпоинты

> 🔒 Требуется заголовок `Authorization: Bearer <access_token>`

Все AI эндпоинты имеют обычную и потоковую (stream) версию. Потоковые версии используют SSE (Server-Sent Events).

### POST /ai/orders/description 🔒 (client)

Сгенерировать описание заказа.

**Request:** `{ "title": "...", "description": "...", "skills": ["Go", "React"] }`

**Response:** `{ "description": "AI-сгенерированное описание..." }`

**Потоковая версия:** `POST /ai/orders/description/stream`

---

### POST /ai/orders/:id/proposal 🔒 (freelancer)

Сгенерировать предложение к заказу.

**Request (опционально):**
```json
{
  "user_skills": ["Go", "React"],
  "user_experience": "senior",
  "user_bio": "...",
  "portfolio": [{ "title": "...", "description": "...", "ai_tags": [...] }]
}
```

**Response:** `{ "proposal": "Сгенерированное сопроводительное письмо..." }`

**Потоковая версия:** `POST /ai/orders/:id/proposal/stream`

---

### GET /ai/orders/:id/proposals/feedback 🔒 (freelancer)

Получить рекомендации по улучшению своего предложения.

**Response:** `{ "feedback": "Рекомендации по улучшению..." }`

**Потоковая версия:** `GET /ai/orders/:id/proposals/feedback/stream`

---

### POST /ai/orders/improve 🔒 (client)

Улучшить описание заказа.

**Request:** `{ "title": "...", "description": "..." }`

**Response:** `{ "description": "Улучшенное описание..." }`

**Потоковая версия:** `POST /ai/orders/improve/stream`

---

### POST /ai/orders/:id/regenerate-summary 🔒

Регенерировать AI-сводку заказа. **Только владелец**.

**Response:** Обновлённый заказ с новым `ai_summary`

**Потоковая версия:** `POST /ai/orders/:id/regenerate-summary/stream`

---

### GET /ai/conversations/:conversationId/summary 🔒

Создать резюме переписки.

**Response:**
```json
{
  "summary": "Краткое описание...",
  "next_steps": ["Шаг 1", "Шаг 2"],
  "agreements": ["Договорённость 1"],
  "open_questions": ["Вопрос 1"]
}
```

**Потоковая версия:** `GET /ai/conversations/:conversationId/summary/stream`

---

### GET /ai/orders/recommended 🔒 (freelancer)

Рекомендовать подходящие заказы.

**Query:** `limit` (default: 50)

**Response:** `{ "recommended_order_ids": [...], "explanation": "..." }`

**Потоковая версия:** `GET /ai/orders/recommended/stream`

---

### GET /ai/orders/:id/price-timeline 🔒 (freelancer)

Рекомендация цены и сроков.

**Response:**
```json
{
  "recommended_amount": 75000,
  "min_amount": 50000,
  "max_amount": 100000,
  "recommended_days": 30,
  "min_days": 20,
  "max_days": 45,
  "explanation": "..."
}
```

**Потоковая версия:** `GET /ai/orders/:id/price-timeline/stream`

---

### GET /ai/orders/:id/quality 🔒

Оценка качества заказа.

**Response:**
```json
{
  "score": 85,
  "strengths": ["Чёткое ТЗ", "Адекватный бюджет"],
  "weaknesses": ["Нет дедлайна"],
  "recommendations": ["Добавьте срок выполнения"]
}
```

**Потоковая версия:** `GET /ai/orders/:id/quality/stream`

---

### GET /ai/orders/:id/suitable-freelancers 🔒 (client)

Найти подходящих фрилансеров.

**Query:** `limit` (default: 10)

**Response:**
```json
{
  "freelancers": [
    { "user_id": "...", "match_score": 0.95, "explanation": "..." }
  ]
}
```

**Потоковая версия:** `GET /ai/orders/:id/suitable-freelancers/stream`

---

### POST /ai/assistant 🔒

AI чат-помощник.

**Request:**
```json
{
  "message": "Вопрос к AI",
  "context_data": { "order_id": "...", "additional_info": "..." }
}
```

**Response:** `{ "response": "Ответ AI..." }`

**Потоковая версия:** `POST /ai/assistant/stream`

---

### POST /ai/profile/improve 🔒

Улучшить описание профиля.

**Request:** `{ "current_bio": "...", "skills": [...], "experience_level": "..." }`

**Response:** `{ "improved_bio": "..." }`

**Потоковая версия:** `POST /ai/profile/improve/stream`

---

### POST /ai/portfolio/improve 🔒

Улучшить описание работы в портфолио.

**Request:** `{ "title": "...", "description": "...", "ai_tags": [...] }`

**Response:** `{ "improved_description": "..." }`

**Потоковая версия:** `POST /ai/portfolio/improve/stream`

---

## WebSocket

### GET /ws?token=ACCESS_TOKEN

Подключение к WebSocket для получения real-time уведомлений.

```javascript
const ws = new WebSocket('ws://localhost:8080/api/ws?token=ACCESS_TOKEN');

ws.onmessage = (event) => {
  const { type, data } = JSON.parse(event.data);
  // Обработка события
};
```

### События

| Событие | Описание | Данные |
|---------|----------|--------|
| `orders.new` | Новый заказ создан | `{ order, message }` |
| `orders.updated` | Заказ обновлён | `{ order, requirements, attachments }` |
| `proposals.new` | Новое предложение | `{ order, proposal, message }` |
| `proposals.sent` | Предложение отправлено | `{ order, proposal, message }` |
| `proposals.updated` | Статус предложения изменён | `{ proposal, conversation, order, message }` |
| `chat.message` | Новое сообщение в чате | `{ message, conversation, order }` |
| `profile.updated` | Профиль обновлён | `{ profile, message }` |
| `notification` | Новое уведомление | `{ id, payload, created_at }` |

---

## Коды ошибок

| Код | Описание |
|-----|----------|
| 200 | Успешный запрос |
| 201 | Ресурс создан |
| 204 | Успешное удаление (без тела) |
| 400 | Ошибка валидации / неверные данные |
| 401 | Не авторизован / токен истёк |
| 403 | Доступ запрещён |
| 404 | Ресурс не найден |
| 429 | Превышен лимит запросов |
| 500 | Внутренняя ошибка сервера |

### Формат ошибки

```json
{
  "error": "Описание ошибки"
}
```

---

## Примеры использования

### Авторизация и запрос

```javascript
// Регистрация
const { tokens } = await fetch('/api/auth/register', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password123',
    role: 'freelancer'
  })
}).then(r => r.json());

// Запрос с авторизацией
const profile = await fetch('/api/profile', {
  headers: { 'Authorization': `Bearer ${tokens.access_token}` }
}).then(r => r.json());

// Обновление токена при истечении
const { tokens: newTokens } = await fetch('/api/auth/refresh', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ refresh_token: tokens.refresh_token })
}).then(r => r.json());
```

### Загрузка файла

```javascript
const formData = new FormData();
formData.append('file', fileInput.files[0]);

const media = await fetch('/api/media/photos', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${accessToken}` },
  body: formData
}).then(r => r.json());

// URL файла
const imageUrl = `http://localhost:8080/media/${media.file_path}`;
```

### SSE (AI Streaming)

```javascript
const eventSource = new EventSource(
  `/api/ai/orders/description/stream?token=${accessToken}`,
  { method: 'POST', body: JSON.stringify({ title: '...' }) }
);

// Или через fetch
const response = await fetch('/api/ai/orders/description/stream', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ title: '...', description: '...', skills: [] })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  const chunk = decoder.decode(value);
  // chunk содержит "data: текст\n\n"
  console.log(chunk);
}
```

---

**Версия документации:** 1.0  
**Последнее обновление:** 2024
