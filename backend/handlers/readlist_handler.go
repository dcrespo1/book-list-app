package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	appauth "github.com/dcrespo1/book-list-app/auth"
	"github.com/dcrespo1/book-list-app/pkg/database"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

// BookStore is the persistence interface for the readlist. *database.Queries satisfies it.
type BookStore interface {
	AddBook(ctx context.Context, arg database.AddBookParams) (int32, error)
	GetAllBooks(ctx context.Context, userID string) ([]database.Book, error)
	GetBookByWorkID(ctx context.Context, arg database.GetBookByWorkIDParams) (database.Book, error)
	GetBookByID(ctx context.Context, arg database.GetBookByIDParams) (database.Book, error)
	UpdateBook(ctx context.Context, arg database.UpdateBookParams) (database.Book, error)
	DeleteBookByID(ctx context.Context, arg database.DeleteBookByIDParams) error
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

type ReadlistHandler struct {
	DB      *sql.DB
	Queries BookStore
}

func (h *ReadlistHandler) AddToReadlist(w http.ResponseWriter, r *http.Request) {
	sub, ok := appauth.SubFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "missing user context")
		return
	}

	var input struct {
		Title       string  `json:"title"`
		Authors     string  `json:"authors"`
		Subjects    *string `json:"subjects"`
		Description *string `json:"description"`
		CoverArtURL *string `json:"cover_art_url"`
		WorkID      string  `json:"work_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Title == "" || input.Authors == "" || input.WorkID == "" {
		WriteError(w, http.StatusUnprocessableEntity, "title, authors, and work_id are required")
		return
	}

	id, err := h.Queries.AddBook(r.Context(), database.AddBookParams{
		UserID:      sub,
		Title:       input.Title,
		Authors:     input.Authors,
		Subjects:    toNullString(input.Subjects),
		Description: toNullString(input.Description),
		CoverArtUrl: toNullString(input.CoverArtURL),
		WorkID:      input.WorkID,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			WriteError(w, http.StatusConflict, "book already in readlist")
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to add book")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *ReadlistHandler) GetReadlist(w http.ResponseWriter, r *http.Request) {
	sub, ok := appauth.SubFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "missing user context")
		return
	}

	books, err := h.Queries.GetAllBooks(r.Context(), sub)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to retrieve readlist")
		return
	}
	if books == nil {
		books = []database.Book{}
	}
	WriteJSON(w, http.StatusOK, books)
}

func (h *ReadlistHandler) GetByWorkID(w http.ResponseWriter, r *http.Request) {
	sub, ok := appauth.SubFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "missing user context")
		return
	}

	book, err := h.Queries.GetBookByWorkID(r.Context(), database.GetBookByWorkIDParams{
		WorkID: chi.URLParam(r, "workID"),
		UserID: sub,
	})
	if errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, "book not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to retrieve book")
		return
	}

	WriteJSON(w, http.StatusOK, book)
}

func (h *ReadlistHandler) PatchReadlist(w http.ResponseWriter, r *http.Request) {
	sub, ok := appauth.SubFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "missing user context")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Fetch current state so unset patch fields preserve their existing values.
	current, err := h.Queries.GetBookByID(r.Context(), database.GetBookByIDParams{
		ID:     int32(id),
		UserID: sub,
	})
	if errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, "book not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to retrieve book")
		return
	}

	var input struct {
		Status *string `json:"status"`
		Rating *int32  `json:"rating"`
		Notes  *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Status != nil {
		switch *input.Status {
		case "want_to_read", "reading", "finished", "abandoned":
		default:
			WriteError(w, http.StatusUnprocessableEntity, "status must be one of: want_to_read, reading, finished, abandoned")
			return
		}
	}
	if input.Rating != nil && (*input.Rating < 1 || *input.Rating > 5) {
		WriteError(w, http.StatusUnprocessableEntity, "rating must be between 1 and 5")
		return
	}

	params := database.UpdateBookParams{
		ID:     int32(id),
		UserID: sub,
		Status: current.Status,
		Rating: current.Rating,
		Notes:  current.Notes,
	}
	if input.Status != nil {
		params.Status = *input.Status
	}
	if input.Rating != nil {
		params.Rating = sql.NullInt32{Int32: *input.Rating, Valid: true}
	}
	if input.Notes != nil {
		params.Notes = sql.NullString{String: *input.Notes, Valid: true}
	}

	updated, err := h.Queries.UpdateBook(r.Context(), params)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

func (h *ReadlistHandler) DeleteFromReadlist(w http.ResponseWriter, r *http.Request) {
	sub, ok := appauth.SubFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "missing user context")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.Queries.DeleteBookByID(r.Context(), database.DeleteBookByIDParams{
		ID:     int32(id),
		UserID: sub,
	}); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
