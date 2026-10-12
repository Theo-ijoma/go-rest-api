package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestRegisterValidatesInputBeforeDatabaseAccess(t *testing.T) {
	service := NewService(nil, nil, time.Hour)
	tests := []struct {
		name     string
		fullName string
		email    string
		password string
	}{
		{name: "empty name", email: "person@example.com", password: "long-enough-password"},
		{name: "invalid email", fullName: "Person", email: "not-an-email", password: "long-enough-password"},
		{name: "short password", fullName: "Person", email: "person@example.com", password: "short"},
		{name: "long password", fullName: "Person", email: "person@example.com", password: strings.Repeat("x", 73)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := service.Register(context.Background(), test.fullName, test.email, test.password)
			var validationError ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("Register error = %v, want ValidationError", err)
			}
		})
	}
}

func TestUserDocumentStoresProfileAndPasswordHashAtTopLevel(t *testing.T) {
	document := struct {
		User         `bson:",inline"`
		PasswordHash string `bson:"passwordHash"`
	}{
		User:         User{ID: bson.NewObjectID(), FullName: "Taylor Example", Email: "taylor@example.com"},
		PasswordHash: "bcrypt-hash",
	}
	encoded, err := bson.Marshal(document)
	if err != nil {
		t.Fatalf("marshal user document: %v", err)
	}
	var stored bson.M
	if err := bson.Unmarshal(encoded, &stored); err != nil {
		t.Fatalf("decode user document: %v", err)
	}
	if stored["email"] != "taylor@example.com" || stored["fullName"] != "Taylor Example" || stored["passwordHash"] != "bcrypt-hash" {
		t.Fatalf("stored user fields = %#v, want email, fullName, and passwordHash at top level", stored)
	}
	if _, ok := stored["user"]; ok {
		t.Fatalf("stored user fields are unexpectedly nested: %#v", stored)
	}

	var account struct {
		User         `bson:",inline"`
		PasswordHash string `bson:"passwordHash"`
	}
	if err := bson.Unmarshal(encoded, &account); err != nil {
		t.Fatalf("decode account document: %v", err)
	}
	if account.User.FullName != document.User.FullName || account.User.Email != document.User.Email {
		t.Fatalf("decoded account profile = %#v, want %+v", account.User, document.User)
	}

	sessionDocument, err := bson.Marshal(session{
		ID:        "session-id",
		User:      document.User,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("marshal session document: %v", err)
	}
	var storedSession session
	if err := bson.Unmarshal(sessionDocument, &storedSession); err != nil {
		t.Fatalf("decode session document: %v", err)
	}
	if storedSession.User.FullName != document.User.FullName || storedSession.User.Email != document.User.Email {
		t.Fatalf("decoded session user = %#v, want %+v", storedSession.User, document.User)
	}
}

func TestLoginSetsHttpOnlyCookieAndLogoutRevokesSession(t *testing.T) {
	service := &memoryAccountService{
		user:  User{ID: bson.NewObjectID(), FullName: "Taylor Example", Email: "taylor@example.com"},
		token: "secret-session-token",
	}
	handler := NewHandler(service, true, time.Hour)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	login := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"taylor@example.com","password":"password"}`))
	loginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(loginRecorder, login)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("POST /auth/login status = %d, want %d; body: %s", loginRecorder.Code, http.StatusOK, loginRecorder.Body)
	}
	cookies := loginRecorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("login cookies = %#v, want one secure HttpOnly session cookie", cookies)
	}
	if !strings.Contains(loginRecorder.Header().Get("Set-Cookie"), "SameSite=Lax") {
		t.Fatalf("Set-Cookie = %q, want SameSite=Lax", loginRecorder.Header().Get("Set-Cookie"))
	}
	var response User
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if response.Email != service.user.Email || strings.Contains(loginRecorder.Body.String(), service.token) {
		t.Fatalf("login response should return the profile without exposing the session token: %s", loginRecorder.Body)
	}

	signOut := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	signOut.AddCookie(cookies[0])
	signOutRecorder := httptest.NewRecorder()
	mux.ServeHTTP(signOutRecorder, signOut)
	if signOutRecorder.Code != http.StatusNoContent || service.active {
		t.Fatalf("POST /auth/logout status = %d active session = %v, want 204 and revoked", signOutRecorder.Code, service.active)
	}
}

func TestRegisterCreatesAccountWithoutBasicAuthChallenge(t *testing.T) {
	service := &memoryAccountService{
		user:  User{ID: bson.NewObjectID(), FullName: "Taylor Example", Email: "taylor@example.com"},
		token: "new-session-token",
	}
	handler := NewHandler(service, false, time.Hour)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(
		`{"fullName":"Taylor Example","email":"taylor@example.com","password":"password"}`,
	))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /auth/register status = %d, want %d; body: %s", recorder.Code, http.StatusCreated, recorder.Body)
	}
	if recorder.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("registration must not trigger a browser Basic Auth prompt")
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly {
		t.Fatalf("registration cookies = %#v, want an HttpOnly session cookie", cookies)
	}
	if strings.Contains(recorder.Body.String(), "password") || strings.Contains(recorder.Body.String(), service.token) {
		t.Fatalf("registration response exposed credentials: %s", recorder.Body)
	}
}

func TestCORSRejectsUntrustedMutationOrigin(t *testing.T) {
	handler := CORS("http://localhost:5173", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	request.Header.Set("Origin", "https://malicious.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("POST from untrusted origin status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCORSAllowsCredentialedConfiguredOrigin(t *testing.T) {
	handler := CORS("http://localhost:5173", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS from configured origin status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", recorder.Header().Get("Access-Control-Allow-Credentials"))
	}
}

type memoryAccountService struct {
	user   User
	token  string
	active bool
}

func (s *memoryAccountService) Register(context.Context, string, string, string) (User, string, error) {
	s.active = true
	return s.user, s.token, nil
}

func (s *memoryAccountService) SignIn(context.Context, string, string) (User, string, error) {
	s.active = true
	return s.user, s.token, nil
}

func (s *memoryAccountService) SessionUser(_ context.Context, token string) (User, error) {
	if !s.active || token != s.token {
		return User{}, ErrInvalidSession
	}
	return s.user, nil
}

func (s *memoryAccountService) SignOut(_ context.Context, token string) error {
	if token == s.token {
		s.active = false
	}
	return nil
}
