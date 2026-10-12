package handler

import (
	"net/http"
	"net/url"

	"example.com/go-rest-api/internal/vercelapi"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("todoId")
	if id == "" {
		http.Error(w, "todo ID is required", http.StatusBadRequest)
		return
	}

	request := r.Clone(r.Context())
	request.URL = new(url.URL)
	*request.URL = *r.URL
	request.URL.Path = "/api/todos/" + id
	query := request.URL.Query()
	query.Del("todoId")
	request.URL.RawQuery = query.Encode()
	vercelapi.ServeHTTP(w, request)
}
