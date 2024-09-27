package handlers

import (
	"net/http"
	"os"
)

func DownloadFile(w http.ResponseWriter, r *http.Request) {
	filePath := "./output/result.csv"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=result.csv")
	w.Header().Set("Content-Type", "text/csv")
	http.ServeFile(w, r, filePath)
}

func ServeFile(w http.ResponseWriter, r *http.Request) {
	filePath := "./output/result.csv"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}
