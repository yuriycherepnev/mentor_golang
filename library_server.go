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

// ========== МОДЕЛИ ==========

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

// ========== ХРАНИЛИЩЕ ==========

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// ---------- AUTHORS ----------

func (s *Storage) CreateAuthor(name string) (*Author, error) {
	result, err := s.db.Exec("INSERT INTO author(name) VALUES(?)", name)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Author{ID: int(id), Name: name}, nil
}

func (s *Storage) GetAuthors() ([]Author, error) {
	rows, err := s.db.Query("SELECT id, name FROM author ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}
	return authors, nil
}

func (s *Storage) GetAuthorByID(id int) (*Author, error) {
	var a Author
	err := s.db.QueryRow("SELECT id, name FROM author WHERE id = ?", id).Scan(&a.ID, &a.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Storage) UpdateAuthor(id int, name string) error {
	result, err := s.db.Exec("UPDATE author SET name = ? WHERE id = ?", name, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Storage) DeleteAuthor(id int) error {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM book WHERE id_author = ?", id).Scan(&count)
	if count > 0 {
		return &ForeignKeyError{"author has books"}
	}

	result, err := s.db.Exec("DELETE FROM author WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---------- READERS ----------
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

// ---------- BOOKS ----------
func (s *Storage) CreateBook(uniqueID string, title *string, idAuthor *int) (*Book, error) {
	result, err := s.db.Exec(
		"INSERT INTO book(unique_id, title, id_author, id_reader) VALUES(?, ?, ?, NULL)",
		uniqueID, title, idAuthor,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.GetBookByID(int(id))
}

func (s *Storage) GetBooks() ([]Book, error) {
	query := `
        SELECT b.id, b.unique_id, b.title, b.id_author, b.id_reader,
               a.id, a.name,
               r.id, r.name
        FROM book b
        LEFT JOIN author a ON b.id_author = a.id
        LEFT JOIN reader r ON b.id_reader = r.id
        ORDER BY b.id
    `
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		var aID, _ sql.NullInt64
		var aNameStr sql.NullString
		var rID, _ sql.NullInt64
		var rNameStr sql.NullString

		err := rows.Scan(
			&b.ID, &b.Title, &b.IDAuthor, &b.IDReader,
			&aID, &aNameStr,
			&rID, &rNameStr,
		)
		if err != nil {
			return nil, err
		}

		if aID.Valid {
			b.Author = &Author{ID: int(aID.Int64), Name: aNameStr.String}
		}

		if rID.Valid {
			b.Reader = &Reader{ID: int(rID.Int64), Name: rNameStr.String}
		}

		books = append(books, b)
	}
	return books, nil
}

func (s *Storage) GetBookByID(id int) (*Book, error) {
	query := `
        SELECT b.id, b.unique_id, b.title, b.id_author, b.id_reader,
               a.id, a.name,
               r.id, r.name
        FROM book b
        LEFT JOIN author a ON b.id_author = a.id
        LEFT JOIN reader r ON b.id_reader = r.id
        WHERE b.id = ?
    `
	var b Book
	var aID, _ sql.NullInt64
	var aNameStr sql.NullString
	var rID, _ sql.NullInt64
	var rNameStr sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&b.ID, &b.Title, &b.IDAuthor, &b.IDReader,
		&aID, &aNameStr,
		&rID, &rNameStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if aID.Valid {
		b.Author = &Author{ID: int(aID.Int64), Name: aNameStr.String}
	}
	if rID.Valid {
		b.Reader = &Reader{ID: int(rID.Int64), Name: rNameStr.String}
	}

	return &b, nil
}

func (s *Storage) GetBookByUniqueID(uniqueID string) (*Book, error) {
	query := `
        SELECT b.id, b.unique_id, b.title, b.id_author, b.id_reader,
               a.id, a.name,
               r.id, r.name
        FROM book b
        LEFT JOIN author a ON b.id_author = a.id
        LEFT JOIN reader r ON b.id_reader = r.id
        WHERE b.unique_id = ?
    `
	var b Book
	var aID, _ sql.NullInt64
	var aNameStr sql.NullString
	var rID, _ sql.NullInt64
	var rNameStr sql.NullString

	err := s.db.QueryRow(query, uniqueID).Scan(
		&b.ID, &b.Title, &b.IDAuthor, &b.IDReader,
		&aID, &aNameStr,
		&rID, &rNameStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if aID.Valid {
		b.Author = &Author{ID: int(aID.Int64), Name: aNameStr.String}
	}
	if rID.Valid {
		b.Reader = &Reader{ID: int(rID.Int64), Name: rNameStr.String}
	}

	return &b, nil
}

func (s *Storage) UpdateBook(id int, title *string, idAuthor *int) error {
	var query string
	var args []interface{}

	if title != nil && idAuthor != nil {
		query = "UPDATE book SET title = ?, id_author = ? WHERE id = ?"
		args = []interface{}{title, idAuthor, id}
	} else if title != nil {
		query = "UPDATE book SET title = ? WHERE id = ?"
		args = []interface{}{title, id}
	} else if idAuthor != nil {
		query = "UPDATE book SET id_author = ? WHERE id = ?"
		args = []interface{}{idAuthor, id}
	} else {
		return nil
	}

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Storage) DeleteBook(id int) error {
	result, err := s.db.Exec("DELETE FROM book WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---------- БИЗНЕС-ОПЕРАЦИИ ----------
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

	// authors
	http.HandleFunc("GET /authors", server.GetAuthors)
	http.HandleFunc("POST /authors", server.CreateAuthor)
	http.HandleFunc("GET /authors/{id}", server.GetAuthorByID)
	http.HandleFunc("PUT /authors/{id}", server.UpdateAuthor)
	http.HandleFunc("DELETE /authors/{id}", server.DeleteAuthor)

	// readers
	http.HandleFunc("GET /readers", server.GetReaders)
	http.HandleFunc("POST /readers", server.CreateReader)
	http.HandleFunc("GET /readers/{id}", server.GetReaderByID)
	http.HandleFunc("PUT /readers/{id}", server.UpdateReader)
	http.HandleFunc("DELETE /readers/{id}", server.DeleteReader)

	// books
	http.HandleFunc("GET /books", server.GetBooks)
	http.HandleFunc("POST /books", server.CreateBook)
	http.HandleFunc("GET /books/{id}", server.GetBookByID)
	http.HandleFunc("PUT /books/{id}", server.UpdateBook)
	http.HandleFunc("DELETE /books/{id}", server.DeleteBook)

	// functions
	http.HandleFunc("POST /books/{id}/borrow", server.BorrowBook)
	http.HandleFunc("POST /books/{id}/return", server.ReturnBook)

	log.Println("Server starting on :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

// ========== ХЕНДЛЕРЫ АВТОРОВ ==========

func (s *Server) GetAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := s.storage.GetAuthors()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: authors})
}

func (s *Server) CreateAuthor(w http.ResponseWriter, r *http.Request) {
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

	author, err := s.storage.CreateAuthor(req.Name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusCreated, Response{Success: true, Data: author})
}

func (s *Server) GetAuthorByID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	author, err := s.storage.GetAuthorByID(id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if author == nil {
		s.writeError(w, http.StatusNotFound, "Author not found")
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: author})
}

func (s *Server) UpdateAuthor(w http.ResponseWriter, r *http.Request) {
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

	if err := s.storage.UpdateAuthor(id, req.Name); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "Author not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"message": "updated"}})
}

func (s *Server) DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := s.storage.DeleteAuthor(id); err != nil {
		var e *ForeignKeyError
		switch {
		case errors.As(err, &e):
			s.writeError(w, http.StatusConflict, e.Message)
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"message": "deleted"}})
}

// ========== ХЕНДЛЕРЫ ЧИТАТЕЛЕЙ ==========

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

// ========== ХЕНДЛЕРЫ КНИГ ==========

func (s *Server) GetBooks(w http.ResponseWriter, r *http.Request) {
	books, err := s.storage.GetBooks()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: books})
}

func (s *Server) CreateBook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UniqueID string  `json:"unique_id"`
		Title    *string `json:"title,omitempty"`
		IDAuthor *int    `json:"id_author,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.UniqueID == "" {
		s.writeError(w, http.StatusBadRequest, "unique_id is required")
		return
	}

	// Проверяем уникальность unique_id
	existing, _ := s.storage.GetBookByUniqueID(req.UniqueID)
	if existing != nil {
		s.writeError(w, http.StatusConflict, "unique_id already exists")
		return
	}

	book, err := s.storage.CreateBook(req.UniqueID, req.Title, req.IDAuthor)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusCreated, Response{Success: true, Data: book})
}

func (s *Server) GetBookByID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	book, err := s.storage.GetBookByID(id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if book == nil {
		s.writeError(w, http.StatusNotFound, "Book not found")
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

func (s *Server) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Title    *string `json:"title,omitempty"`
		IDAuthor *int    `json:"id_author,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.storage.UpdateBook(id, req.Title, req.IDAuthor); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "Book not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	book, _ := s.storage.GetBookByID(id)
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

func (s *Server) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := s.storage.DeleteBook(id); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "Book not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"message": "deleted"}})
}

// ========== БИЗНЕС-ОПЕРАЦИИ ==========

func (s *Server) BorrowBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	var req struct {
		ReaderID int `json:"reader_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReaderID == 0 {
		s.writeError(w, http.StatusBadRequest, "reader_id is required")
		return
	}

	if err := s.storage.BorrowBook(bookID, req.ReaderID); err != nil {
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

	book, _ := s.storage.GetBookByID(bookID)
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

func (s *Server) ReturnBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := extractID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	if err := s.storage.ReturnBook(bookID); err != nil {
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

	book, _ := s.storage.GetBookByID(bookID)
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

// ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ==========

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
