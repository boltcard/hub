package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func databaseDownload(w http.ResponseWriter) {

	dbPath := "/card_data/cards.db"

	dbBytes, err := os.ReadFile(dbPath)
	if err != nil {
		log.Warn("databaseDownload: failed to read database file: ", err.Error())
		http.Error(w, "failed to read database", http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("cards_%s.db", timestamp)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(dbBytes)
}

func databaseImport(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 50 MB max
	r.ParseMultipartForm(50 << 20)

	file, _, err := r.FormFile("database_file")
	if err != nil {
		log.Warn("databaseImport: failed to get uploaded file: ", err.Error())
		http.Error(w, "failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dbBytes, err := io.ReadAll(file)
	if err != nil {
		log.Warn("databaseImport: failed to read uploaded file: ", err.Error())
		http.Error(w, "failed to read uploaded file", http.StatusInternalServerError)
		return
	}

	// check for sqlite header
	if len(dbBytes) < 16 || string(dbBytes[:16]) != "SQLite format 3\x00" {
		http.Error(w, "file is not a valid SQLite database", http.StatusBadRequest)
		return
	}

	dbPath := "/card_data/cards.db"

	err = os.WriteFile(dbPath, dbBytes, 0644)
	if err != nil {
		log.Warn("databaseImport: failed to write database file: ", err.Error())
		http.Error(w, "failed to write database", http.StatusInternalServerError)
		return
	}

	log.Info("databaseImport: database imported successfully, restarting service")

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))

	// flush the response to the client before exiting
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// exit the process so the container restarts with the new database
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}
