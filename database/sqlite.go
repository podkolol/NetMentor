package database

import (
	"NetMentor_bot/config"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	_ "modernc.org/sqlite"
)

type Question struct {
	ID           int      `json:"id"`
	QuestionText string   `json:"question_text"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correct_index"`
	Category     string   `json:"category"`
}

type DB struct {
	conn *sql.DB
}

func New(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("sqlite", cfg.GetSQLitePath())
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка ping: %v", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	log.Println("✅ Подключено к SQLite")
	return &DB{conn: db}, nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *DB) CreateTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS questions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		question_text TEXT NOT NULL,
		options TEXT NOT NULL,
		correct_index INTEGER NOT NULL,
		category TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	log.Println("✅ Таблица создана")
	return nil
}

func (db *DB) InitSampleData() error {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки данных: %v", err)
	}

	if count > 0 {
		log.Printf("✅ В базе уже есть %d вопросов\n", count)
		return nil
	}

	questions := []Question{
		{
			QuestionText: "Что такое IP-адрес?",
			Options: []string{
				"Уникальный идентификатор устройства в сети",
				"Протокол передачи данных",
				"Тип кабеля",
				"Сетевое приложение",
			},
			CorrectIndex: 0,
			Category:     "Основы",
		},
		{
			QuestionText: "Какой порт у HTTP?",
			Options:      []string{"80", "443", "21", "25"},
			CorrectIndex: 0,
			Category:     "Протоколы",
		},
		{
			QuestionText: "Что такое DNS?",
			Options: []string{
				"Система доменных имен",
				"Сетевой протокол",
				"Тип сервера",
				"Язык программирования",
			},
			CorrectIndex: 0,
			Category:     "Протоколы",
		},
		{
			QuestionText: "Какой протокол устанавливает соединение?",
			Options:      []string{"TCP", "UDP", "HTTP", "ICMP"},
			CorrectIndex: 0,
			Category:     "Протоколы",
		},
		{
			QuestionText: "Что такое MAC-адрес?",
			Options: []string{
				"Физический адрес сетевой карты",
				"IP-адрес маршрутизатора",
				"Доменное имя",
				"Пароль Wi-Fi",
			},
			CorrectIndex: 0,
			Category:     "Оборудование",
		},
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO questions (question_text, options, correct_index, category)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	for _, q := range questions {
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			return fmt.Errorf("ошибка маршалинга JSON: %v", err)
		}

		_, err = stmt.Exec(q.QuestionText, optionsJSON, q.CorrectIndex, q.Category)
		if err != nil {
			return fmt.Errorf("ошибка добавления вопроса: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка коммита: %v", err)
	}

	log.Printf("✅ Добавлено %d вопросов\n", len(questions))
	return nil
}

func (db *DB) AddQuestion(q Question) error {
	optionsJSON, err := json.Marshal(q.Options)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO questions (question_text, options, correct_index, category)
	VALUES (?, ?, ?, ?)
	`

	_, err = db.conn.Exec(query, q.QuestionText, optionsJSON, q.CorrectIndex, q.Category)
	return err
}

func (db *DB) GetRandomQuestion() (*Question, error) {
	var q Question
	var optionsJSON string

	query := `
	SELECT id, question_text, options, correct_index, category
	FROM questions
	ORDER BY RANDOM()
	LIMIT 1
	`

	err := db.conn.QueryRow(query).Scan(
		&q.ID,
		&q.QuestionText,
		&optionsJSON,
		&q.CorrectIndex,
		&q.Category,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(optionsJSON), &q.Options)
	return &q, nil
}
