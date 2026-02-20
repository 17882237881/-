package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gemini-clone/internal/common"
)

func main() {
	port := common.Getenv("PORT", "8082")
	dataDir := common.Getenv("DATA_DIR", "./data")
	_ = os.MkdirAll(filepath.Join(dataDir, "streams"), 0o755)

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.URL.Query().Get("request_id")
		if reqID == "" {
			http.Error(w, "missing request_id", http.StatusBadRequest)
			return
		}
		path := filepath.Join(dataDir, "streams", reqID+".stream")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}
		deadline := time.Now().Add(60 * time.Second)
		offset := int64(0)
		for time.Now().Before(deadline) {
			f, err := os.Open(path)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			_, _ = f.Seek(offset, 0)
			s := bufio.NewScanner(f)
			for s.Scan() {
				line := s.Text()
				offset += int64(len(line) + 1)
				if line == "[DONE]" {
					fmt.Fprint(w, "event: done\ndata: [DONE]\n\n")
					flusher.Flush()
					_ = f.Close()
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", strings.TrimSpace(line))
				flusher.Flush()
			}
			_ = f.Close()
			time.Sleep(80 * time.Millisecond)
		}
		fmt.Fprint(w, "event: done\ndata: timeout\n\n")
		flusher.Flush()
	})

	log.Printf("chat-query listening :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
