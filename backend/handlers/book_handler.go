package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type BookHandler struct{}

type Book struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	PublishYear int      `json:"first_publish_year"`
	WorkID      string   `json:"work_id"`
}

type BookDetails struct {
	Title       string      `json:"title"`
	Description interface{} `json:"description"`
	Subjects    []string    `json:"subjects"`
	Links       []string    `json:"links"`
	CoverArtURL string      `json:"cover_art_link"`
}
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}


func (h *BookHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	books, err := h.SearchBooks(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func (h *BookHandler) Details(w http.ResponseWriter, r *http.Request) {
	workID := r.URL.Query().Get("id")
	if workID == "" {
		http.Error(w, "Missing query parameter 'id'", http.StatusBadRequest)
		return
	}

	details, err := h.GetBookDetails(workID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}


func (h *BookHandler) SearchBooks(query string) ([]Book, error) {
	baseurl := "https://openlibrary.org/search.json"
	reqURL, err := url.Parse(baseurl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %d", err)
	}

	queryParams := url.Values{}
	queryParams.Set("q", query)
	reqURL.RawQuery = queryParams.Encode()

	resp, err := http.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch books: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %d\n url attempted: %s", resp.StatusCode, reqURL.String())
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
			Key         string   `json:"key"` // This contains "/works/{work_id}"
		} `json:"docs"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	books := []Book{}
	for _, doc := range result.Docs {
		// Extract WorkID by trimming the "/works/" prefix
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
	// Build the correct URL without using url.Values
	url := fmt.Sprintf("https://openlibrary.org/works/%s.json", workID)

	resp, err := http.Get(url)
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

	// Define a struct to capture the raw response
	var rawDetails struct {
		Title       string      `json:"title"`
		Description interface{} `json:"description"`
		Subjects    []string    `json:"subjects"`
		Covers      []int       `json:"covers"`
		Links       []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"links"`
	}

	if err := json.Unmarshal(body, &rawDetails); err != nil {
		return BookDetails{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract description
	var description string
	switch v := rawDetails.Description.(type) {
	case string:
		description = v
	case map[string]interface{}:
		if val, ok := v["value"].(string); ok {
			description = val
		}
	}

	var coverArtURL string
	if len(rawDetails.Covers) > 0 {
		coverArtURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", rawDetails.Covers[0])
	}

	// Build final struct
	bookDetails := BookDetails{
		Title:       rawDetails.Title,
		Description: description,
		Subjects:    rawDetails.Subjects,
		CoverArtURL: coverArtURL,
	}

	for _, link := range rawDetails.Links {
		bookDetails.Links = append(bookDetails.Links, link.URL)
	}

	return bookDetails, nil
}