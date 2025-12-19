package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config содержит все конфигурационные параметры приложения
type Config struct {
	BotToken   string
	SQLitePath string
}

// Load загружает конфигурацию из переменных окружения и .env файла
func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		BotToken:   getEnv("BOT_TOKEN", ""),
		SQLitePath: getEnv("SQLITE_PATH", "./database/quiz.db"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не установлен. Проверьте .env файл")
	}

	// Создаем директорию для базы данных
	dir := filepath.Dir(cfg.SQLitePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию для БД: %v", err)
	}

	return cfg, nil
}

// GetSQLitePath возвращает путь к файлу SQLite
func (c *Config) GetSQLitePath() string {
	return c.SQLitePath
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
