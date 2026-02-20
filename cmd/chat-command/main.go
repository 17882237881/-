package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gemini-clone/internal/common"
)

type submitReq struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
}

type submitResp struct {
	RequestID string `json:"request_id"`
}

func main() {
	port := common.Getenv("PORT", "8081")
	dataDir := common.Getenv("DATA_DIR", "./data")
	_ = os.MkdirAll(filepath.Join(dataDir, "requests"), 0o755)

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req submitReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Prompt == "" {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		reqID := time.Now().Format("20060102150405.000000000")
		b, _ := json.Marshal(req)
		if err := os.WriteFile(filepath.Join(dataDir, "requests", reqID+".json"), b, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(submitResp{RequestID: reqID})
	})

	log.Printf("chat-command listening :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
