package app

import "testing"

func TestFrontendOrigin(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "", want: "http://localhost:5173"},
		{input: " http://localhost:5173/ ", want: "http://localhost:5173"},
		{input: "https://daymark.example", want: "https://daymark.example"},
	} {
		got, err := FrontendOrigin(test.input)
		if err != nil || got != test.want {
			t.Errorf("FrontendOrigin(%q) = %q, %v; want %q, nil", test.input, got, err, test.want)
		}
	}

	for _, input := range []string{
		"ftp://daymark.example",
		"https://daymark.example/app",
		"https://user@daymark.example",
		"https://daymark.example?next=/",
	} {
		if _, err := FrontendOrigin(input); err == nil {
			t.Errorf("FrontendOrigin(%q) succeeded, want validation error", input)
		}
	}
}
