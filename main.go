package main

import (
	"log"
	"net/http"

	"github.com/adiakhileshsingh15/interiitprepathon-backend/handlers"
	"github.com/adiakhileshsingh15/interiitprepathon-backend/middlewares"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middlewares.EnableCORS)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Inter-IIT Prepathon Backend"))
	})

	r.Post("/upload", middlewares.FilePreparer(handlers.UploadFile))

	log.Println("Server running on port 8000")
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
