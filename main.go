package main

import (
	"log"
	"net/http"
	"os"

	"github.com/adiakhileshsingh15/interiitprepathon-backend/handlers"
	"github.com/adiakhileshsingh15/interiitprepathon-backend/middlewares"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middlewares.EnableCORS)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Inter-IIT Prepathon Backend"))
	})

	r.Post("/upload", middlewares.FilePreparer(handlers.UploadFile))

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port :", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
