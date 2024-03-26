package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"time"
)

type ChatServer struct {
	address string
	storage Storage
}

var (
	authorizationDuration time.Duration = 120
)

func NewChatServer(address string, storage Storage) *ChatServer {
	return &ChatServer{
		address: address,
		storage: storage,
	}
}

func (ch *ChatServer) Run() error {
	r := mux.NewRouter()
	r.HandleFunc("/login", MakeHTTPHandler(ch.LoginHandler))
	r.HandleFunc("/register", MakeHTTPHandler(ch.RegisterHandler))
	r.HandleFunc("/chat/{username}", withJWTAuth(MakeHTTPHandler(ch.ChatUserHandler), ch.storage))
	r.HandleFunc("/chat", withJWTAuth(MakeHTTPHandler(ch.ChatPageHandler), ch.storage))

	r.HandleFunc("/unAuthorize", MakeHTTPHandler(ch.UnAuthorizeHandler))
	r.HandleFunc("/getAccountInfo", MakeHTTPHandler(ch.GetAccountInfoHandler))
	r.HandleFunc("/getAllAccounts", MakeHTTPHandler(ch.GetAllAccountsHandler))
	r.HandleFunc("/getMessages/{username}", MakeHTTPHandler(ch.GetInitialMessagesHandler))
	r.HandleFunc("/getChatNames", MakeHTTPHandler(ch.GetChatNamesHandler))
	r.HandleFunc("/search", MakeHTTPHandler(ch.SearchHandler))

	r.HandleFunc("/", MakeHTTPHandler(ch.MainHandler))
	return http.ListenAndServe(ch.address, r)
}

func (ch *ChatServer) MainHandler(w http.ResponseWriter, r *http.Request) error {
	html, err := os.ReadFile("static/index.html")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(html))
	return err
}

func (ch *ChatServer) UnAuthorizeHandler(w http.ResponseWriter, r *http.Request) error {
	DeleteJWT(w)
	html, err := os.ReadFile("static/redirectIndex.html")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(html))
	return err
}

func (ch *ChatServer) GetAllAccountsHandler(w http.ResponseWriter, r *http.Request) error {
	accounts, err := ch.storage.GetAllUsers()
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, accounts)
}

func (ch *ChatServer) GetAccountInfoHandler(w http.ResponseWriter, r *http.Request) error {
	account, err := ReadJWT(r, ch.storage)
	if err != nil {
		return err
	}
	webAccount := account.ToWebAccount()
	return WriteJSON(w, http.StatusOK, webAccount)
}

func (ch *ChatServer) ChatPageHandler(w http.ResponseWriter, r *http.Request) error {
	html, err := os.ReadFile("static/chat.html")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(html))
	return err
}

func (ch *ChatServer) SearchHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed")
	}
	var input struct {
		Input string `json:"input"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		return err
	}
	fmt.Println(input)
	names, err := ch.storage.SearchUsername(input.Input)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, names)
}

func (ch *ChatServer) ChatUserHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		html, err := os.ReadFile("static/chatAccount.html")
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(html))
		return err
	}
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed")
	}
	var message struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		return err
	}
	to := GetAccountInChat(r)
	if err := ch.storage.CheckUserExistence(to); err != nil {
		return err
	}
	from, err := ReadJWT(r, ch.storage)
	fmt.Println(from, err)
	if err != nil {
		return err
	}
	newMess := &Message{
		From:    from.Username,
		To:      to,
		Message: message.Message,
		Time:    time.Now().UTC(),
	}
	if err := ch.storage.CreateMessage(*newMess); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, newMess)
}

func (ch *ChatServer) GetInitialMessagesHandler(w http.ResponseWriter, r *http.Request) error {
	from, err := ReadJWT(r, ch.storage)
	if err != nil {
		return err
	}
	to := GetAccountInChat(r)
	messages, err := ch.storage.GetMessages(from.Username, to)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, messages)
}

func (ch *ChatServer) GetChatNamesHandler(w http.ResponseWriter, r *http.Request) error {
	from, err := ReadJWT(r, ch.storage)
	if err != nil {
		return err
	}
	names, err := ch.storage.GetChatNames(from.Username)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, names)
}

func (ch *ChatServer) LoginHandler(w http.ResponseWriter, r *http.Request) error {
	authError := fmt.Errorf("authorization error")
	if r.Method == "GET" {
		html, err := os.ReadFile("static/login.html")
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(html))
		return err
	}
	if r.Method != "POST" {
		return authError
	}
	var accReq Account
	if err := json.NewDecoder(r.Body).Decode(&accReq); err != nil {
		return err
	}
	if !accReq.Valid() {
		return authError
	}
	acc, err := ch.storage.GetUserByUsername(accReq.Username)
	if err != nil || !acc.Valid() {
		return authError
	}
	if !acc.ValidPassword(accReq.Password) {
		fmt.Println("Incorrect password")
		return authError
	}
	return CreateJWT(w, acc.Username)
}

func (ch *ChatServer) RegisterHandler(w http.ResponseWriter, r *http.Request) error {
	authError := fmt.Errorf("authorization error")
	if r.Method == "GET" {
		html, err := os.ReadFile("static/register.html")
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(html))
		return err
	}
	if r.Method != "POST" {
		return authError
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return authError
	}
	fmt.Println(req)
	if !req.Valid() {
		fmt.Println("Invalid request")
		return authError
	}
	var acc = &Account{
		Username: req.Username,
		Password: req.Password,
	}
	fmt.Println(acc)
	if err := acc.GenerateHashedPassword(); err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(acc)
	if err := ch.storage.CreateUser(*acc); err != nil {
		fmt.Println(err)
		return authError
	}
	return WriteJSON(w, http.StatusOK, acc.ToWebAccount())
}

func CreateJWT(w http.ResponseWriter, username string) error {
	claims := &jwt.MapClaims{
		"username": username,
	}
	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return err
	}
	cookie := http.Cookie{
		Name:     "x-jwt-token",
		Value:    tokenString,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(time.Minute * authorizationDuration),
	}
	http.SetCookie(w, &cookie)
	return nil
}

func DeleteJWT(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "x-jwt-token",
		Value:   "",
		Expires: time.Now(),
	})
}

func ReadJWT(r *http.Request, store Storage) (*Account, error) {
	unauthorized := fmt.Errorf("error unauthorized")
	tokenString, err := r.Cookie("x-jwt-token")
	if err != nil {
		return nil, unauthorized
	}
	token, err := GetJWTToken(tokenString.Value)
	if err != nil || !token.Valid {
		return nil, unauthorized
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["username"] == nil {
		return nil, unauthorized
	}
	account, err := store.GetUserByUsername(claims["username"].(string))

	return account, err
}

func GetJWTToken(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("incorrect token format: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}
