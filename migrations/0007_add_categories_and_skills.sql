-- Миграция для добавления категорий и предустановленных навыков

-- Категории заказов
CREATE TABLE IF NOT EXISTS categories (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug            TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT,
    icon            TEXT,
    parent_id       UUID REFERENCES categories(id) ON DELETE SET NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Предустановленные навыки
CREATE TABLE IF NOT EXISTS skills (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug            TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    category_id     UUID REFERENCES categories(id) ON DELETE SET NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Связь категорий с заказами
ALTER TABLE orders ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

-- Индексы
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_skills_category_id ON skills(category_id);
CREATE INDEX IF NOT EXISTS idx_skills_slug ON skills(slug);
CREATE INDEX IF NOT EXISTS idx_orders_category_id ON orders(category_id);

-- Заполнение категорий
INSERT INTO categories (slug, name, description, icon, sort_order) VALUES
('web-development', 'Веб-разработка', 'Создание сайтов и веб-приложений', '🌐', 1),
('mobile-development', 'Мобильная разработка', 'iOS и Android приложения', '📱', 2),
('design', 'Дизайн', 'Графический и UI/UX дизайн', '🎨', 3),
('marketing', 'Маркетинг', 'SMM, SEO, реклама', '📈', 4),
('writing', 'Копирайтинг', 'Тексты, статьи, контент', '✍️', 5),
('video', 'Видео и анимация', 'Монтаж, моушн-дизайн', '🎬', 6),
('data', 'Данные и аналитика', 'Data Science, ML, аналитика', '📊', 7),
('admin', 'Администрирование', 'DevOps, системное администрирование', '⚙️', 8),
('other', 'Другое', 'Прочие услуги', '📦', 99)
ON CONFLICT (slug) DO NOTHING;

-- Подкатегории веб-разработки
INSERT INTO categories (slug, name, parent_id, sort_order)
SELECT 'frontend', 'Frontend разработка', id, 1 FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (slug, name, parent_id, sort_order)
SELECT 'backend', 'Backend разработка', id, 2 FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (slug, name, parent_id, sort_order)
SELECT 'fullstack', 'Fullstack разработка', id, 3 FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;

-- Заполнение навыков
-- Веб-разработка
INSERT INTO skills (slug, name, category_id) 
SELECT 'javascript', 'JavaScript', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'typescript', 'TypeScript', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'react', 'React', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'vue', 'Vue.js', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'angular', 'Angular', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'nodejs', 'Node.js', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'python', 'Python', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'php', 'PHP', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'go', 'Go', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'java', 'Java', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'csharp', 'C#', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'ruby', 'Ruby', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'html-css', 'HTML/CSS', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'postgresql', 'PostgreSQL', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'mongodb', 'MongoDB', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'redis', 'Redis', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'graphql', 'GraphQL', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'rest-api', 'REST API', id FROM categories WHERE slug = 'web-development'
ON CONFLICT (slug) DO NOTHING;

-- Мобильная разработка
INSERT INTO skills (slug, name, category_id) 
SELECT 'swift', 'Swift', id FROM categories WHERE slug = 'mobile-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'kotlin', 'Kotlin', id FROM categories WHERE slug = 'mobile-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'react-native', 'React Native', id FROM categories WHERE slug = 'mobile-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'flutter', 'Flutter', id FROM categories WHERE slug = 'mobile-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'ios', 'iOS', id FROM categories WHERE slug = 'mobile-development'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'android', 'Android', id FROM categories WHERE slug = 'mobile-development'
ON CONFLICT (slug) DO NOTHING;

-- Дизайн
INSERT INTO skills (slug, name, category_id) 
SELECT 'figma', 'Figma', id FROM categories WHERE slug = 'design'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'photoshop', 'Photoshop', id FROM categories WHERE slug = 'design'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'illustrator', 'Illustrator', id FROM categories WHERE slug = 'design'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'ui-ux', 'UI/UX дизайн', id FROM categories WHERE slug = 'design'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'logo-design', 'Дизайн логотипов', id FROM categories WHERE slug = 'design'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'web-design', 'Веб-дизайн', id FROM categories WHERE slug = 'design'
ON CONFLICT (slug) DO NOTHING;

-- Маркетинг
INSERT INTO skills (slug, name, category_id) 
SELECT 'seo', 'SEO', id FROM categories WHERE slug = 'marketing'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'smm', 'SMM', id FROM categories WHERE slug = 'marketing'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'context-ads', 'Контекстная реклама', id FROM categories WHERE slug = 'marketing'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'target-ads', 'Таргетированная реклама', id FROM categories WHERE slug = 'marketing'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'email-marketing', 'Email-маркетинг', id FROM categories WHERE slug = 'marketing'
ON CONFLICT (slug) DO NOTHING;

-- Data & Analytics
INSERT INTO skills (slug, name, category_id) 
SELECT 'data-analysis', 'Анализ данных', id FROM categories WHERE slug = 'data'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'machine-learning', 'Machine Learning', id FROM categories WHERE slug = 'data'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'sql', 'SQL', id FROM categories WHERE slug = 'data'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'tableau', 'Tableau', id FROM categories WHERE slug = 'data'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'power-bi', 'Power BI', id FROM categories WHERE slug = 'data'
ON CONFLICT (slug) DO NOTHING;

-- DevOps
INSERT INTO skills (slug, name, category_id) 
SELECT 'docker', 'Docker', id FROM categories WHERE slug = 'admin'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'kubernetes', 'Kubernetes', id FROM categories WHERE slug = 'admin'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'aws', 'AWS', id FROM categories WHERE slug = 'admin'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'linux', 'Linux', id FROM categories WHERE slug = 'admin'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO skills (slug, name, category_id) 
SELECT 'ci-cd', 'CI/CD', id FROM categories WHERE slug = 'admin'
ON CONFLICT (slug) DO NOTHING;

COMMENT ON TABLE categories IS 'Категории заказов';
COMMENT ON TABLE skills IS 'Предустановленные навыки для выбора';
