package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dcrespo1/book-list-app/pkg/database"
)

type ReadlistHandler struct {
	DB      *sql.DB
	Queries *database.Queries
	Tmpl    *template.Template
}

// Helper functions ---------------------------------------------------------------------
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func joinNullable(s *[]string) *string {
	if s == nil {
		return nil
	}
	joined := strings.Join(*s, ",")
	return &joined
}

func derefOrEmpty(s *[]string) []string {
	if s == nil {
		return []string{}
	}
	return *s
}

func derefOrEmptyString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
// -------------------------------------------------------------------------------------

// HTTP Handlers ---------------------------------------------------------------------
func (h *ReadlistHandler) AddToReadlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	var input struct {
		Title       string    `json:"title"`         // NOT NULL
		Authors     []string  `json:"authors"`       // NOT NULL
		Subjects    *[]string `json:"subjects"`      // Nullable
		Description *string   `json:"description"`   // Nullable
		CoverArtURL *string   `json:"cover_art_url"` // Nullable
		WorkID      string    `json:"work_id"`       // NOT NULL
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate required fields
	missingFields := []string{}
	if input.Title == "" {
		missingFields = append(missingFields, "title")
	}
	if len(input.Authors) == 0 {
		missingFields = append(missingFields, "authors")
	}
	if input.WorkID == "" {
		missingFields = append(missingFields, "work_id")
	}

	if len(missingFields) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":          "Missing required fields",
			"missing_fields": missingFields,
		})
		return
	}

	// Insert into DB
	id, err := h.Queries.AddBook(context.Background(), database.AddBookParams{
	Title:       input.Title,
	Authors:     strings.Join(input.Authors, ","),
	Subjects:    toNullString(joinNullable(input.Subjects)),
	Description: toNullString(input.Description),
	CoverArtUrl: toNullString(input.CoverArtURL),
	WorkID:      input.WorkID,
})
	if err != nil {
		http.Error(w, "Failed to add book to Readlist", http.StatusInternalServerError)
	return
}

	view := ViewBook{
		ID:          id, // this is now correct
		Title:       input.Title,
		Authors:     input.Authors,
		Subjects:    derefOrEmpty(input.Subjects),
		Description: derefOrEmptyString(input.Description),
		CoverArtURL: derefOrEmptyString(input.CoverArtURL),
		WorkID:      input.WorkID,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(view)
}



func (h *ReadlistHandler) GetReadlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	books, err := h.Queries.GetAllBooks(context.Background())
	if err != nil {
		http.Error(w, "Failed to retrieve Readlist", http.StatusInternalServerError)
		return
	}

	viewBooks := make([]ViewBook, 0, len(books))
	for _, b := range books {
		viewBooks = append(viewBooks, ViewBook{
			ID:          b.ID,
			Title:       b.Title,
			Authors:     splitAndTrim(b.Authors),
			Subjects:    splitAndTrim(b.Subjects.String),
			Description: b.Description.String,
			CoverArtURL: b.CoverArtUrl.String,
			WorkID:      b.WorkID,
			ShowDeleteButton: true, // Always show delete button in Readlist
		})
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := h.Tmpl.ExecuteTemplate(w, "partials/book_list.html", viewBooks); err != nil {
			log.Printf("❌ Template error: %v", err)
			http.Error(w, "Failed to render book list", http.StatusInternalServerError)
			return
		}
		return //this can be commented out if you want to return JSON as well
	}

	// fallback: JSON API
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(viewBooks)
}

func (h *ReadlistHandler) DeleteFromReadlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from query param
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	// Parse ID to int32
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid 'id' parameter", http.StatusBadRequest)
		return
	}

	// Delete book from DB
	err = h.Queries.DeleteBookByID(context.Background(), int32(id))
	if err != nil {
		http.Error(w, "Failed to delete book", http.StatusInternalServerError)
		return
	}

	// Respond success
	
	w.WriteHeader(http.StatusNoContent)
}

// -------------------------------------------------------------------------------------
