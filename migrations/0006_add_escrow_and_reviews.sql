-- Миграция для добавления системы защищённой оплаты (escrow) и улучшения отзывов

-- Расширение профиля пользователя
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS phone TEXT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS telegram TEXT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS website TEXT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS company_name TEXT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS inn TEXT;

-- Баланс пользователя (в рублях)
CREATE TABLE IF NOT EXISTS user_balances (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    available       NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (available >= 0),
    frozen          NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (frozen >= 0),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Транзакции
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'transaction_type') THEN
        CREATE TYPE transaction_type AS ENUM ('deposit', 'withdrawal', 'escrow_hold', 'escrow_release', 'escrow_refund');
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'transaction_status') THEN
        CREATE TYPE transaction_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS transactions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id        UUID REFERENCES orders(id) ON DELETE SET NULL,
    type            transaction_type NOT NULL,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    status          transaction_status NOT NULL DEFAULT 'pending',
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_order_id ON transactions(order_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);

-- Escrow (защищённая сделка)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'escrow_status') THEN
        CREATE TYPE escrow_status AS ENUM ('held', 'released', 'refunded', 'disputed');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS escrow (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    client_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    freelancer_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    status          escrow_status NOT NULL DEFAULT 'held',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at     TIMESTAMPTZ,
    UNIQUE (order_id)
);

CREATE INDEX IF NOT EXISTS idx_escrow_client_id ON escrow(client_id);
CREATE INDEX IF NOT EXISTS idx_escrow_freelancer_id ON escrow(freelancer_id);
CREATE INDEX IF NOT EXISTS idx_escrow_status ON escrow(status);

-- Добавляем поле final_amount в orders для фиксации итоговой суммы сделки
ALTER TABLE orders ADD COLUMN IF NOT EXISTS final_amount NUMERIC(12,2);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS freelancer_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- Комментарии
COMMENT ON TABLE user_balances IS 'Баланс пользователя в рублях';
COMMENT ON COLUMN user_balances.available IS 'Доступные средства';
COMMENT ON COLUMN user_balances.frozen IS 'Замороженные средства (в escrow)';
COMMENT ON TABLE escrow IS 'Защищённые сделки - средства замораживаются до завершения заказа';
COMMENT ON TABLE transactions IS 'История всех финансовых операций';
