package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeAPIUsesPathQueryAndKeepsOtherParameters(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/todos/507f1f77bcf86cd799439011" {
			t.Errorf("API path = %q, want todo path", r.URL.Path)
		}
		if r.URL.Query().Has("path") {
			t.Error("internal path routing parameter was not removed")
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", r.URL.Query().Get("limit"))
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api?path=%2Ftodos%2F507f1f77bcf86cd799439011&limit=10", nil)
	recorder := httptest.NewRecorder()
	serveAPI(recorder, request, "/todos/507f1f77bcf86cd799439011", api)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("function status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestValidAPIPathRejectsTraversal(t *testing.T) {
	for _, path := range []string{"/../secret", "/todos/../../auth", `\secret`} {
		if validAPIPath(path) {
			t.Errorf("validAPIPath(%q) = true, want false", path)
		}
	}
	if !validAPIPath("/todos/507f1f77bcf86cd799439011") {
		t.Fatal("valid todo route was rejected")
	}
}
