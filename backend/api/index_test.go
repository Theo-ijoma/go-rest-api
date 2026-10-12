package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidAPIPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "root", path: "/", want: true},
		{name: "account route", path: "/auth/login", want: true},
		{name: "todo ID route", path: "/todos/123", want: true},
		{name: "missing leading slash", path: "auth/login", want: false},
		{name: "backslash", path: `/auth\login`, want: false},
		{name: "dot segment", path: "/auth/../todos", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validAPIPath(test.path); got != test.want {
				t.Errorf("validAPIPath(%q) = %t, want %t", test.path, got, test.want)
			}
		})
	}
}

func TestServeAPIUsesPathAndPreservesRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api?path=%2Fauth%2Flogin&source=test", nil)
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			t.Errorf("path = %q, want /auth/login", r.URL.Path)
		}
		if r.URL.Query().Get("path") != "" {
			t.Error("routing path query parameter should not reach API handler")
		}
		if r.URL.Query().Get("source") != "test" {
			t.Errorf("unrelated query parameter = %q, want test", r.URL.Query().Get("source"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	serveAPI(recorder, request, request.URL.Query().Get("path"), handler)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}
