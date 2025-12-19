package bot

import (
	"NetMentor_bot/database"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

type Bot struct {
	api              *tgbotapi.BotAPI
	db               *database.DB
	currentQuestions map[int64]*database.Question
	botUsername      string
}

func New(token string, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Бот %s запущен", api.Self.UserName)

	return &Bot{
		api:              api,
		db:               db,
		currentQuestions: make(map[int64]*database.Question),
		botUsername:      api.Self.UserName,
	}, nil
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		if question, exists := b.currentQuestions[chatID]; exists {
			b.checkAnswer(chatID, text, question)
			delete(b.currentQuestions, chatID)
			continue
		}

		if !b.isMessageForBot(update.Message) {
			continue
		}

		log.Printf("[%d] Команда: %s", chatID, text)

		command := b.extractCommand(text)
		switch command {
		case "start":
			b.sendMessage(chatID, "Привет! Отправь /quiz чтобы начать викторину")
		case "quiz":
			b.sendQuestion(chatID)
		case "help":
			b.sendMessage(chatID, "Команды:\n/quiz - начать викторину\n/help - помощь")
		default:
		}
	}

	return nil
}

func (b *Bot) isMessageForBot(msg *tgbotapi.Message) bool {
	if msg.Chat.Type == "private" {
		return true
	}

	if !strings.HasPrefix(msg.Text, "/") {
		return false
	}

	parts := strings.Fields(msg.Text)
	if len(parts) == 0 {
		return false
	}

	commandParts := strings.Split(parts[0], "@")
	if len(commandParts) == 1 {
		return false
	}

	return strings.ToLower(commandParts[1]) == strings.ToLower(b.botUsername)
}

func (b *Bot) extractCommand(text string) string {
	if !strings.HasPrefix(text, "/") {
		return ""
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return ""
	}

	command := strings.TrimPrefix(parts[0], "/")
	commandParts := strings.Split(command, "@")

	return strings.ToLower(commandParts[0])
}

func (b *Bot) sendQuestion(chatID int64) {
	question, err := b.db.GetRandomQuestion()
	if err != nil {
		b.sendMessage(chatID, "Ошибка: "+err.Error())
		return
	}

	b.currentQuestions[chatID] = question

	var options strings.Builder
	for i, opt := range question.Options {
		options.WriteString(fmt.Sprintf("%d) %s\n", i+1, opt))
	}

	message := fmt.Sprintf("📚 Категория: %s\n\n❓ Вопрос:\n%s\n\n%s\n*Отправьте номер ответа (1, 2, 3 или 4):*",
		question.Category,
		question.QuestionText,
		options.String())

	b.sendMessage(chatID, message)
}

func (b *Bot) checkAnswer(chatID int64, answer string, question *database.Question) {
	answer = strings.TrimSpace(answer)
	answerNum, err := strconv.Atoi(answer)

	var resultText string

	if err != nil || answerNum < 1 || answerNum > 4 {
		selectedOption := -1
		for i, option := range question.Options {
			if strings.EqualFold(strings.TrimSpace(answer), strings.TrimSpace(option)) {
				selectedOption = i
				break
			}
		}

		if selectedOption >= 0 {
			answerNum = selectedOption + 1
			err = nil
		}
	}

	if err != nil || answerNum < 1 || answerNum > 4 {
		resultText = "⚠️ Пожалуйста, отправьте номер от 1 до 4.\n\nПопробуйте еще раз: /quiz"
		b.sendMessage(chatID, resultText)
		return
	}

	userChoice := answerNum - 1
	correctIndex := question.CorrectIndex

	if userChoice == correctIndex {
		resultText = fmt.Sprintf("✅ *Правильно!*\n\nОтвет: %d. %s\n\nОтличная работа! 🎉",
			correctIndex+1, question.Options[correctIndex])
	} else {
		resultText = fmt.Sprintf("❌ *Неправильно.*\n\nВаш ответ: %d. %s\n\nПравильный ответ: %d. %s\n\nПопробуйте еще раз! 💪",
			userChoice+1, question.Options[userChoice],
			correctIndex+1, question.Options[correctIndex])
	}

	resultText += "\n\nХотите еще вопрос? Отправьте /quiz"

	b.sendMessage(chatID, resultText)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки: %v", err)
	}
}

func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
}
