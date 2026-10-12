package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"example.com/go-rest-api/internal/auth"
	"example.com/go-rest-api/internal/todos"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run returns errors to main so deferred cleanup runs before the process exits.
func run() error {
	// Go does not load .env files automatically. Existing environment variables
	// still work, and godotenv does not overwrite values already set by the OS.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	var cookieSecure bool
	if configuredSecure := os.Getenv("SESSION_COOKIE_SECURE"); configuredSecure != "" {
		secure, err := strconv.ParseBool(configuredSecure)
		if err != nil {
			return errors.New("SESSION_COOKIE_SECURE must be true or false")
		}
		cookieSecure = secure
	}
	allowedOrigin, err := frontendOrigin(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		return err
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return errors.New("MONGODB_URI must be set in .env or the environment")
	}

	// Create one MongoDB client and reuse its built-in connection pool.
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("failed to disconnect MongoDB client: %v", err)
		}
	}()

	// Ping once at startup so a bad URI or unreachable cluster fails clearly.
	pingContext, cancelPing := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPing()
	if err := client.Ping(pingContext, nil); err != nil {
		return err
	}

	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "go_rest_api"
	}
	store := todos.NewStore(client.Database(databaseName).Collection("todos"))
	sessionDuration := 7 * 24 * time.Hour
	authService := auth.NewService(
		client.Database(databaseName).Collection("users"),
		client.Database(databaseName).Collection("sessions"),
		sessionDuration,
	)
	indexContext, cancelIndexes := context.WithTimeout(context.Background(), 10*time.Second)
	if err := authService.EnsureIndexes(indexContext); err != nil {
		cancelIndexes()
		return err
	}
	if err := store.EnsureIndexes(indexContext); err != nil {
		cancelIndexes()
		return err
	}
	cancelIndexes()

	mux := http.NewServeMux()
	authHandler := auth.NewHandler(authService, cookieSecure, sessionDuration)
	authHandler.RegisterRoutes(mux)
	mux.Handle("/", todos.NewHandler(store, authHandler))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           auth.CORS(allowedOrigin, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API listening on http://localhost:%s (MongoDB database: %s)", port, databaseName)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func frontendOrigin(value string) (string, error) {
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
