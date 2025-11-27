-- Миграция для добавления системы отзывов и рейтингов

-- Таблица отзывов
CREATE TABLE IF NOT EXISTS reviews (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    reviewer_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewed_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating          INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Один отзыв на один заказ от одного рецензента
    UNIQUE (order_id, reviewer_id)
);

-- Индексы для reviews
CREATE INDEX IF NOT EXISTS idx_reviews_reviewed_id ON reviews(reviewed_id);
CREATE INDEX IF NOT EXISTS idx_reviews_reviewer_id ON reviews(reviewer_id);
CREATE INDEX IF NOT EXISTS idx_reviews_order_id ON reviews(order_id);
CREATE INDEX IF NOT EXISTS idx_reviews_rating ON reviews(rating);
CREATE INDEX IF NOT EXISTS idx_reviews_created_at ON reviews(created_at DESC);

-- Триггер для обновления updated_at
DROP TRIGGER IF EXISTS reviews_set_updated_at ON reviews;
CREATE TRIGGER reviews_set_updated_at
BEFORE UPDATE ON reviews
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Комментарии к таблицам
COMMENT ON TABLE reviews IS 'Отзывы пользователей друг о друге после завершения заказа';
COMMENT ON COLUMN reviews.reviewer_id IS 'ID пользователя, который оставляет отзыв';
COMMENT ON COLUMN reviews.reviewed_id IS 'ID пользователя, о котором оставляют отзыв';
COMMENT ON COLUMN reviews.rating IS 'Оценка от 1 до 5';
COMMENT ON COLUMN reviews.comment IS 'Текстовый комментарий к отзыву';

