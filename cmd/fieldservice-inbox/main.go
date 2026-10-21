package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"fieldservice-inbox/internal/inbox"
)

func main() {
	client, err := inbox.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	teamInbox := os.Getenv("TEAM_INBOX_EMAIL")
	if teamInbox == "" {
		log.Fatal("TEAM_INBOX_EMAIL is required")
	}
	router := inbox.NewRouter(client, teamInbox)
	http.HandleFunc("/contact", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer req.Body.Close()
		var form inbox.ContactForm
		if err := json.NewDecoder(req.Body).Decode(&form); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		messageID, err := router.Submit(req.Context(), form, inbox.ClientIP(req))
		if err != nil {
			var apiErr *inbox.APIError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				http.Error(w, apiErr.Error(), apiErr.Status)
				return
			}
			http.Error(w, "unable to route contact form", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message_id": messageID})
	})
	log.Println("field-service inbox listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
