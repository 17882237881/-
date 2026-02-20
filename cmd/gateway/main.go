package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"gemini-clone/internal/common"
)

type sendReq struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
}

func main() {
	port := common.Getenv("PORT", "8080")
	commandURL := common.Getenv("CHAT_COMMAND_URL", "http://localhost:8081/submit")
	queryBase := common.Getenv("CHAT_QUERY_URL", "http://localhost:8082/stream")

	http.Handle("/", http.FileServer(http.Dir("api/gateway-go/static")))

	http.HandleFunc("/api/chat/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req sendReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		b, _ := json.Marshal(req)
		resp, err := http.Post(commandURL, "application/json", bytes.NewReader(b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	http.HandleFunc("/api/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.URL.Query().Get("request_id")
		if reqID == "" {
			http.Error(w, "missing request_id", http.StatusBadRequest)
			return
		}
		resp, err := http.Get(queryBase + "?request_id=" + reqID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	log.Printf("gateway listening :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
