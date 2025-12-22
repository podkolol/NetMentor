package bot

import (
	"NetMentor_bot/database"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Bot struct {
	token            string
	baseURL          string
	db               *database.DB
	currentQuestions map[int64]*database.Question
	lastUpdateID     int
	botUsername      string
}

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	From      User   `json:"from"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func New(token string, db *database.DB) (*Bot, error) {
	bot := &Bot{
		token:            token,
		baseURL:          "https://api.telegram.org/bot" + token + "/",
		db:               db,
		currentQuestions: make(map[int64]*database.Question),
		lastUpdateID:     0,
	}

	// Получаем информацию о боте
	info, err := bot.getMe()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить информацию о боте: %v", err)
	}
	bot.botUsername = info.Username

	log.Printf("Бот @%s запущен", bot.botUsername)
	return bot, nil
}

type BotInfo struct {
	Username string `json:"username"`
}

func (b *Bot) getMe() (*BotInfo, error) {
	resp, err := http.Get(b.baseURL + "getMe")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool    `json:"ok"`
		Result BotInfo `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("ошибка API: %s", string(body))
	}

	return &result.Result, nil
}

func (b *Bot) Start() error {
	log.Println("Запуск получения обновлений...")

	for {
		updates, err := b.getUpdates()
		if err != nil {
			log.Printf("Ошибка получения обновлений: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			b.processUpdate(update)
			b.lastUpdateID = update.UpdateID
		}

		time.Sleep(1 * time.Second)
	}
}

func (b *Bot) getUpdates() ([]Update, error) {
	url := fmt.Sprintf("%sgetUpdates?offset=%d&timeout=30", b.baseURL, b.lastUpdateID+1)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("ошибка API: %s", string(body))
	}

	return result.Result, nil
}

func (b *Bot) processUpdate(update Update) {
	if update.Message.Text == "" {
		return
	}

	chatID := update.Message.Chat.ID
	text := update.Message.Text

	if question, exists := b.currentQuestions[chatID]; exists {
		b.checkAnswer(chatID, text, question)
		delete(b.currentQuestions, chatID)
		return
	}

	if !b.isMessageForBot(&update.Message) {
		return
	}

	log.Printf("[%d] Команда: %s", chatID, text)

	command := b.extractCommand(text)
	switch command {
	case "start":
		b.sendMessage(chatID, "Отправь /quiz чтобы начать викторину")
	case "quiz":
		b.sendQuestion(chatID)
	case "help":
		b.sendMessage(chatID, "Команды:\n/quiz - начать викторину\n/help - помощь")
	default:
	}
}

func (b *Bot) isMessageForBot(msg *Message) bool {
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
		resultText = "Пожалуйста, отправьте номер от 1 до 4.\n\nПопробуйте еще раз: /quiz"
		b.sendMessage(chatID, resultText)
		return
	}

	userChoice := answerNum - 1
	correctIndex := question.CorrectIndex

	if userChoice == correctIndex {
		resultText = fmt.Sprintf("✅ *Правильно!*\n\nОтвет: %d. %s",
			correctIndex+1, question.Options[correctIndex])
	} else {
		resultText = fmt.Sprintf("❌ *Неправильно.*\n\nВаш ответ: %d. %s\n\nПравильный ответ: %d. %s\n\nПопробуйте еще раз.",
			userChoice+1, question.Options[userChoice],
			correctIndex+1, question.Options[correctIndex])
	}

	resultText += "\n\nХотите еще вопрос? Отправьте /quiz"
	b.sendMessage(chatID, resultText)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	params.Set("text", text)
	params.Set("parse_mode", "Markdown")

	url := b.baseURL + "sendMessage?" + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка чтения ответа: %v", err)
		return
	}

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.OK {
		log.Printf("Ошибка API при отправке: %s", string(body))
	}
}

func (b *Bot) Stop() {
	log.Println("Бот остановлен")
}
