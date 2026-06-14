package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// ========== MODELS ==========

type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Reader struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Book struct {
	ID       int     `json:"id"`
	Title    *string `json:"title,omitempty"` // может быть NULL
	IDAuthor *int    `json:"id_author,omitempty"`
	IDReader *int    `json:"id_reader,omitempty"`
	Author   *Author `json:"author,omitempty"` // для JOIN
	Reader   *Reader `json:"reader,omitempty"` // для JOIN
}

// ========== STORAGE ==========

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// ---------- AUTHOR STORAGE ----------

// ---------- READER STORAGE ----------

func (s *Storage) CreateReader(name string) (*Reader, error) {
	result, err := s.db.Exec("INSERT INTO reader(name) VALUES(?)", name)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Reader{ID: int(id), Name: name}, nil
}

func (s *Storage) GetReaders() ([]Reader, error) {
	rows, err := s.db.Query("SELECT id, name FROM reader ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readers []Reader
	for rows.Next() {
		var r Reader
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		readers = append(readers, r)
	}
	return readers, nil
}

func (s *Storage) GetReaderByID(id int) (*Reader, error) {
	var r Reader
	err := s.db.QueryRow("SELECT id, name FROM reader WHERE id = ?", id).Scan(&r.ID, &r.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Storage) UpdateReader(id int, name string) error {
	result, err := s.db.Exec("UPDATE reader SET name = ? WHERE id = ?", name, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Storage) DeleteReader(id int) error {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM book WHERE id_reader = ?", id).Scan(&count)
	if count > 0 {
		return &ForeignKeyError{"reader has books"}
	}

	result, err := s.db.Exec("DELETE FROM reader WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---------- BOOK STORAGE ----------

// ========== ОШИБКИ ==========

type ForeignKeyError struct {
	Message string
}

func (e *ForeignKeyError) Error() string {
	return e.Message
}

type BusinessError struct {
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

// ========== HTTP СЕРВЕР ==========

type Server struct {
	storage *Storage
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	// Подключение к MySQL
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3307)/library?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	storage := NewStorage(db)
	server := &Server{storage: storage}

	// author
	http.HandleFunc("GET /author", server.GetAuthors)
	http.HandleFunc("POST /author", server.CreateAuthor)
	http.HandleFunc("GET /author/{id}", server.GetAuthorByID)
	http.HandleFunc("PUT /author/{id}", server.UpdateAuthor)
	http.HandleFunc("DELETE /author/{id}", server.DeleteAuthor)

	// reader
	http.HandleFunc("GET /reader", server.GetReaders)
	http.HandleFunc("POST /reader", server.CreateReader)
	http.HandleFunc("GET /reader/{id}", server.GetReaderByID)
	http.HandleFunc("PUT /reader/{id}", server.UpdateReader)
	http.HandleFunc("DELETE /reader/{id}", server.DeleteReader)

	// book
	http.HandleFunc("GET /book", server.GetBooks)
	http.HandleFunc("POST /book", server.CreateBook)
	http.HandleFunc("GET /book/{id}", server.GetBookByID)
	http.HandleFunc("PUT /book/{id}", server.UpdateBook)
	http.HandleFunc("DELETE /book/{id}", server.DeleteBook)

	// functions
	http.HandleFunc("POST /book/borrow", server.BorrowBook)
	http.HandleFunc("POST /book/return", server.ReturnBook)

	log.Println("Server starting on :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

// ========== AUTHOR HANDLERS ==========

// ========== READER HANDLERS ==========

func (s *Server) GetReaders(w http.ResponseWriter, r *http.Request) {
	readers, err := s.storage.GetReaders()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: readers})
}

func (s *Server) CreateReader(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	reader, err := s.storage.CreateReader(req.Name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusCreated, Response{Success: true, Data: reader})
}

func (s *Server) GetReaderByID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	reader, err := s.storage.GetReaderByID(id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if reader == nil {
		s.writeError(w, http.StatusNotFound, "Reader not found")
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: reader})
}

func (s *Server) UpdateReader(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if err := s.storage.UpdateReader(id, req.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "Reader not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"message": "updated"}})
}

func (s *Server) DeleteReader(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := s.storage.DeleteReader(id); err != nil {
		var e *ForeignKeyError
		switch {
		case errors.As(err, &e):
			s.writeError(w, http.StatusConflict, e.Message)
		default:
			if errors.Is(err, sql.ErrNoRows) {
				s.writeError(w, http.StatusNotFound, "Reader not found")
			} else {
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
		}
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"message": "deleted"}})
}

// ========== BOOK HANDLERS ==========

// ========== BUSINESS LOGIC ==========

func (s *Storage) BorrowBook(bookID int, readerID int) error {
	var readerExists bool
	s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM reader WHERE id = ?)", readerID).Scan(&readerExists)
	if !readerExists {
		return &ForeignKeyError{"reader not found"}
	}

	var currentReader sql.NullInt64
	err := s.db.QueryRow("SELECT id_reader FROM book WHERE id = ?", bookID).Scan(&currentReader)
	if err == sql.ErrNoRows {
		return &ForeignKeyError{"book not found"}
	}
	if err != nil {
		return err
	}

	if currentReader.Valid {
		return &BusinessError{"book already borrowed"}
	}

	result, err := s.db.Exec("UPDATE book SET id_reader = ? WHERE id = ?", readerID, bookID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Storage) ReturnBook(bookID int) error {
	var currentReader sql.NullInt64
	err := s.db.QueryRow("SELECT id_reader FROM book WHERE id = ?", bookID).Scan(&currentReader)
	if err == sql.ErrNoRows {
		return &ForeignKeyError{"book not found"}
	}
	if err != nil {
		return err
	}

	if !currentReader.Valid {
		return &BusinessError{"book is not borrowed"}
	}

	result, err := s.db.Exec("UPDATE book SET id_reader = NULL WHERE id = ?", bookID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Server) BorrowBook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BookID   int `json:"book_id"`
		ReaderID int `json:"reader_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BookID == 0 || req.ReaderID == 0 {
		s.writeError(w, http.StatusBadRequest, "book_id and reader_id are required")
		return
	}

	if err := s.storage.BorrowBook(req.BookID, req.ReaderID); err != nil {
		switch e := err.(type) {
		case *ForeignKeyError:
			s.writeError(w, http.StatusNotFound, e.Message)
		case *BusinessError:
			s.writeError(w, http.StatusConflict, e.Message)
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	book, _ := s.storage.GetBookByID(req.BookID)
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

func (s *Server) ReturnBook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BookID int `json:"book_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BookID == 0 {
		s.writeError(w, http.StatusBadRequest, "book_id is required")
		return
	}

	if err := s.storage.ReturnBook(req.BookID); err != nil {
		switch e := err.(type) {
		case *ForeignKeyError:
			s.writeError(w, http.StatusNotFound, e.Message)
		case *BusinessError:
			s.writeError(w, http.StatusConflict, e.Message)
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	book, _ := s.storage.GetBookByID(req.BookID)
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

// ========== ADDITION FUNCTIONS ==========

func extractID(r *http.Request) (int, error) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return 0, http.ErrNoLocation
	}
	return strconv.Atoi(parts[len(parts)-1])
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, Response{Success: false, Error: message})
}
