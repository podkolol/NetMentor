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

	log.Println("Подключено к существующей БД SQLite")
	return &DB{conn: db}, nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
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
		return nil, fmt.Errorf("ошибка получения вопроса: %v", err)
	}

	if err := json.Unmarshal([]byte(optionsJSON), &q.Options); err != nil {
		return nil, fmt.Errorf("ошибка разбора JSON опций: %v", err)
	}

	return &q, nil
}
