package todos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/go-rest-api/internal/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTodoLifecycle(t *testing.T) {
	handler := NewHandler(newMemoryRepository(), testAuthenticator{})

	home := request(handler, http.MethodGet, "/", "")
	if home.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", home.Code, http.StatusOK)
	}

	created := authenticatedRequest(handler, http.MethodPost, "/todos", `{"title":"Learn Go"}`)
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
	if strings.Contains(created.Body.String(), "userId") {
		t.Fatalf("todo response exposes its internal owner ID: %s", created.Body.String())
	}

	todoURL := "/todos/" + todo.ID.Hex()
	updated := authenticatedRequest(handler, http.MethodPut, todoURL, `{"title":"Learn Go well","completed":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("PUT %s status = %d, want %d", todoURL, updated.Code, http.StatusOK)
	}

	fetched := authenticatedRequest(handler, http.MethodGet, todoURL, "")
	if fetched.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", todoURL, fetched.Code, http.StatusOK)
	}
	if err := json.Unmarshal(fetched.Body.Bytes(), &todo); err != nil {
		t.Fatalf("decode fetched todo: %v", err)
	}
	if todo.Title != "Learn Go well" || !todo.Completed {
		t.Fatalf("fetched todo = %#v, want updated title and completed", todo)
	}

	deleted := authenticatedRequest(handler, http.MethodDelete, todoURL, "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("DELETE %s status = %d, want %d", todoURL, deleted.Code, http.StatusNoContent)
	}

	missing := authenticatedRequest(handler, http.MethodGet, todoURL, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("GET deleted todo status = %d, want %d", missing.Code, http.StatusNotFound)
	}
}

func TestCreateRequiresTitle(t *testing.T) {
	response := authenticatedRequest(NewHandler(newMemoryRepository(), testAuthenticator{}), http.MethodPost, "/todos", `{"title":"  "}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /todos status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestGetRejectsInvalidObjectID(t *testing.T) {
	response := authenticatedRequest(NewHandler(newMemoryRepository(), testAuthenticator{}), http.MethodGet, "/todos/1", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET /todos/1 status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestTodosRequireSession(t *testing.T) {
	handler := NewHandler(newMemoryRepository(), auth.NewHandler(&testAccountService{}, false, time.Hour))

	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/todos"},
		{method: http.MethodPost, path: "/todos"},
		{method: http.MethodGet, path: "/todos/507f1f77bcf86cd799439011"},
		{method: http.MethodPut, path: "/todos/507f1f77bcf86cd799439011"},
		{method: http.MethodDelete, path: "/todos/507f1f77bcf86cd799439011"},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := request(handler, test.method, test.path, "")
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			if !strings.Contains(response.Body.String(), "sign in") {
				t.Fatalf("response does not explain sign in is required: %s", response.Body.String())
			}
			if response.Header().Get("WWW-Authenticate") != "" {
				t.Fatal("session auth must not trigger a browser Basic Auth prompt")
			}
		})
	}
}

