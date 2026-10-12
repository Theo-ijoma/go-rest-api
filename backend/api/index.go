package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"example.com/go-rest-api/internal/app"
)

var (
	initialize sync.Mutex
	apiHandler http.Handler
)

func Handler(w http.ResponseWriter, r *http.Request) {
	handler, err := getHandler()
	if err != nil {
		log.Printf("failed to initialize API: %v", err)
		writeError(w, http.StatusInternalServerError, "API initialization failed. Check the function logs and environment variables.")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = r.URL.Path
		if path == "/api" || path == "/api/" {
			path = "/"
		}
	}
	if !validAPIPath(path) {
		writeError(w, http.StatusBadRequest, "invalid API path")
		return
	}

	serveAPI(w, r, path, handler)
}

func getHandler() (http.Handler, error) {
	initialize.Lock()
	defer initialize.Unlock()
	if apiHandler != nil {
		return apiHandler, nil
	}
	handler, _, err := app.NewHandler()
	if err != nil {
		return nil, err
	}
	apiHandler = handler
	return apiHandler, nil
}

func serveAPI(w http.ResponseWriter, r *http.Request, path string, handler http.Handler) {
	request := r.Clone(r.Context())
	request.URL = new(url.URL)
	*request.URL = *r.URL
	request.URL.Path = path
	request.URL.RawPath = ""
	query := request.URL.Query()
	query.Del("path")
	request.URL.RawQuery = query.Encode()
	handler.ServeHTTP(w, request)
}

func validAPIPath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "\\") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("failed to write API error: %v", err)
	}
}
