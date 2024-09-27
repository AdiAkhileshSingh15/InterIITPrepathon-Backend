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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Inter-IIT Prepathon Backend"))
	})

	r.Post("/upload", middlewares.FilePreparer(handlers.UploadFile))
	r.Get("/result/download", handlers.DownloadFile)
	r.Get("/result", handlers.ServeFile)

	log.Println("Server running on port 3000")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
