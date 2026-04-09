package organizer

import "os"

// Config хранит параметры запуска органайзера
type Config struct {
	EnableCrossLinks bool // Включить авто-связи между модулями
	AIDrivenRules    bool // Переключить правила связей на ИИ (stub)
}

// LoadConfig считывает настройки окружения
func LoadConfig() Config {
	return Config{
		EnableCrossLinks: os.Getenv("ORG_CROSS_LINKS") != "false",
		AIDrivenRules:    os.Getenv("ORG_AI_RULES") == "true",
	}
}
