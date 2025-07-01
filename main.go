package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	// "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"context"
    "github.com/docker/docker/api/types/registry"
	"io"
	"log"
)

type dbase struct {
	Name string `json:"name"`
}

type DockerService struct {
	client *client.Client
	ctx    context.Context
}

func NewDockerService() (*DockerService, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerService{
		client: cli,
		ctx:    ctx,
	}, nil
}

func (ds *DockerService) Close() error {
	return ds.client.Close()
}

func (ds *DockerService) SearchImages(term string) ([]registry.SearchResult, error) {
	return ds.client.ImageSearch(ds.ctx, term, registry.SearchOptions{Limit: 5})
}

type Server struct {
	docker *DockerService
}

func (ds *DockerService) pullImage(term string) error {
	fmt.Println(term)
	out, err := ds.client.ImagePull(ds.ctx, term, image.PullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()

	// Optionally: stream output to log or terminal
	io.Copy(io.Discard, out) // Or use os.Stdout if you want to see progress

	return nil
}



var res string
func (s *Server) submitHandler(w http.ResponseWriter, r *http.Request) {
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
	// ctx:= context.Background()
	// cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// if err != nil {
	// 	http.Error(w, "Docker client error", http.StatusInternalServerError)
	// 	return
	// }
	// response := dbase{Name: "Hello from Go backend!"}

	results, err := s.docker.SearchImages(input.Name)
	if err != nil {
		fmt.Println("Docker search error:", err)
		http.Error(w, "Docker search error", http.StatusInternalServerError)
		return
	}
	fmt.Println(results)
	json.NewEncoder(w).Encode(results)
}

func (s *Server) imageHandler(w http.ResponseWriter, r *http.Request){
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
	fmt.Println(input.Name)

	err1:= s.docker.pullImage(input.Name)
	if err1 != nil {
		fmt.Println("Docker pull error:", err1)
		http.Error(w, "Docker pull error", http.StatusInternalServerError)
		return
	}
	//fmt.Println(results)
	json.NewEncoder(w).Encode(err1)
	//fmt.Println(err1)


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
	dockerService, err := NewDockerService()
	if err != nil {
		log.Fatal("Failed to create Docker service:", err)
	}
	defer dockerService.Close()

	// Create server with dependencies
	server := &Server{
		docker: dockerService,
	}
	
	fmt.Println("Server starting on http://localhost:8081")
	
	fmt.Println("")

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/api/submit", server.submitHandler)
	http.HandleFunc("/api/createImg",server.imageHandler)

	http.ListenAndServe(":8081", nil)
}
