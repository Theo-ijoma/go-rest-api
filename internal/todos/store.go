package todos

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var ErrNotFound = errors.New("todo not found")

// Todo is the data shape shared by the HTTP API and the MongoDB collection.
// The JSON and BSON tags tell Go how to name fields in each format.
type Todo struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Title     string        `json:"title" bson:"title"`
	Completed bool          `json:"completed" bson:"completed"`
}

// Repository describes the database operations handlers need. Interfaces let
// tests substitute a small fake repository without connecting to MongoDB.
type Repository interface {
	List(context.Context) ([]Todo, error)
	Create(context.Context, string) (Todo, error)
	Get(context.Context, bson.ObjectID) (Todo, error)
	Update(context.Context, bson.ObjectID, string, bool) (Todo, error)
	Delete(context.Context, bson.ObjectID) error
}

// Store is the MongoDB-backed implementation of Repository.
type Store struct {
	collection *mongo.Collection
}

// NewStore wraps a MongoDB collection so the handlers can use it as a repository.
func NewStore(collection *mongo.Collection) *Store {
	return &Store{collection: collection}
}

// List reads every todo from MongoDB. The request context lets a client
// disconnect cancel work that is no longer needed.
func (s *Store) List(ctx context.Context) ([]Todo, error) {
	cursor, err := s.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("find todos: %w", err)
	}
	defer cursor.Close(ctx)

	todos := make([]Todo, 0)
	if err := cursor.All(ctx, &todos); err != nil {
		return nil, fmt.Errorf("decode todos: %w", err)
	}
	return todos, nil
}

// Create inserts a todo. MongoDB ObjectIDs are unique identifiers that remain
// stable across server restarts, unlike an in-memory counter.
func (s *Store) Create(ctx context.Context, title string) (Todo, error) {
	todo := Todo{
		ID:    bson.NewObjectID(),
		Title: title,
	}
	if _, err := s.collection.InsertOne(ctx, todo); err != nil {
		return Todo{}, fmt.Errorf("insert todo: %w", err)
	}
	return todo, nil
}

// Get looks up one document by its MongoDB _id field.
func (s *Store) Get(ctx context.Context, id bson.ObjectID) (Todo, error) {
	var todo Todo
	err := s.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&todo)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Todo{}, ErrNotFound
	}
	if err != nil {
		return Todo{}, fmt.Errorf("find todo: %w", err)
	}
	return todo, nil
}

// Update changes the title and completed status, then returns the saved todo.
func (s *Store) Update(ctx context.Context, id bson.ObjectID, title string, completed bool) (Todo, error) {
	result, err := s.collection.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "title", Value: title},
			{Key: "completed", Value: completed},
		}}},
	)
	if err != nil {
		return Todo{}, fmt.Errorf("update todo: %w", err)
	}
	if result.MatchedCount == 0 {
		return Todo{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete removes one document. DeletedCount is zero when the ID does not exist.
func (s *Store) Delete(ctx context.Context, id bson.ObjectID) error {
	result, err := s.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
