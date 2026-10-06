package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"devtools-image-compressor/internal/compression"
)

type processRequest struct {
	Image  []byte `json:"image"`
	Format string `json:"format"`
}

func main() {
	client, err := compression.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var input processRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := client.Compress(input.Image, input.Format)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		before := int64(len(input.Image))
		after := before
		if value, ok := result.Metadata["size"]; ok {
			if number, ok := value.(float64); ok {
				after = int64(number)
			}
		}
		decision := compression.DecideRelease(before, after)
		status := http.StatusAccepted
		if !decision.Publish {
			status = http.StatusUnprocessableEntity
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"publish": decision.Publish, "reason": decision.Reason, "data": result.Data})
	})
	port := 8080
	if value := rport(); value > 0 {
		port = value
	}
	log.Printf("image release service listening on :%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

func rport() int {
	if value := os.Getenv("PORT"); value != "" {
		port, _ := strconv.Atoi(value)
		return port
	}
	return 0
}
