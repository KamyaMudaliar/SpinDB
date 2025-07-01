package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	// "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"context"
    // "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/registry"
    // "github.com/docker/docker/client"
)

type dbase struct {
	Name string `json:"name"`
}

var res string
func submitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	//cors preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}


	var input dbase
	fmt.Println("1")
	err := json.NewDecoder(r.Body).Decode(&input)
	if err!=nil{
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	fmt.Println(input)
	

	//docker client
	ctx:= context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Docker client error", http.StatusInternalServerError)
		return
	}
	// response := dbase{Name: "Hello from Go backend!"}
	res= input.Name
	results, err := cli.ImageSearch(ctx, res, registry.SearchOptions{Limit: 5})
	if err !=nil{
		fmt.Println("Docker search error:", err)
		http.Error(w, "Docker search error", http.StatusInternalServerError)
		return
	}
	fmt.Println(results)
	json.NewEncoder(w).Encode(results)
}

func imageHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	//cors preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input dbase
	err := json.NewDecoder(r.Body).Decode(&input)
	if err!=nil{
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	fmt.Println(input)
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
	fmt.Println("Server starting on http://localhost:8081")
	
	fmt.Println("")

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/api/submit", submitHandler)
	http.HandleFunc("/api/createImg",imageHandler)

	http.ListenAndServe(":8081", nil)
}
