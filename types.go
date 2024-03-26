package main

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Password2 string `json:"password2"`
}

type WebAccount struct {
	Username string `json:"username"`
}

type ChatName struct {
	Username string    `json:"username"`
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
}

type Message struct {
	From    string    `json:"from"`
	To      string    `json:"to"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

func (acc *Account) ToWebAccount() *WebAccount {
	return &WebAccount{
		Username: acc.Username,
	}
}

func (acc *Account) Valid() bool {
	return true
}

func (req *RegisterRequest) Valid() bool {
	return true
}

func (acc *Account) ValidPassword(pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(pw)) == nil
}

func (acc *Account) GenerateHashedPassword() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(acc.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	acc.Password = string(hash)
	return nil
}
