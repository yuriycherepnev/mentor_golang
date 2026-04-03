package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Book struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Author    Author    `json:"author"`
	Year      int       `json:"year"`
	CreatedAt time.Time `json:"created_at"`
}

type Author struct {
	ID   int    `json:"id"`
	Name string `json:"title"`
}

type Library struct {
	mu     sync.RWMutex
	books  map[int]Book
	nextID int
}

// NewLibrary - создает новое хранилище
func NewLibrary() *Library {
	return &Library{
		books:  make(map[int]Book),
		nextID: 1,
	}
}

// Server - HTTP сервер
type Server struct {
	library *Library
}

// Response - стандартный ответ
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	library := NewLibrary()
	server := &Server{library: library}

	// Регистрация роутов
	http.HandleFunc("GET /books", server.GetBooks)
	http.HandleFunc("GET /books/{id}", server.GetBookByID)
	http.HandleFunc("POST /books", server.CreateBook)
	http.HandleFunc("PUT /books/{id}", server.UpdateBook)
	http.HandleFunc("DELETE /books/{id}", server.DeleteBook)

	// Запуск сервера
	println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// GetBooks - получение всех книг
func (s *Server) GetBooks(w http.ResponseWriter, r *http.Request) {
	s.library.mu.RLock()
	defer s.library.mu.RUnlock()

	books := make([]Book, 0, len(s.library.books))
	for _, book := range s.library.books {
		books = append(books, book)
	}

	s.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    books,
	})
}

// GetBookByID - получение книги по ID
func (s *Server) GetBookByID(w http.ResponseWriter, r *http.Request) {
	id, err := s.extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	s.library.mu.RLock()
	defer s.library.mu.RUnlock()

	book, exists := s.library.books[id]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Book not found")
		return
	}

	s.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    book,
	})
}

// CreateBook - создание новой книги
func (s *Server) CreateBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if book.Title == "" || book.Author == "" {
		s.writeError(w, http.StatusBadRequest, "Title and author are required")
		return
	}

	s.library.mu.Lock()
	defer s.library.mu.Unlock()

	book.ID = s.library.nextID
	book.CreatedAt = time.Now()
	s.library.books[book.ID] = book
	s.library.nextID++

	s.writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    book,
	})
}

// UpdateBook - обновление книги
func (s *Server) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := s.extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	var updates Book
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.library.mu.Lock()
	defer s.library.mu.Unlock()

	existing, exists := s.library.books[id]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Book not found")
		return
	}

	// Обновляем только переданные поля
	if updates.Title != "" {
		existing.Title = updates.Title
	}
	if updates.Author != "" {
		existing.Author = updates.Author
	}
	if updates.Year != 0 {
		existing.Year = updates.Year
	}

	s.library.books[id] = existing

	s.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    existing,
	})
}

// DeleteBook - удаление книги
func (s *Server) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := s.extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	s.library.mu.Lock()
	defer s.library.mu.Unlock()

	if _, exists := s.library.books[id]; !exists {
		s.writeError(w, http.StatusNotFound, "Book not found")
		return
	}

	delete(s.library.books, id)

	s.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "Book deleted successfully"},
	})
}

// extractID - извлекает ID из URL path
func (s *Server) extractID(r *http.Request) (int, error) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return 0, http.ErrNoLocation
	}
	return strconv.Atoi(parts[len(parts)-1])
}

// writeJSON - вспомогательная функция для отправки JSON
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError - вспомогательная функция для отправки ошибки
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, Response{
		Success: false,
		Error:   message,
	})
}
