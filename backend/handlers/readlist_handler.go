package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/dcrespo1/book-list-app/pkg/database"
)

// Helper function to convert *string to sql.NullString
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

type ReadlistHandler struct {
	DB      *sql.DB
	Queries *database.Queries
}

func (h *ReadlistHandler) AddToReadlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Use pointers for nullable fields
	var input struct {
		Title       string  `json:"title"`         // NOT NULL
		Authors     string  `json:"authors"`       // NOT NULL
		Subjects    *string `json:"subjects"`      // Nullable
		Description *string `json:"description"`   // Nullable
		CoverArtURL *string `json:"cover_art_url"` // Nullable
		WorkID      string  `json:"work_id"`       // NOT NULL
	}

	// Decode input JSON
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Use SQLC's generated struct for parameters
	id, err := h.Queries.AddBook(context.Background(), database.AddBookParams{
		Title:       input.Title,                  // Non-nullable string
		Authors:     input.Authors,                // Non-nullable string
		Subjects:    toNullString(input.Subjects), // Nullable
		Description: toNullString(input.Description),
		CoverArtUrl: toNullString(input.CoverArtURL),
		WorkID:      input.WorkID, // Non-nullable string
	})

	if err != nil {
		http.Error(w, "Failed to add book to Readlist", http.StatusInternalServerError)
		return
	}

	// Respond with the created ID
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

func (h *ReadlistHandler) GetReadlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Fetch all books using SQLC
	books, err := h.Queries.GetAllBooks(context.Background())
	if err != nil {
		http.Error(w, "Failed to retrieve Readlist", http.StatusInternalServerError)
		return
	}

	// Respond with the list of books
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}
