-- Индексы для улучшения производительности запросов

-- Индексы для таблицы orders
CREATE INDEX IF NOT EXISTS idx_orders_client_id ON orders(client_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_budget_min ON orders(budget_min) WHERE budget_min IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_orders_budget_max ON orders(budget_max) WHERE budget_max IS NOT NULL;

-- Индексы для таблицы proposals
CREATE INDEX IF NOT EXISTS idx_proposals_freelancer_id ON proposals(freelancer_id);
CREATE INDEX IF NOT EXISTS idx_proposals_order_id ON proposals(order_id);
CREATE INDEX IF NOT EXISTS idx_proposals_status ON proposals(status);
CREATE INDEX IF NOT EXISTS idx_proposals_created_at ON proposals(created_at DESC);

-- Индексы для таблицы messages
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);

-- Индексы для таблицы notifications
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_read ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

-- Индексы для таблицы conversations
CREATE INDEX IF NOT EXISTS idx_conversations_order_id ON conversations(order_id);
CREATE INDEX IF NOT EXISTS idx_conversations_client_id ON conversations(client_id);
CREATE INDEX IF NOT EXISTS idx_conversations_freelancer_id ON conversations(freelancer_id);
CREATE INDEX IF NOT EXISTS idx_conversations_participants ON conversations(client_id, freelancer_id) WHERE order_id IS NOT NULL;

-- Индексы для таблицы user_sessions
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Индексы для таблицы order_requirements
CREATE INDEX IF NOT EXISTS idx_order_requirements_order_id ON order_requirements(order_id);
CREATE INDEX IF NOT EXISTS idx_order_requirements_skill ON order_requirements(skill);

-- Индексы для таблицы order_attachments
CREATE INDEX IF NOT EXISTS idx_order_attachments_order_id ON order_attachments(order_id);
CREATE INDEX IF NOT EXISTS idx_order_attachments_media_id ON order_attachments(media_id);

-- Индексы для таблицы media_files
CREATE INDEX IF NOT EXISTS idx_media_files_user_id ON media_files(user_id);

-- Индексы для таблицы portfolio_items
CREATE INDEX IF NOT EXISTS idx_portfolio_items_user_id ON portfolio_items(user_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_items_created_at ON portfolio_items(created_at DESC);