func TestHealthDoesNotRequireSession(t *testing.T) {
	response := request(NewHandler(newMemoryRepository(), testAuthenticator{}), http.MethodGet, "/healthz", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestCORSPreflightAllowsConfiguredFrontend(t *testing.T) {
	handler := auth.CORS("http://localhost:5173", NewHandler(newMemoryRepository(), testAuthenticator{}))
	request := httptest.NewRequest(http.MethodOptions, "/todos", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS /todos status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want http://localhost:5173", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want Content-Type", got)
	}
}

func TestTodosAreIsolatedByUser(t *testing.T) {
	repository := newMemoryRepository()
	firstUser := auth.User{ID: bson.NewObjectID()}
	secondUser := auth.User{ID: bson.NewObjectID()}
	firstHandler := NewHandler(repository, testAuthenticator{userID: firstUser.ID})
	created := authenticatedRequest(firstHandler, http.MethodPost, "/todos", `{"title":"Private task"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("first user's create status = %d, want %d", created.Code, http.StatusCreated)
	}

	var firstTodo Todo
	if err := json.Unmarshal(created.Body.Bytes(), &firstTodo); err != nil {
		t.Fatalf("decode first user's todo: %v", err)
	}
	secondHandler := NewHandler(repository, testAuthenticator{userID: secondUser.ID})
	listed := authenticatedRequest(secondHandler, http.MethodGet, "/todos", "")
	var todos []Todo
	if err := json.Unmarshal(listed.Body.Bytes(), &todos); err != nil {
		t.Fatalf("decode second user's todo list: %v", err)
	}
	if listed.Code != http.StatusOK || len(todos) != 0 {
		t.Fatalf("second user's list = %s status %d, want an empty 200 response", listed.Body, listed.Code)
	}
	fetched := authenticatedRequest(secondHandler, http.MethodGet, "/todos/"+firstTodo.ID.Hex(), "")
	if fetched.Code != http.StatusNotFound {
		t.Fatalf("second user's get status = %d, want %d", fetched.Code, http.StatusNotFound)
	}
	deleted := authenticatedRequest(secondHandler, http.MethodDelete, "/todos/"+firstTodo.ID.Hex(), "")
	if deleted.Code != http.StatusNotFound {
		t.Fatalf("second user's delete status = %d, want %d", deleted.Code, http.StatusNotFound)
	}
	updated := authenticatedRequest(secondHandler, http.MethodPut, "/todos/"+firstTodo.ID.Hex(), `{"title":"stolen task","completed":true}`)
	if updated.Code != http.StatusNotFound {
		t.Fatalf("second user's update status = %d, want %d", updated.Code, http.StatusNotFound)
	}
}

// memoryRepository is only a test double; the running API always uses MongoDB.
type memoryRepository struct {
	todos map[bson.ObjectID]Todo
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{todos: make(map[bson.ObjectID]Todo)}
}

func (r *memoryRepository) List(_ context.Context, userID bson.ObjectID) ([]Todo, error) {
	todos := make([]Todo, 0, len(r.todos))
	for _, todo := range r.todos {
		if todo.UserID == userID {
			todos = append(todos, todo)
		}
	}
	return todos, nil
}

func (r *memoryRepository) Create(_ context.Context, userID bson.ObjectID, title string) (Todo, error) {
	todo := Todo{ID: bson.NewObjectID(), UserID: userID, Title: title}
	r.todos[todo.ID] = todo
	return todo, nil
}

func (r *memoryRepository) Get(_ context.Context, userID, id bson.ObjectID) (Todo, error) {
	todo, ok := r.todos[id]
	if !ok || todo.UserID != userID {
		return Todo{}, ErrNotFound
	}
	return todo, nil
}

func (r *memoryRepository) Update(_ context.Context, userID, id bson.ObjectID, title string, completed bool) (Todo, error) {
	todo, ok := r.todos[id]
	if !ok || todo.UserID != userID {
		return Todo{}, ErrNotFound
	}
	todo.Title = title
	todo.Completed = completed
	r.todos[id] = todo
	return todo, nil
}

func (r *memoryRepository) Delete(_ context.Context, userID, id bson.ObjectID) error {
	if todo, ok := r.todos[id]; !ok || todo.UserID != userID {
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

func authenticatedRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(&http.Cookie{Name: "daymark_session", Value: "test-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type testAuthenticator struct {
	userID bson.ObjectID
}

func (a testAuthenticator) Require(next http.Handler) http.Handler {
	return auth.NewHandler(&testAccountService{userID: a.userID}, false, time.Hour).Require(next)
}

type testAccountService struct {
	userID bson.ObjectID
}

func (*testAccountService) Register(context.Context, string, string, string) (auth.User, string, error) {
	return auth.User{}, "", errors.New("unexpected register call")
}

func (*testAccountService) SignIn(context.Context, string, string) (auth.User, string, error) {
	return auth.User{}, "", errors.New("unexpected sign-in call")
}

func (s *testAccountService) SessionUser(_ context.Context, token string) (auth.User, error) {
	if token != "test-token" {
		return auth.User{}, auth.ErrInvalidSession
	}
	userID := s.userID
	if userID.IsZero() {
		userID = bson.ObjectID{1}
	}
	return auth.User{ID: userID}, nil
}

func (*testAccountService) SignOut(context.Context, string) error {
	return nil
}
