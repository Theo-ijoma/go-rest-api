package todos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// A request flows from a registered route below to its handler, which validates
// the HTTP input, calls the repository, and writes an HTTP response.

// Handler connects HTTP requests to the repository that stores todos.
type Handler struct {
	store Repository
}

// NewHandler builds the API routes and injects the repository they use.
// The repository is passed in instead of created here, which keeps HTTP code
// separate from MongoDB setup and makes the handlers easy to test.
func NewHandler(store Repository) http.Handler {
	handler := &Handler{store: store}
	mux := http.NewServeMux()

	// Go's ServeMux matches both the HTTP method and path. The {id} part is a
	// path parameter that handlers can read with r.PathValue("id").
	mux.HandleFunc("GET /", handler.home)
	mux.HandleFunc("GET /healthz", handler.health)
	mux.HandleFunc("GET /todos", handler.list)
	mux.HandleFunc("POST /todos", handler.create)
	mux.HandleFunc("GET /todos/{id}", handler.get)
	mux.HandleFunc("PUT /todos/{id}", handler.update)
	mux.HandleFunc("DELETE /todos/{id}", handler.delete)
	return mux
}

// home provides a quick message when opening the server's base URL in a browser.
func (h *Handler) home(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Go REST API is running",
		"endpoints": map[string]string{
			"health": "GET /healthz",
			"todos":  "GET /todos",
		},
	})
}

// health is a lightweight check that does not call MongoDB.
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// list handles GET /todos. The request context is passed through so cancelled
// HTTP requests can stop waiting on the database operation too.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	todos, err := h.store.List(r.Context())
	if err != nil {
		writeInternalError(w, "list todos", err)
		return
	}
	writeJSON(w, http.StatusOK, todos)
}

// create handles POST /todos. It decodes JSON, checks the title, saves the new
// todo, then responds with 201 Created and the saved document.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// An anonymous struct is a small one-off type for the expected request body.
	var input struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	todo, err := h.store.Create(r.Context(), input.Title)
	if err != nil {
		writeInternalError(w, "create todo", err)
		return
	}
	// Location tells API clients where this newly created resource can be fetched.
	w.Header().Set("Location", fmt.Sprintf("/todos/%s", todo.ID.Hex()))
	writeJSON(w, http.StatusCreated, todo)
}

// get handles GET /todos/{id}. A malformed ID is a 400; a valid but missing ID
// is a 404; a database failure is a 500.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	todo, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeInternalError(w, "get todo", err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

// update handles PUT /todos/{id}. PUT replaces the editable fields with the
// values in the JSON body, so both title and completed must be supplied.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var input struct {
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	todo, err := h.store.Update(r.Context(), id, input.Title, input.Completed)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeInternalError(w, "update todo", err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

// delete handles DELETE /todos/{id}. A successful delete has no response body,
// which is why it returns the HTTP status 204 No Content.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	err := h.store.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeInternalError(w, "delete todo", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseID converts the route's 24-character MongoDB ObjectID string into the
// driver's ObjectID type. It writes a 400 response itself when the ID is invalid.
func parseID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid MongoDB ObjectID")
		return bson.ObjectID{}, false
	}
	return id, true
}

// decodeJSON parses one JSON value from the request body. Unknown fields and
// extra JSON values are rejected so typos do not silently pass validation.
func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// writeJSON sets the response content type and encodes a Go value as JSON.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeError keeps error responses in one consistent JSON shape.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Database details are logged for the developer, while the response avoids
// exposing internal errors to API clients.
func writeInternalError(w http.ResponseWriter, operation string, err error) {
	log.Printf("failed to %s: %v", operation, err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
