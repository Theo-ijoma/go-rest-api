package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/go-rest-api/internal/auth"
	"example.com/go-rest-api/internal/todos"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewHandler() (http.Handler, func() error, error) {
	cookieSecure := os.Getenv("VERCEL") == "1"
	if configuredSecure := os.Getenv("SESSION_COOKIE_SECURE"); configuredSecure != "" {
		secure, err := strconv.ParseBool(configuredSecure)
		if err != nil {
			return nil, nil, errors.New("SESSION_COOKIE_SECURE must be true or false")
		}
		cookieSecure = secure
	}
	allowedOrigin, err := FrontendOrigin(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		return nil, nil, err
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return nil, nil, errors.New("MONGODB_URI must be set in the environment")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("connect MongoDB client: %w", err)
	}
	cleanup := func() error {
		return client.Disconnect(context.Background())
	}

	pingContext, cancelPing := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPing()
	if err := client.Ping(pingContext, nil); err != nil {
		_ = cleanup()
		return nil, nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "go_rest_api"
	}
	database := client.Database(databaseName)
	store := todos.NewStore(database.Collection("todos"))
	sessionDuration := 7 * 24 * time.Hour
	authService := auth.NewService(database.Collection("users"), database.Collection("sessions"), sessionDuration)
	indexContext, cancelIndexes := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelIndexes()
	if err := authService.EnsureIndexes(indexContext); err != nil {
		_ = cleanup()
		return nil, nil, err
	}
	if err := store.EnsureIndexes(indexContext); err != nil {
		_ = cleanup()
		return nil, nil, err
	}

	mux := http.NewServeMux()
	authHandler := auth.NewHandler(authService, cookieSecure, sessionDuration)
	authHandler.RegisterRoutes(mux)
	mux.Handle("/", todos.NewHandler(store, authHandler))
	return auth.CORS(allowedOrigin, mux), cleanup, nil
}

func FrontendOrigin(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "http://localhost:5173", nil
	}
	origin, err := url.Parse(value)
	if err != nil || !origin.IsAbs() || origin.Host == "" ||
		origin.User != nil ||
		(origin.Scheme != "http" && origin.Scheme != "https") ||
		(origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		return "", errors.New("FRONTEND_ORIGIN must be an HTTP or HTTPS origin without a path")
	}
	return origin.Scheme + "://" + origin.Host, nil
}
