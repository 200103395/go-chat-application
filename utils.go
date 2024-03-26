package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request) error

type ErrorJson struct {
	Error string `json:"error"`
}

func GetAccountInChat(r *http.Request) string {
	username := mux.Vars(r)["username"]
	return username
}

func MakeHTTPHandler(f ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			fmt.Println("Error is ", err)
			fmt.Println("Write JSON error: ", WriteJSON(w, http.StatusBadRequest, &ErrorJson{Error: err.Error()}))
		}
	}
}

func withJWTAuth(f http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := ReadJWT(r, s); err != nil {
			html, err := os.ReadFile("static/redirectIndex.html")
			if err != nil {
				fmt.Println(err)
				return
			}
			_, err = fmt.Fprint(w, string(html))
			if err != nil {
				fmt.Println(err)
			}
			return
		}
		f(w, r)
	}
}
