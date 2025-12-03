-- Скрипт для полного сброса базы данных
-- ВНИМАНИЕ: Этот скрипт удалит ВСЕ данные из базы!

-- Отключаем проверку внешних ключей временно
SET session_replication_role = 'replica';

-- Удаляем все таблицы (новые фичи)
DROP TABLE IF EXISTS order_history CASCADE;
DROP TABLE IF EXISTS proposal_templates CASCADE;
DROP TABLE IF EXISTS reports CASCADE;
DROP TABLE IF EXISTS verification_codes CASCADE;
DROP TABLE IF EXISTS favorites CASCADE;
DROP TABLE IF EXISTS withdrawals CASCADE;
DROP TABLE IF EXISTS disputes CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS escrow CASCADE;
DROP TABLE IF EXISTS user_balances CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS skill_categories CASCADE;
DROP TABLE IF EXISTS skills CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS message_reactions CASCADE;
DROP TABLE IF EXISTS message_read_status CASCADE;

-- Удаляем основные таблицы
DROP TABLE IF EXISTS schema_migrations CASCADE;
DROP TABLE IF EXISTS portfolio_media CASCADE;
DROP TABLE IF EXISTS portfolio_items CASCADE;
DROP TABLE IF EXISTS proposals CASCADE;
DROP TABLE IF EXISTS order_requirements CASCADE;
DROP TABLE IF EXISTS order_attachments CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS ai_cache CASCADE;
DROP TABLE IF EXISTS profiles CASCADE;
DROP TABLE IF EXISTS user_sessions CASCADE;
DROP TABLE IF EXISTS media_files CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Удаляем пользовательские типы
DROP TYPE IF EXISTS order_status CASCADE;
DROP TYPE IF EXISTS proposal_status CASCADE;
DROP TYPE IF EXISTS author_type CASCADE;
DROP TYPE IF EXISTS escrow_status CASCADE;
DROP TYPE IF EXISTS transaction_type CASCADE;
DROP TYPE IF EXISTS transaction_status CASCADE;
DROP TYPE IF EXISTS dispute_status CASCADE;
DROP TYPE IF EXISTS withdrawal_status CASCADE;
DROP TYPE IF EXISTS report_status CASCADE;

-- Включаем обратно проверку внешних ключей
SET session_replication_role = 'origin';

-- Примечание: расширения не удаляем, они нужны для миграций
-- uuid-ossp, pgcrypto, citext
