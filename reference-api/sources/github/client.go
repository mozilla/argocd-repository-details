package github

import (
	"encoding/json"
	"net/http"
)

func errorEncoder(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{Error: message})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func responseEncoder(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
