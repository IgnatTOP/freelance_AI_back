-- Добавление полей для кэширования AI анализа

-- Добавляем поля для кэширования рекомендации лучшего исполнителя в orders
ALTER TABLE orders 
ADD COLUMN IF NOT EXISTS best_recommendation_proposal_id UUID REFERENCES proposals(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS best_recommendation_justification TEXT,
ADD COLUMN IF NOT EXISTS ai_analysis_updated_at TIMESTAMPTZ;

-- Индекс для быстрого поиска заказов с устаревшим анализом
CREATE INDEX IF NOT EXISTS idx_orders_ai_analysis_updated_at ON orders(ai_analysis_updated_at) WHERE ai_analysis_updated_at IS NOT NULL;

