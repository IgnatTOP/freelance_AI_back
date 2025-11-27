#!/bin/bash
# Bash скрипт для сброса базы данных
# Использование: ./reset_db.sh

# Загружаем переменные окружения из .env
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Получаем DATABASE_URL из переменных окружения
DATABASE_URL=${DATABASE_URL:-"postgres://postgres:123@localhost:5432/freelance_ai?sslmode=disable"}

# Парсим DATABASE_URL
if [[ $DATABASE_URL =~ postgres://([^:]+):([^@]+)@([^:]+):([0-9]+)/([^?]+) ]]; then
    USERNAME="${BASH_REMATCH[1]}"
    PASSWORD="${BASH_REMATCH[2]}"
    HOST="${BASH_REMATCH[3]}"
    PORT="${BASH_REMATCH[4]}"
    DATABASE="${BASH_REMATCH[5]}"
    
    echo "Подключение к базе данных: $HOST:$PORT/$DATABASE"
    
    # Проверяем наличие psql
    if ! command -v psql &> /dev/null; then
        echo "ОШИБКА: psql не найден. Установите PostgreSQL или добавьте его в PATH."
        exit 1
    fi
    
    # Устанавливаем переменную окружения для пароля
    export PGPASSWORD="$PASSWORD"
    
    # Получаем путь к скрипту
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SQL_SCRIPT="$SCRIPT_DIR/reset_db.sql"
    
    # Выполняем SQL скрипт
    echo "Выполняю сброс базы данных..."
    if psql -h "$HOST" -p "$PORT" -U "$USERNAME" -d "$DATABASE" -f "$SQL_SCRIPT"; then
        echo "База данных успешно сброшена!"
        echo "Запустите сервер, чтобы применить миграции заново."
    else
        echo "ОШИБКА при сбросе базы данных"
        exit 1
    fi
    
    # Очищаем пароль из переменных окружения
    unset PGPASSWORD
else
    echo "ОШИБКА: Неверный формат DATABASE_URL"
    echo "Ожидается формат: postgres://user:password@host:port/database"
    exit 1
fi

