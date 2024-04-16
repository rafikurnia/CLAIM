package p

import (
	"encoding/json"
	"net/http"
)

// Data structure for response  on HTTP calls
type HTTPResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// A function that return message and code on HTTP calls

func sendRespond(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(HTTPResponse{Code: status, Message: msg})
}
