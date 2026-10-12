package vercelapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"example.com/go-rest-api/internal/app"
)

var (
	initialize sync.Once
	handler    http.Handler
	initErr    error
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	initialize.Do(func() {
		handler, _, initErr = app.NewHandler()
	})
	if initErr != nil {
		log.Printf("failed to initialize API: %v", initErr)
		writeError(w, "API initialization failed. Check the function logs and environment variables.")
		return
	}

	request := r.Clone(r.Context())
	request.URL.Path = apiPath(r.URL.Path)
	request.URL.RawPath = ""
	handler.ServeHTTP(w, request)
}

func apiPath(path string) string {
	if path == "/api/index" {
		return "/"
	}
	path = strings.TrimPrefix(path, "/api")
	if path == "" {
		return "/"
	}
	return path
}

func writeError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("failed to write initialization error: %v", err)
	}
}
