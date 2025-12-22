package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BotToken   string
	SQLitePath string
}

func Load() (*Config, error) {
	// Читаем .env файл вручную
	loadEnv()

	cfg := &Config{
		BotToken:   getEnv("BOT_TOKEN", ""),
		SQLitePath: getEnv("SQLITE_PATH", "./database/quiz.db"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не установлен")
	}

	dir := filepath.Dir(cfg.SQLitePath)
	os.MkdirAll(dir, 0755)

	return cfg, nil
}

func (c *Config) GetSQLitePath() string {
	return c.SQLitePath
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return // Файла нет, используем системные переменные
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}
