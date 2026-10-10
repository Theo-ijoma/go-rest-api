package todos

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("todo not found")

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type Store struct {
	mu     sync.RWMutex
	nextID int
	todos  []Todo
}

func NewStore() *Store {
	return &Store{nextID: 1}
}

func (s *Store) List() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Todo, len(s.todos))
	copy(result, s.todos)
	return result
}

func (s *Store) Create(title string) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo := Todo{ID: s.nextID, Title: title}
	s.nextID++
	s.todos = append(s.todos, todo)
	return todo
}

func (s *Store) Get(id int) (Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, todo := range s.todos {
		if todo.ID == id {
			return todo, nil
		}
	}
	return Todo{}, ErrNotFound
}

func (s *Store) Update(id int, title string, completed bool) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Title = title
			s.todos[i].Completed = completed
			return s.todos[i], nil
		}
	}
	return Todo{}, ErrNotFound
}

func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, todo := range s.todos {
		if todo.ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
