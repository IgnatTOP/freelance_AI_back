package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config хранит все параметры запуска приложения.
type Config struct {
	Env              string
	HTTPPort         string
	DatabaseURL      string
	JWTSecret        string
	RefreshSecret    string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	MediaStoragePath string
	AIBaseURL        string
	AIModel          string
	MaxUploadSizeMB  int64
	MigrationsPath   string
	AllowedOrigins   []string
	RateLimitLimit   int64
	RateLimitPeriod  time.Duration
}

// Load читает переменные окружения и возвращает готовую конфигурацию.
func Load() (*Config, error) {
	// Загружаем .env только если он существует, иначе используем системные переменные.
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("config: .env не найден, используем переменные окружения: %v", err)
	}

	env := getEnv("APP_ENV", "development")

	// Получаем DatabaseURL - либо напрямую, либо собираем из отдельных переменных
	databaseURL := getDatabaseURL()

	cfg := &Config{
		Env:              env,
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DatabaseURL:      databaseURL,
		MediaStoragePath: getEnv("MEDIA_STORAGE_PATH", "./storage/media"),
		AIBaseURL:        getEnv("AI_BASE_URL", "http://localhost:9000"),
		AIModel:          getEnv("AI_MODEL", "gpt-4o-mini"),
		MigrationsPath:   getEnv("MIGRATIONS_PATH", "./migrations"),
	}

	// Валидация JWT секретов
	jwtSecret := getEnv("JWT_SECRET", "")
	refreshSecret := getEnv("REFRESH_SECRET", "")

	if env == "production" {
		if jwtSecret == "" || len(jwtSecret) < 32 {
			return nil, fmt.Errorf("config: JWT_SECRET обязателен и должен быть не менее 32 символов в production")
		}
		if refreshSecret == "" || len(refreshSecret) < 32 {
			return nil, fmt.Errorf("config: REFRESH_SECRET обязателен и должен быть не менее 32 символов в production")
		}
	} else {
		// В development используем дефолтные значения, но предупреждаем
		if jwtSecret == "" {
			jwtSecret = "super-secret-development-only-change-in-production"
			log.Printf("config: WARNING - используется дефолтный JWT_SECRET, измените в production!")
		}
		if refreshSecret == "" {
			refreshSecret = "super-refresh-secret-development-only-change-in-production"
			log.Printf("config: WARNING - используется дефолтный REFRESH_SECRET, измените в production!")
		}
	}

	cfg.JWTSecret = jwtSecret
	cfg.RefreshSecret = refreshSecret

	// CORS allowed origins
	originsStr := getEnv("CORS_ALLOWED_ORIGINS", "")
	if originsStr == "" {
		// Дефолтные значения для development
		if env == "production" {
			return nil, fmt.Errorf("config: CORS_ALLOWED_ORIGINS обязателен в production")
		}
		cfg.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	} else {
		cfg.AllowedOrigins = strings.Split(originsStr, ",")
		// Убираем пробелы
		for i, origin := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(origin)
		}
	}

	cfg.AccessTokenTTL = mustParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"))
	cfg.RefreshTokenTTL = mustParseDuration(getEnv("REFRESH_TOKEN_TTL", "720h"))
	cfg.MaxUploadSizeMB = mustParseInt64(getEnv("MAX_UPLOAD_MB", "10"))

	// Rate limiting настройки
	cfg.RateLimitLimit = mustParseInt64(getEnv("RATE_LIMIT_LIMIT", "10"))
	rateLimitPeriodStr := getEnv("RATE_LIMIT_PERIOD", "1m")
	cfg.RateLimitPeriod = mustParseDuration(rateLimitPeriodStr)

	return cfg, nil
}

// getEnv возвращает значение переменной окружения или дефолт.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// getDatabaseURL возвращает DATABASE_URL либо из переменной, либо собирает из отдельных переменных.
func getDatabaseURL() string {
	// Если DATABASE_URL задан напрямую, используем его
	if dbURL := getEnv("DATABASE_URL", ""); dbURL != "" {
		return dbURL
	}

	// Иначе собираем из отдельных переменных (формат платформы)
	host := getEnv("POSTGRESQL_HOST", "")
	port := getEnv("POSTGRESQL_PORT", "5432")
	user := getEnv("POSTGRESQL_USER", "")
	password := getEnv("POSTGRESQL_PASSWORD", "")
	dbname := getEnv("POSTGRESQL_DBNAME", "")

	// Если все переменные заданы, собираем URL
	if host != "" && user != "" && dbname != "" {
		// URL-кодируем пароль и имя пользователя для безопасности
		// Используем url.UserPassword для правильного кодирования
		userInfo := url.UserPassword(user, password)
		
		dbURL := fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable",
			userInfo.String(), host, port, dbname)
		return dbURL
	}

	// Если ничего не задано, возвращаем дефолт
	return "postgres://postgres:123@localhost:5432/freelance_ai?sslmode=disable"
}

// mustParseDuration безопасно парсит строку в duration.
func mustParseDuration(v string) time.Duration {
	dur, err := time.ParseDuration(v)
	if err != nil {
		log.Fatalf("config: не удалось распарсить длительность %q: %v", v, err)
	}
	return dur
}

// mustParseInt64 безопасно парсит строку в int64.
func mustParseInt64(v string) int64 {
	num, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log.Fatalf("config: не удалось распарсить число %q: %v", v, err)
	}
	return num
}
