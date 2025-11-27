#!/bin/bash
# Скрипт для проверки конфигурации базы данных

echo "Проверка переменных окружения для подключения к базе данных..."
echo ""

# Проверяем наличие переменных
if [ -n "$DATABASE_URL" ]; then
    echo "✓ DATABASE_URL установлен"
    echo "  Host: $(echo $DATABASE_URL | sed -E 's|.*@([^:]+):.*|\1|')"
else
    echo "✗ DATABASE_URL не установлен"
fi

if [ -n "$POSTGRESQL_HOST" ]; then
    echo "✓ POSTGRESQL_HOST=$POSTGRESQL_HOST"
else
    echo "✗ POSTGRESQL_HOST не установлен"
fi

if [ -n "$POSTGRESQL_USER" ]; then
    echo "✓ POSTGRESQL_USER=$POSTGRESQL_USER"
else
    echo "✗ POSTGRESQL_USER не установлен"
fi

if [ -n "$POSTGRESQL_DBNAME" ]; then
    echo "✓ POSTGRESQL_DBNAME=$POSTGRESQL_DBNAME"
else
    echo "✗ POSTGRESQL_DBNAME не установлен"
fi

echo ""
echo "Проверка подключения к базе данных..."

if [ -n "$DATABASE_URL" ]; then
    PGPASSWORD=$(echo $DATABASE_URL | sed -E 's|.*://[^:]+:([^@]+)@.*|\1|') \
    psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1
elif [ -n "$POSTGRESQL_HOST" ] && [ -n "$POSTGRESQL_USER" ] && [ -n "$POSTGRESQL_DBNAME" ]; then
    PGPASSWORD="$POSTGRESQL_PASSWORD" \
    psql -h "$POSTGRESQL_HOST" -p "${POSTGRESQL_PORT:-5432}" -U "$POSTGRESQL_USER" -d "$POSTGRESQL_DBNAME" \
         -c "SELECT 1;" > /dev/null 2>&1
else
    echo "✗ Недостаточно переменных для подключения"
    exit 1
fi

if [ $? -eq 0 ]; then
    echo "✓ Подключение к базе данных успешно!"
else
    echo "✗ Ошибка подключения к базе данных"
    echo ""
    echo "Возможные причины:"
    echo "1. Неправильные учетные данные"
    echo "2. База данных недоступна с этого хоста (проверьте файрвол)"
    echo "3. База данных не разрешает подключения с вашего IP"
    exit 1
fi

