package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

// UploadFile handles file uploads and processing
func UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to process file", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err = os.Mkdir(dataDir, 0755)
		if err != nil {
			http.Error(w, "Unable to create uploads directory", http.StatusInternalServerError)
			return
		}
	}

	filePath := filepath.Join(dataDir, handler.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error copying the file", http.StatusInternalServerError)
		return
	}

	// call python script for our model
	cmd := exec.Command("python3", "./model.py")
	cmdInput, err := cmd.StdinPipe()
	if err != nil {
		http.Error(w, "Failed to create stdin pipe for python command", http.StatusInternalServerError)
		return
	}
	cmdOutput, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to create stdout pipe for python command", http.StatusInternalServerError)
		return
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start python script: %v", err), http.StatusInternalServerError)
		return
	}

	// write file path to python script input
	if _, err := cmdInput.Write([]byte(filePath)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to send input to python script: %v", err), http.StatusInternalServerError)
		return
	}
	cmdInput.Close()

	// read python script output
	output, err := io.ReadAll(cmdOutput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read output from python script: %v", err), http.StatusInternalServerError)
		return
	}

	// wait for model to run
	if err := cmd.Wait(); err != nil {
		http.Error(w, fmt.Sprintf("Python script finished with error: %v", err), http.StatusInternalServerError)
		return
	}

	// parse output from python script as json
	type PythonOutput struct {
		DetectedFlares [][]interface{}          `json:"detected_flares"`
		LcData         []map[string]interface{} `json:"lc_data"`
	}

	var result PythonOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse JSON from Python script: %v", err), http.StatusInternalServerError)
		return
	}

	// save result to csv file
	err = saveResultToFile(result.DetectedFlares)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save result to CSV: %v", err), http.StatusInternalServerError)
		return
	}

	// send result as json response
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, "Failed to send result as JSON", http.StatusInternalServerError)
		return
	}
}

func saveResultToFile(data [][]interface{}) error {
	outDir := "./output"
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err = os.Mkdir(outDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	filePath := filepath.Join(outDir, "result.csv")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create result.csv file: %w", err)
	}
	defer file.Close()

	// write csv headers
	_, err = file.WriteString("flare_type,start,precise_start,start_rate,peak,peak_rate,background_level,decay,decay_rate\n")
	if err != nil {
		return fmt.Errorf("failed to write headers to result.csv: %w", err)
	}

	// write the result data in csv format
	for j := 0; j < len(data[0]); j++ {
		for i := 0; i < 9; i++ {
			if i < len(data) {
				switch data[i][j].(type) {
				case float64:
					file.WriteString(fmt.Sprintf("%.8f", data[i][j]))
				case string:
					file.WriteString(fmt.Sprintf("%s", data[i][j]))
				}
				if i != 8 {
					file.WriteString(",")
				} else if j != len(data[0])-1 {
					file.WriteString("\n")
				}
			}
		}
	}
	return nil
}
