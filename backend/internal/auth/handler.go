package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

const sessionCookieName = "daymark_session"

type Handler struct {
	service         AccountService
	cookieSecure    bool
	sessionDuration time.Duration
}

type userContextKey struct{}

type AccountService interface {
	Register(context.Context, string, string, string) (User, string, error)
	SignIn(context.Context, string, string) (User, string, error)
	SessionUser(context.Context, string) (User, error)
	SignOut(context.Context, string) error
}

func NewHandler(service AccountService, cookieSecure bool, sessionDuration time.Duration) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure, sessionDuration: sessionDuration}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/login", h.signIn)
	mux.HandleFunc("POST /auth/logout", h.signOut)
	mux.Handle("GET /auth/me", h.Require(http.HandlerFunc(h.me)))
}

func (h *Handler) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "sign in to continue")
			return
		}
		user, err := h.service.SessionUser(r.Context(), cookie.Value)
		if errors.Is(err, ErrInvalidSession) {
			clearSessionCookie(w, h.cookieSecure)
			writeError(w, http.StatusUnauthorized, "sign in to continue")
			return
		}
		if err != nil {
			log.Printf("failed to validate session: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
	})
}

func CurrentUser(r *http.Request) User {
	user, _ := r.Context().Value(userContextKey{}).(User)
	return user
}

func CORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Add("Vary", "Origin")
		}
		if origin != "" && allowedOrigin != "" && origin != allowedOrigin && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			writeError(w, http.StatusForbidden, "request origin is not allowed")
			return
		}
		if r.Method == http.MethodOptions {
			if origin != "" && origin != allowedOrigin {
				writeError(w, http.StatusForbidden, "request origin is not allowed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, token, err := h.service.Register(r.Context(), input.FullName, input.Email, input.Password)
	if errors.Is(err, ErrEmailExists) {
		writeError(w, http.StatusConflict, "an account with that email already exists")
		return
	}
	if err != nil {
		var validationError ValidationError
		if errors.As(err, &validationError) {
			writeError(w, http.StatusBadRequest, validationError.Error())
			return
		}
		log.Printf("failed to register account: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, token, err := h.service.SignIn(r.Context(), input.Email, input.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "email or password is incorrect")
		return
	}
	if err != nil {
		log.Printf("failed to sign in: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) signOut(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if err := h.service.SignOut(r.Context(), cookie.Value); err != nil {
			log.Printf("failed to sign out: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}
	clearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, CurrentUser(r))
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionDuration.Seconds()),
		Expires:  time.Now().Add(h.sessionDuration),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errors.New("invalid JSON body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
