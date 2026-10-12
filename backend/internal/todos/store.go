package todos

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("todo not found")

// Todo is the data shape shared by the HTTP API and the MongoDB collection.
// The JSON and BSON tags tell Go how to name fields in each format.
type Todo struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    bson.ObjectID `json:"-" bson:"userId"`
	Title     string        `json:"title" bson:"title"`
	Completed bool          `json:"completed" bson:"completed"`
}

// Repository describes the database operations handlers need. Interfaces let
// tests substitute a small fake repository without connecting to MongoDB.
type Repository interface {
	List(context.Context, bson.ObjectID) ([]Todo, error)
	Create(context.Context, bson.ObjectID, string) (Todo, error)
	Get(context.Context, bson.ObjectID, bson.ObjectID) (Todo, error)
	Update(context.Context, bson.ObjectID, bson.ObjectID, string, bool) (Todo, error)
	Delete(context.Context, bson.ObjectID, bson.ObjectID) error
}

// Store is the MongoDB-backed implementation of Repository.
type Store struct {
	collection *mongo.Collection
}

// NewStore wraps a MongoDB collection so the handlers can use it as a repository.
func NewStore(collection *mongo.Collection) *Store {
	return &Store{collection: collection}
}

func (s *Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetName("user_id"),
	})
	if err != nil {
		return fmt.Errorf("create todo owner index: %w", err)
	}
	return nil
}

// List reads every todo from MongoDB. The request context lets a client
// disconnect cancel work that is no longer needed.
func (s *Store) List(ctx context.Context, userID bson.ObjectID) ([]Todo, error) {
	cursor, err := s.collection.Find(ctx, bson.D{{Key: "userId", Value: userID}})
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
func (s *Store) Create(ctx context.Context, userID bson.ObjectID, title string) (Todo, error) {
	todo := Todo{
		ID:     bson.NewObjectID(),
		UserID: userID,
		Title:  title,
	}
	if _, err := s.collection.InsertOne(ctx, todo); err != nil {
		return Todo{}, fmt.Errorf("insert todo: %w", err)
	}
	return todo, nil
}

// Get looks up one document by its MongoDB _id field.
func (s *Store) Get(ctx context.Context, userID, id bson.ObjectID) (Todo, error) {
	var todo Todo
	err := s.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}, {Key: "userId", Value: userID}}).Decode(&todo)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Todo{}, ErrNotFound
	}
	if err != nil {
		return Todo{}, fmt.Errorf("find todo: %w", err)
	}
	return todo, nil
}

// Update changes the title and completed status, then returns the saved todo.
func (s *Store) Update(ctx context.Context, userID, id bson.ObjectID, title string, completed bool) (Todo, error) {
	result, err := s.collection.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "userId", Value: userID}},
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
	return s.Get(ctx, userID, id)
}

// Delete removes one document. DeletedCount is zero when the ID does not exist.
func (s *Store) Delete(ctx context.Context, userID, id bson.ObjectID) error {
	result, err := s.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}, {Key: "userId", Value: userID}})
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
