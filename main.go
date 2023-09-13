package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type statReq struct {
	URI     string
	Headers map[string]string
	Body    string
	Method  string
	Time    string
}

var (
	mux = http.NewServeMux()
	// server *http.Server
	mu   sync.Mutex
	stat map[string][]statReq
)

func main() {

	// Load config from env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("LISTEN_PORT")

	// Create HTTPS server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	stat = make(map[string][]statReq)

	// Run server
	go func() {
		fmt.Println("Server started on port", port)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Add default route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

	})

	// Add routes in runtime (example)
	mux.HandleFunc("/addRoute", func(w http.ResponseWriter, r *http.Request) {
		path := normalizeURI(r.URL.Query().Get("path"))

		// check if path is valid
		if !isURIValid(path) {
			fmt.Fprintf(w, "Invalid path: %s\n", path)
			return
		}

		// Prepare request for checking if route already exists
		testReq, err := http.NewRequest("GET", path, nil)
		if err != nil {
			fmt.Fprintf(w, "Error creating request: %v\n", err)
			return
		}

		// check if route already exists
		_, pattern := mux.Handler(testReq)
		if pattern != "" {
			fmt.Fprintf(w, "Route [%s] already registered, remove first.\n", path)
			return
		}

		// handler := func(w http.ResponseWriter, r *http.Request) {
		// 	fmt.Fprintf(w, "Hello, You visited: %s\n", path)
		// }
		AddRoute(path, handler)
		fmt.Fprintf(w, "New route was added: %s\n", path)
	})

	// Remove routes in runtime (example)
	mux.HandleFunc("/removeRoute", func(w http.ResponseWriter, r *http.Request) {
		path := normalizeURI(r.URL.Query().Get("path"))

		// Prepare request for checking if route already exists
		testReq, err := http.NewRequest("GET", path, nil)
		if err != nil {
			fmt.Fprintf(w, "Error creating request: %v\n", err)
			return
		}

		// check if route not exists
		_, pattern := mux.Handler(testReq)
		if pattern == "" {
			fmt.Fprintf(w, "Route doesn't exist, nothing to remove: %s\n", path)
			return
		}

		RemoveRoute(path)
		fmt.Fprintf(w, "Route removed: %s\n", path)
	})

	// Get stat
	mux.HandleFunc("/getStat", func(w http.ResponseWriter, r *http.Request) {
		path := normalizeURI(r.URL.Query().Get("path"))

		// check if path is valid
		if !isURIValid(path) {
			fmt.Fprintf(w, "Invalid path: %s\n", path)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		statItems, ok := stat[path]
		if !ok {
			return
		}

		respData, err := json.Marshal(statItems)
		if err != nil {
			fmt.Fprintf(w, "Error marshaling stat: %v\n", err)
			return
		}
		fmt.Fprintf(w, "%s\n", respData)

		stat[path] = make([]statReq, 0)
	})

	<-make(chan struct{})
}

func handler(w http.ResponseWriter, r *http.Request) {
	// get pattern for request
	_, pattern := mux.Handler(r)

	// read body
	var body []byte
	r.Body.Read(body)

	// Add request to stat
	mu.Lock()
	if _, ok := stat[pattern]; !ok {
		stat[pattern] = make([]statReq, 0)
	}
	stat[pattern] = append(
		stat[pattern],
		statReq{
			URI: r.RequestURI,
			// Headers: r.Header.,
			Body:   string(body),
			Method: r.Method,
			Time:   time.Now().Format("2006-01-02 15:04:05.000"),
		},
	)

	mu.Unlock()

	fmt.Fprintf(w, "Hello, You visited: %s\n", pattern)
}

// AddRoute - add new route to mux
func AddRoute(path string, handler http.HandlerFunc) {
	mu.Lock()
	defer mu.Unlock()
	mux.HandleFunc(path, handler)
}

// RemoveRoute - remove route from mux
func RemoveRoute(path string) {
	mu.Lock()
	defer mu.Unlock()
	mux.HandleFunc(path, nil)
}

func isURIValid(uri string) bool {

	if uri == "" || uri == "/" {
		return false
	}

	parts := strings.Split(uri, "/")
	if parts[0] != "" {
		return false
	}

	parts = parts[1:]
	if len(parts) == 0 {
		return false
	}

	re := regexp.MustCompile(`^\w+$`)

	for _, part := range parts {
		if part == "" {
			return false
		}

		if !re.Match([]byte(part)) {
			return false
		}
	}

	return true
}

func normalizeURI(uri string) string {
	if uri[0] != '/' {
		uri = "/" + uri
	}
	return uri
}
