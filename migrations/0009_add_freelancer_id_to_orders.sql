-- Добавляем колонку freelancer_id в таблицу orders
ALTER TABLE orders ADD COLUMN IF NOT EXISTS freelancer_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- Добавляем колонку final_amount если её нет
ALTER TABLE orders ADD COLUMN IF NOT EXISTS final_amount NUMERIC(12,2);

-- Добавляем колонку category_id если её нет
ALTER TABLE orders ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

-- Индекс для быстрого поиска заказов по фрилансеру
CREATE INDEX IF NOT EXISTS idx_orders_freelancer_id ON orders(freelancer_id) WHERE freelancer_id IS NOT NULL;

-- Обновляем freelancer_id на основе принятых предложений
UPDATE orders o
SET freelancer_id = p.freelancer_id
FROM proposals p
WHERE p.order_id = o.id 
  AND p.status = 'accepted'
  AND o.freelancer_id IS NULL;
