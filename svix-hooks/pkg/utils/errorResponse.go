package utils

import (
	"fmt"
	"log"
	"net/http"
)

// ErrorResponse helper function for returning error responses
func ErrorResponse(w http.ResponseWriter, err error, message string, statusCode int) {
	errorMessage := fmt.Sprintf("%s:%s", message, err)
	log.Printf("Error %d: %s\n", statusCode, errorMessage)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, errorMessage)))
	if err != nil {
		log.Printf("Error writing response: %s\n", err)
		return
	}
}

// ServerErrorResponse helper function for returning 500 error responses
func ServerErrorResponse(w http.ResponseWriter, err error, message string) {
	ErrorResponse(w, err, message, http.StatusInternalServerError)
}

// BadRequestResponse helper function for returning 400 error responses
func BadRequestResponse(w http.ResponseWriter, err error, message string) {
	ErrorResponse(w, err, message, http.StatusBadRequest)
}
