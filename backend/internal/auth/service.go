package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists        = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("email or password is incorrect")
	ErrInvalidSession     = errors.New("session is invalid or expired")
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type User struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	FullName  string        `json:"fullName" bson:"fullName"`
	Email     string        `json:"email" bson:"email"`
	CreatedAt time.Time     `json:"createdAt" bson:"createdAt"`
}

type session struct {
	ID        string    `bson:"_id"`
	User      User      `bson:"user"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

type Service struct {
	users           *mongo.Collection
	sessions        *mongo.Collection
	sessionDuration time.Duration
}

func NewService(users, sessions *mongo.Collection, sessionDuration time.Duration) *Service {
	return &Service{users: users, sessions: sessions, sessionDuration: sessionDuration}
}

func (s *Service) EnsureIndexes(ctx context.Context) error {
	_, err := s.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("email_unique"),
	})
	if err != nil {
		return fmt.Errorf("create unique user email index: %w", err)
	}
	_, err = s.sessions.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0).SetName("session_expiry"),
	})
	if err != nil {
		return fmt.Errorf("create session expiry index: %w", err)
	}
	return nil
}

func (s *Service) Register(ctx context.Context, fullName, email, password string) (User, string, error) {
	fullName = strings.TrimSpace(fullName)
	email = normalizeEmail(email)
	if fullName == "" || len([]rune(fullName)) > 100 {
		return User{}, "", ValidationError{Message: "full name must be between 1 and 100 characters"}
	}
	if !validEmail(email) {
		return User{}, "", ValidationError{Message: "enter a valid email address"}
	}
	if len(password) < 8 || len(password) > 72 {
		return User{}, "", ValidationError{Message: "password must be between 8 and 72 bytes"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", fmt.Errorf("hash password: %w", err)
	}

	user := User{
		ID:        bson.NewObjectID(),
		FullName:  fullName,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}
	document := struct {
		User         `bson:",inline"`
		PasswordHash string `bson:"passwordHash"`
	}{User: user, PasswordHash: string(hash)}

	if _, err := s.users.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return User{}, "", ErrEmailExists
		}
		return User{}, "", fmt.Errorf("create user: %w", err)
	}
	token, err := s.createSession(ctx, user)
	if err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

func (s *Service) SignIn(ctx context.Context, email, password string) (User, string, error) {
	email = normalizeEmail(email)
	var account struct {
		User         `bson:",inline"`
		PasswordHash string `bson:"passwordHash"`
	}
	err := s.users.FindOne(ctx, bson.D{{Key: "email", Value: email}}).Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("find user by email: %w", err)
	}
	if len(password) > 72 || bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return User{}, "", ErrInvalidCredentials
	}
	token, err := s.createSession(ctx, account.User)
	if err != nil {
		return User{}, "", err
	}
	return account.User, token, nil
}

func (s *Service) SessionUser(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrInvalidSession
	}
	var stored session
	err := s.sessions.FindOne(ctx, bson.D{{Key: "_id", Value: tokenDigest(token)}}).Decode(&stored)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return User{}, ErrInvalidSession
	}
	if err != nil {
		return User{}, fmt.Errorf("find session: %w", err)
	}
	if !stored.ExpiresAt.After(time.Now()) {
		return User{}, ErrInvalidSession
	}
	return stored.User, nil
}

func (s *Service) SignOut(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if _, err := s.sessions.DeleteOne(ctx, bson.D{{Key: "_id", Value: tokenDigest(token)}}); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Service) createSession(ctx context.Context, user User) (string, error) {
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	expiresAt := time.Now().UTC().Add(s.sessionDuration)
	if _, err := s.sessions.InsertOne(ctx, session{
		ID:        tokenDigest(token),
		User:      user,
		ExpiresAt: expiresAt,
	}); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

func tokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}
