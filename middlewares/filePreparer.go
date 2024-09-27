package middlewares

import (
	"fmt"
	"net/http"
	"path/filepath"
)

func FilePreparer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		ext := filepath.Ext(handler.Filename)
		if ext != ".fits" && ext != ".csv" && ext != ".lc" {
			http.Error(w, "Invalid file type", http.StatusBadRequest)
			return
		}

		handler.Filename = "file" + ext

		fmt.Println("Pre-processing of the file complete")
		next(w, r)
	}
}
