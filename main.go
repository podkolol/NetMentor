package main

import (
	"log"

	"NetMentor_bot/bot"
	"NetMentor_bot/config"
	"NetMentor_bot/database"
)

func main() {
	log.Println("🚀 Запуск бота...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Ошибка конфигурации:", err)
	}

	db, err := database.New(cfg)
	if err != nil {
		log.Fatal("Ошибка БД:", err)
	}
	defer db.Close()

	bot, err := bot.New(cfg.BotToken, db)
	if err != nil {
		log.Fatal("Ошибка бота:", err)
	}

	log.Println("✅ Бот запущен")
	bot.Start()
}
