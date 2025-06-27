package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type dbase struct {
	Name string `json:"name"`
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	response := dbase{Name: "Hello from Go backend!"}
	json.NewEncoder(w).Encode(response)
}

// Root handler for `/`
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Go API backend is running.")
}

// Optional: Handle favicon requests
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	// Return 204 No Content to avoid 404 errors
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	fmt.Println("Server starting on http://localhost:8080")

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/api/submit", submitHandler)

	http.ListenAndServe(":8080", nil)
}
