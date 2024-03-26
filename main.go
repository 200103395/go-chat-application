package main

import "log"

func main() {
	storage, err := NewPostgresStorage()
	if err != nil {
		log.Fatal(err)
	}

	if err := storage.Init(); err != nil {
		log.Fatal(err)
	}

	server := NewChatServer(":8080", *storage)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
