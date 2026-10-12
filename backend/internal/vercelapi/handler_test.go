package vercelapi

import "testing"

func TestAPIPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "function root", path: "/api", want: "/"},
		{name: "root function", path: "/api/index", want: "/"},
		{name: "health route", path: "/api/healthz", want: "/healthz"},
		{name: "nested API route", path: "/api/auth/register", want: "/auth/register"},
		{name: "todo ID route", path: "/api/todos/123", want: "/todos/123"},
		{name: "non API path remains unchanged", path: "/healthz", want: "/healthz"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := apiPath(test.path); got != test.want {
				t.Errorf("apiPath(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}
