-- Миграция для добавления функциональности сообщений: ответы, вложения, реакции

-- Добавляем поля для ответов и отслеживания редактирования
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS parent_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;

-- Обновляем updated_at для существующих сообщений
UPDATE messages SET updated_at = created_at WHERE updated_at IS NULL;

-- Делаем updated_at обязательным
ALTER TABLE messages
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT NOW();

-- Таблица вложений к сообщениям
CREATE TABLE IF NOT EXISTS message_attachments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id      UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    media_id        UUID NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (message_id, media_id)
);

-- Таблица реакций на сообщения
CREATE TABLE IF NOT EXISTS message_reactions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id      UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji           TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Один пользователь может поставить только одну реакцию на сообщение
    UNIQUE (message_id, user_id)
);

-- Индексы для производительности
CREATE INDEX IF NOT EXISTS idx_messages_parent_message_id ON messages(parent_message_id);
CREATE INDEX IF NOT EXISTS idx_messages_updated_at ON messages(updated_at);
CREATE INDEX IF NOT EXISTS idx_message_attachments_message_id ON message_attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_message_attachments_media_id ON message_attachments(media_id);
CREATE INDEX IF NOT EXISTS idx_message_reactions_message_id ON message_reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_message_reactions_user_id ON message_reactions(user_id);

-- Триггер для обновления updated_at при редактировании сообщения
DROP TRIGGER IF EXISTS messages_set_updated_at ON messages;
CREATE TRIGGER messages_set_updated_at
BEFORE UPDATE ON messages
FOR EACH ROW
WHEN (OLD.content IS DISTINCT FROM NEW.content)
EXECUTE FUNCTION set_updated_at();

-- Комментарии к таблицам
COMMENT ON COLUMN messages.parent_message_id IS 'ID сообщения, на которое данное сообщение является ответом';
COMMENT ON COLUMN messages.updated_at IS 'Время последнего редактирования сообщения';
COMMENT ON TABLE message_attachments IS 'Вложения (файлы) к сообщениям';
COMMENT ON TABLE message_reactions IS 'Реакции (эмодзи) на сообщения';




