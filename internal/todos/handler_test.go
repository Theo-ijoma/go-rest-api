package todos

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTodoLifecycle(t *testing.T) {
	handler := NewHandler()

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
	if todo.ID != 1 || todo.Title != "Learn Go" || todo.Completed {
		t.Fatalf("created todo = %#v, want ID 1, title Learn Go, incomplete", todo)
	}

	updated := request(handler, http.MethodPut, "/todos/1", `{"title":"Learn Go well","completed":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("PUT /todos/1 status = %d, want %d", updated.Code, http.StatusOK)
	}

	fetched := request(handler, http.MethodGet, "/todos/1", "")
	if fetched.Code != http.StatusOK {
		t.Fatalf("GET /todos/1 status = %d, want %d", fetched.Code, http.StatusOK)
	}
	if err := json.Unmarshal(fetched.Body.Bytes(), &todo); err != nil {
		t.Fatalf("decode fetched todo: %v", err)
	}
	if todo.Title != "Learn Go well" || !todo.Completed {
		t.Fatalf("fetched todo = %#v, want updated title and completed", todo)
	}

	deleted := request(handler, http.MethodDelete, "/todos/1", "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("DELETE /todos/1 status = %d, want %d", deleted.Code, http.StatusNoContent)
	}

	missing := request(handler, http.MethodGet, "/todos/1", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("GET deleted todo status = %d, want %d", missing.Code, http.StatusNotFound)
	}
}

func TestCreateRequiresTitle(t *testing.T) {
	response := request(NewHandler(), http.MethodPost, "/todos", `{"title":"  "}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /todos status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func request(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
