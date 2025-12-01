#!/bin/bash

# Скрипт для установки и настройки PostgreSQL на Arch Linux

set -e

echo "=== Настройка PostgreSQL ==="

# Проверяем, инициализирована ли база данных
if [ ! -d "/var/lib/postgres/data" ] || [ ! -f "/var/lib/postgres/data/PG_VERSION" ]; then
    echo "Инициализация базы данных PostgreSQL..."
    # Создаем директорию, если она не существует
    sudo mkdir -p /var/lib/postgres/data
    sudo chown postgres:postgres /var/lib/postgres/data
    sudo chmod 700 /var/lib/postgres/data
    # Инициализируем базу данных
    sudo -u postgres initdb --locale=C.UTF-8 --encoding=UTF8 -D /var/lib/postgres/data
    echo "✓ База данных инициализирована"
else
    echo "✓ База данных уже инициализирована"
fi

# Запускаем службу PostgreSQL
echo "Запуск службы PostgreSQL..."
sudo systemctl enable postgresql.service
sudo systemctl start postgresql.service

# Ждем немного, чтобы PostgreSQL запустился
sleep 2

# Проверяем статус
if systemctl is-active --quiet postgresql; then
    echo "✓ PostgreSQL успешно запущен"
else
    echo "✗ Ошибка: PostgreSQL не запустился"
    exit 1
fi

# Создаем базу данных и пользователя
DB_NAME="freelance_ai"
DB_USER="postgres"
DB_PASSWORD="123"

echo "Создание базы данных '$DB_NAME' и настройка пользователя..."

# Создаем базу данных
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || echo "База данных уже существует"

# Устанавливаем пароль для пользователя postgres
sudo -u postgres psql -c "ALTER USER postgres PASSWORD '$DB_PASSWORD';"

echo "✓ База данных и пользователь настроены"
echo ""
echo "Параметры подключения:"
echo "  Host: localhost"
echo "  Port: 5432"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo "  Password: $DB_PASSWORD"
echo ""
echo "Для подключения используйте:"
echo "  psql -h localhost -U $DB_USER -d $DB_NAME"

