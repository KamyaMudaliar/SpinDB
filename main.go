package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"


	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// -------------------- Structs --------------------

type dbase struct {
	Name string `json:"name"`
}

type DockerService struct {
	client *client.Client
	ctx    context.Context
}

type Server struct {
	docker *DockerService
}

// -------------------- Docker Service --------------------

func NewDockerService() (*DockerService, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerService{client: cli, ctx: ctx}, nil
}

func (ds *DockerService) Close() error {
	return ds.client.Close()
}

func (ds *DockerService) SearchImages(term string) ([]registry.SearchResult, error) {
	return ds.client.ImageSearch(ds.ctx, term, registry.SearchOptions{Limit: 5})
}

func (ds *DockerService) PullImage(imageName string) error {
	fmt.Println("Pulling image:", imageName)
	out, err := ds.client.ImagePull(ds.ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()

	io.Copy(io.Discard, out) // Discard output; use os.Stdout to log it
	return nil
}

func (ds *DockerService) CreateContainer(imageName string, containerName string, exposedPort string) error {	
    // 1. Pull the image if not present
    fmt.Println("Ensuring image is present:", imageName)
    _, _, err := ds.client.ImageInspectWithRaw(ds.ctx, imageName)
    if err != nil {
        fmt.Println("Image not found locally. Pulling...")
        reader, err := ds.client.ImagePull(ds.ctx, imageName, image.PullOptions{})
        if err != nil {
            return fmt.Errorf("error pulling image: %v", err)
        }
        defer reader.Close()
        io.Copy(io.Discard, reader) // discard pull output
    }

    // 2. Define port bindings if any
    portSet := nat.PortSet{}
    portMap := nat.PortMap{}
    if exposedPort != "" {
        port, err := nat.NewPort("tcp", exposedPort)
        if err != nil {
            return err
        }
        portSet[port] = struct{}{}
        portMap[port] = []nat.PortBinding{
            {HostIP: "0.0.0.0", HostPort: exposedPort},
        }
    }

    // 3. Create container
    resp, err := ds.client.ContainerCreate(
        ds.ctx,
        &container.Config{
            Image:        imageName,
            Tty:          true,
            ExposedPorts: portSet,
        },
        &container.HostConfig{
            PortBindings: portMap,
        },
        nil, nil,
        containerName,
    )
    if err != nil {
        return fmt.Errorf("error creating container: %v", err)
    }

    fmt.Println("Container created with ID:", resp.ID)

    // 4. Start container
    if err := ds.client.ContainerStart(ds.ctx, resp.ID, container.StartOptions{}); err != nil {
        return fmt.Errorf("error starting container: %v", err)
    }

    fmt.Println("Container started successfully.")
    return nil
}


// -------------------- Handlers --------------------

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) submitHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input dbase
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	fmt.Println("Searching for:", input.Name)
	results, err := s.docker.SearchImages(input.Name)
	if err != nil {
		log.Println("Docker search error:", err)
		http.Error(w, "Docker search failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *Server) imageHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input dbase
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	fmt.Println("Image to pull:", input.Name)
	if err := s.docker.PullImage(input.Name); err != nil {
		log.Println("Docker pull error:", err)
		http.Error(w, "Failed to pull image", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Image pulled successfully"})
}

func (s *Server) containerHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	var input dbase
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		fmt.Fprintln(w, "Invalid input")
		flusher.Flush()
		return
	}

	log := func(msg string) {
		fmt.Fprintln(w, msg)
		flusher.Flush()
	}

	log(fmt.Sprintf("Image to pull: %s", input.Name))

	if err := s.docker.PullImage(input.Name); err != nil {
		log(fmt.Sprintf("Error pulling image: %v", err))
		return
	}
	log("Image pulled successfully.")

	containerName := input.Name + "_container"
	port := "5432" // or 3306 if mysql
	if err := s.docker.CreateContainer(input.Name, containerName, port); err != nil {
		log(fmt.Sprintf("Error creating container: %v", err))
		return
	}
	log("Container started successfully.")
}


func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Go API backend is running.")
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// -------------------- Main --------------------

func main() {
	dockerService, err := NewDockerService()
	if err != nil {
		log.Fatal("Failed to create Docker service:", err)
	}
	defer dockerService.Close()

	server := &Server{docker: dockerService}

	fmt.Println("Server running on http://localhost:8081")

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/api/submit", server.submitHandler)
	http.HandleFunc("/api/createImg", server.imageHandler)
	http.HandleFunc("/api/createContainer", server.containerHandler)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal("Server error:", err)
	}
}
