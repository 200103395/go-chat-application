package main

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"os"
	"strings"
)

type Storage struct {
	DB *sql.DB
}

var (
	dbuser string
	dbpass string
)

func NewPostgresStorage() (*Storage, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	dbuser = os.Getenv("DB_USERNAME")
	dbpass = os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	connect := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", dbuser, dbpass, dbname)
	DB, err := sql.Open("postgres", connect)
	if err != nil {
		return nil, err
	}

	return &Storage{
		DB: DB,
	}, nil
}

func (s *Storage) Init() error {
	if err := s.DB.Ping(); err != nil {
		fmt.Println("Ping error", err)
		return err
	}
	if err := s.CreateTables(); err != nil {
		return err
	}
	return nil
}

func (s *Storage) CreateUser(acc Account) error {
	query := `insert into account (username, password) values ($1, $2);`
	_, err := s.DB.Exec(query, acc.Username, acc.Password)
	return err
}

func (s *Storage) GetAllUsers() (*[]Account, error) {
	var accounts []Account
	query := `select * from account;`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var acc Account
		err = rows.Scan(&acc.Username, &acc.Password)
		if err != nil {
			continue
		}
		accounts = append(accounts, acc)
	}
	return &accounts, nil
}

func (s *Storage) GetUserByUsername(username string) (*Account, error) {
	var acc Account
	query := `select * from account where username = $1 limit 1;`
	err := s.DB.QueryRow(query, username).Scan(&acc.Username, &acc.Password)
	return &acc, err
}

func (s *Storage) CheckUserExistence(username string) error {
	curError := fmt.Errorf("user not exists")
	query := `select count(*) from account where username = $1`
	var count int
	if err := s.DB.QueryRow(query, username).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return curError
	}
	return nil
}

func (s *Storage) CreateMessage(m Message) error {
	query := `insert into message (fromAcc, toAcc, message, time) values($1, $2, $3, $4)`
	_, err := s.DB.Exec(query, m.From, m.To, m.Message, m.Time)
	return err
}

func (s *Storage) GetMessages(from string, to string) (*[]Message, error) {
	query := `select * from message where 
                          (fromAcc = $1 and toAcc = $2) or (fromAcc = $2 and toAcc = $1)
                          order by time;`
	var messages []Message
	rows, err := s.DB.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.From, &m.To, &m.Message, &m.Time); err != nil {
			continue
		}
		messages = append(messages, m)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found")
	}
	return &messages, nil
}

func (s *Storage) GetChatNames(username string) ([]ChatName, error) {
	query := `SELECT 
		subquery.chat_name,
		subquery.max_time,
		message.message
	FROM (
		SELECT 
			CASE 
				WHEN fromAcc = $1 THEN toAcc
				ELSE fromAcc 
			END AS chat_name,
			MAX(time) AS max_time
		FROM message
		WHERE fromAcc = $1 OR toAcc = $1
		GROUP BY chat_name
	) AS subquery
	JOIN message ON (
		(subquery.chat_name = message.fromAcc AND message.toAcc = $1)
		OR
		(subquery.chat_name = message.toAcc AND message.fromAcc = $1)
	) AND subquery.max_time = message.time
	ORDER BY subquery.max_time DESC;`
	rows, err := s.DB.Query(query, username)
	if err != nil {
		return nil, err
	}
	var names []ChatName
	for rows.Next() {
		var chat ChatName
		// Maybe time show
		if err = rows.Scan(&chat.Username, &chat.Time, &chat.Message); err != nil {
			return nil, err
		}
		names = append(names, chat)
	}
	return names, nil
}

func (s *Storage) SearchUsername(username string) ([]WebAccount, error) {
	query := `select username from account where lower(username) like $1`
	username = "%" + strings.ToLower(username) + "%"

	rows, err := s.DB.Query(query, username)
	if err != nil {
		return nil, err
	}
	var names []WebAccount
	for rows.Next() {
		var username WebAccount
		if err = rows.Scan(&username.Username); err != nil {
			return nil, err
		}
		names = append(names, username)
	}
	return names, nil
}

func (s *Storage) CreateTables() error {
	fmt.Println("Creating Tables")
	query := `CREATE TABLE IF NOT EXISTS account (
    username VARCHAR(100) PRIMARY KEY,
    password VARCHAR(400) NOT NULL
);`
	if _, err := s.DB.Exec(query); err != nil {
		return err
	}

	query = `CREATE TABLE IF NOT EXISTS message (
    fromAcc VARCHAR(100) NOT NULL,
    toAcc VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    time TIMESTAMP NOT NULL
);`
	if _, err := s.DB.Exec(query); err != nil {
		return err
	}

	return nil
}
