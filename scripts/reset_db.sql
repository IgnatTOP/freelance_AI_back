-- Скрипт для полного сброса базы данных
-- ВНИМАНИЕ: Этот скрипт удалит ВСЕ данные из базы!

-- Отключаем проверку внешних ключей временно
SET session_replication_role = 'replica';

-- Удаляем все таблицы в правильном порядке (с учетом зависимостей)
DROP TABLE IF EXISTS schema_migrations CASCADE;
DROP TABLE IF EXISTS portfolio_media CASCADE;
DROP TABLE IF EXISTS portfolio_items CASCADE;
DROP TABLE IF EXISTS proposals CASCADE;
DROP TABLE IF EXISTS order_requirements CASCADE;
DROP TABLE IF EXISTS order_attachments CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS ai_cache CASCADE;
DROP TABLE IF EXISTS profiles CASCADE;
DROP TABLE IF EXISTS user_sessions CASCADE;
DROP TABLE IF EXISTS media_files CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Удаляем пользовательские типы
DROP TYPE IF EXISTS order_status CASCADE;
DROP TYPE IF EXISTS proposal_status CASCADE;
DROP TYPE IF EXISTS author_type CASCADE;

-- Включаем обратно проверку внешних ключей
SET session_replication_role = 'origin';

-- Удаляем расширения (опционально, если они больше не нужны)
-- DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
-- DROP EXTENSION IF EXISTS "pgcrypto" CASCADE;
-- DROP EXTENSION IF EXISTS "citext" CASCADE;

