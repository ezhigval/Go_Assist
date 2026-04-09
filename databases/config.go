package databases

import "os"

// Config хранит параметры подключения к PostgreSQL
type Config struct {
	Host     string
	Port     string
	Name     string
	User     string
	Pass     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// LoadConfig загружает настройки из переменных окружения с безопасными дефолтами
func LoadConfig() Config {
	return Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		Name:     getEnv("DB_NAME", "telegram_bot"),
		User:     getEnv("DB_USER", "postgres"),
		Pass:     getEnv("DB_PASS", ""),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		MaxConns: 20,
		MinConns: 5,
	}
}

// DSN формирует строку подключения для pgx
func (c Config) DSN() string {
	return "postgres://" + c.User + ":" + c.Pass + "@" + c.Host + ":" + c.Port + "/" + c.Name + "?sslmode=" + c.SSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
