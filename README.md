# freelance_AI_back

Бэкенд для платформы фриланса с AI-ассистентом.

## Технологии

- Go 1.21+
- PostgreSQL
- Gin Web Framework
- JWT аутентификация
- WebSocket

## Установка

1. Установите зависимости:
```bash
go mod download
```

2. Настройте базу данных PostgreSQL и создайте файл `.env` на основе `example_env`

3. Запустите миграции:
```bash
# Миграции выполняются автоматически при запуске сервера
```

## Запуск

```bash
go run cmd/server/main.go
```

Сервер будет доступен на `http://localhost:8080`

## API Документация

Подробная документация API доступна в файле [API_DOCUMENTATION.md](./API_DOCUMENTATION.md)

## Переменные окружения

См. файл `example_env` для списка необходимых переменных окружения.

### Настройка на хостинге

При деплое на хостинг необходимо установить следующие переменные окружения:

**Обязательные переменные для подключения к базе данных:**
```bash
POSTGRESQL_HOST=85.92.110.79
POSTGRESQL_PORT=5432
POSTGRESQL_USER=gen_user
POSTGRESQL_PASSWORD=e}?3{G?&ljO,|O
POSTGRESQL_DBNAME=freelance_ai
```

**Или можно использовать одну переменную:**
```bash
DATABASE_URL=postgres://gen_user:e}?3{G?&ljO,|O@85.92.110.79:5432/freelance_ai?sslmode=disable
```

**Другие важные переменные:**
```bash
APP_ENV=production
HTTP_PORT=8080
JWT_SECRET=your-secret-key-minimum-32-characters-long
REFRESH_SECRET=your-refresh-secret-minimum-32-characters-long
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

**Примечание:** Если переменные окружения не установлены, приложение будет пытаться подключиться к `localhost:5432`, что приведет к ошибке подключения на хостинге.

