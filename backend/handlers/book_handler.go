package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type BookHandler struct {
	// baseURL overrides the Open Library base URL. Leave empty for production.
	baseURL string
}

type Book struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	PublishYear int      `json:"first_publish_year"`
	WorkID      string   `json:"work_id"`
}

type BookDetails struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Subjects    []string `json:"subjects"`
	Links       []Link   `json:"links"`
	CoverArtURL string   `json:"cover_art_link"`
}

type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (h *BookHandler) openLibraryURL() string {
	if h.baseURL != "" {
		return h.baseURL
	}
	return "https://openlibrary.org"
}

func (h *BookHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		WriteError(w, http.StatusBadRequest, "missing query parameter 'q'")
		return
	}

	books, err := h.SearchBooks(query)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to search books")
		return
	}

	WriteJSON(w, http.StatusOK, books)
}

func (h *BookHandler) Details(w http.ResponseWriter, r *http.Request) {
	workID := r.URL.Query().Get("id")
	if workID == "" {
		WriteError(w, http.StatusBadRequest, "missing query parameter 'id'")
		return
	}

	details, err := h.GetBookDetails(workID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to fetch book details")
		return
	}

	WriteJSON(w, http.StatusOK, details)
}

func (h *BookHandler) SearchBooks(query string) ([]Book, error) {
	reqURL, err := url.Parse(h.openLibraryURL() + "/search.json")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	reqURL.RawQuery = url.Values{"q": {query}}.Encode()

	resp, err := http.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch books: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		Docs []struct {
			Title       string   `json:"title"`
			AuthorName  []string `json:"author_name"`
			PublishYear int      `json:"first_publish_year"`
			Key         string   `json:"key"`
		} `json:"docs"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	books := make([]Book, 0, len(result.Docs))
	for _, doc := range result.Docs {
		workID := ""
		if len(doc.Key) > 7 {
			workID = doc.Key[7:]
		}
		books = append(books, Book{
			Title:       doc.Title,
			Authors:     doc.AuthorName,
			PublishYear: doc.PublishYear,
			WorkID:      workID,
		})
	}

	return books, nil
}

func (h *BookHandler) GetBookDetails(workID string) (BookDetails, error) {
	reqURL := fmt.Sprintf("%s/works/%s.json", h.openLibraryURL(), workID)

	resp, err := http.Get(reqURL)
	if err != nil {
		return BookDetails{}, fmt.Errorf("failed to fetch book details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BookDetails{}, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BookDetails{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var raw struct {
		Title       string      `json:"title"`
		Description any `json:"description"`
		Subjects    []string    `json:"subjects"`
		Covers      []int       `json:"covers"`
		Links       []Link      `json:"links"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return BookDetails{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var description string
	switch v := raw.Description.(type) {
	case string:
		description = v
	case map[string]any:
		if val, ok := v["value"].(string); ok {
			description = val
		}
	}

	var coverArtURL string
	if len(raw.Covers) > 0 {
		coverArtURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", raw.Covers[0])
	}

	return BookDetails{
		Title:       raw.Title,
		Description: description,
		Subjects:    raw.Subjects,
		Links:       raw.Links,
		CoverArtURL: coverArtURL,
	}, nil
}
