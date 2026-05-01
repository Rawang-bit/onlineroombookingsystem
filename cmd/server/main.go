package main

import (
	"log"
	"net/http"
	"os"

	"smartbook-go/internal/handlers"
	"smartbook-go/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://smartbook:smartbook@localhost:5432/smartbook?sslmode=disable"
	}

	s, err := store.New(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	mux := http.NewServeMux()
	handlers.New(s).Register(mux)

	fileServer := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fileServer)

	log.Printf("SmartBook running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
