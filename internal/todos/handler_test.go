package todos

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTodoLifecycle(t *testing.T) {
	handler := NewHandler(newMemoryRepository())

	home := request(handler, http.MethodGet, "/", "")
	if home.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", home.Code, http.StatusOK)
	}

	created := request(handler, http.MethodPost, "/todos", `{"title":"Learn Go"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("POST /todos status = %d, want %d; body: %s", created.Code, http.StatusCreated, created.Body)
	}

	var todo Todo
	if err := json.Unmarshal(created.Body.Bytes(), &todo); err != nil {
		t.Fatalf("decode created todo: %v", err)
	}
	if todo.ID.IsZero() || todo.Title != "Learn Go" || todo.Completed {
		t.Fatalf("created todo = %#v, want a generated ID, title Learn Go, and incomplete status", todo)
	}

	todoURL := "/todos/" + todo.ID.Hex()
	updated := request(handler, http.MethodPut, todoURL, `{"title":"Learn Go well","completed":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("PUT %s status = %d, want %d", todoURL, updated.Code, http.StatusOK)
	}

	fetched := request(handler, http.MethodGet, todoURL, "")
	if fetched.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", todoURL, fetched.Code, http.StatusOK)
	}
	if err := json.Unmarshal(fetched.Body.Bytes(), &todo); err != nil {
		t.Fatalf("decode fetched todo: %v", err)
	}
	if todo.Title != "Learn Go well" || !todo.Completed {
		t.Fatalf("fetched todo = %#v, want updated title and completed", todo)
	}

	deleted := request(handler, http.MethodDelete, todoURL, "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("DELETE %s status = %d, want %d", todoURL, deleted.Code, http.StatusNoContent)
	}

	missing := request(handler, http.MethodGet, todoURL, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("GET deleted todo status = %d, want %d", missing.Code, http.StatusNotFound)
	}
}

func TestCreateRequiresTitle(t *testing.T) {
	response := request(NewHandler(newMemoryRepository()), http.MethodPost, "/todos", `{"title":"  "}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /todos status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestGetRejectsInvalidObjectID(t *testing.T) {
	response := request(NewHandler(newMemoryRepository()), http.MethodGet, "/todos/1", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET /todos/1 status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

// memoryRepository is only a test double; the running API always uses MongoDB.
type memoryRepository struct {
	todos map[bson.ObjectID]Todo
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{todos: make(map[bson.ObjectID]Todo)}
}

func (r *memoryRepository) List(_ context.Context) ([]Todo, error) {
	todos := make([]Todo, 0, len(r.todos))
	for _, todo := range r.todos {
		todos = append(todos, todo)
	}
	return todos, nil
}

func (r *memoryRepository) Create(_ context.Context, title string) (Todo, error) {
	todo := Todo{ID: bson.NewObjectID(), Title: title}
	r.todos[todo.ID] = todo
	return todo, nil
}

func (r *memoryRepository) Get(_ context.Context, id bson.ObjectID) (Todo, error) {
	todo, ok := r.todos[id]
	if !ok {
		return Todo{}, ErrNotFound
	}
	return todo, nil
}

func (r *memoryRepository) Update(_ context.Context, id bson.ObjectID, title string, completed bool) (Todo, error) {
	todo, ok := r.todos[id]
	if !ok {
		return Todo{}, ErrNotFound
	}
	todo.Title = title
	todo.Completed = completed
	r.todos[id] = todo
	return todo, nil
}

func (r *memoryRepository) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := r.todos[id]; !ok {
		return ErrNotFound
	}
	delete(r.todos, id)
	return nil
}

var _ Repository = (*memoryRepository)(nil)

func request(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	// httptest runs a request directly through the handler without opening a port.
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
