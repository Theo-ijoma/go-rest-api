package handler

import (
	"net/http"

	"example.com/go-rest-api/internal/vercelapi"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	vercelapi.ServeHTTP(w, r)
}
