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

