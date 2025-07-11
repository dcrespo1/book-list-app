package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
)

type BookHandler struct {
	tmpl *template.Template
}

type Book struct {
	ID          int32    `json:"id"`
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
	WorkID      string      `json:"work_id"` 
}
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func NewBookHandler(tmpl *template.Template) BookHandler {
	return BookHandler{tmpl: tmpl}
}

// Index serves the main search form
func (h BookHandler) Index(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "index.html", nil)
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

	log.Printf("Search query: %s", query)
	log.Printf("Found %d books", len(books))

	// Convert []Book → []ViewBook
	viewBooks := make([]ViewBook, 0, len(books))
	for _, b := range books {
		viewBooks = append(viewBooks, ViewBook{
			Title:       b.Title,
			Authors:     b.Authors,
			Subjects:    nil,
			Description: "",
			CoverArtURL: "",
			WorkID:      b.WorkID,
			PublishYear: b.PublishYear,
			ID:          0,
			ShowDeleteButton: false,
  })
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		err := h.tmpl.ExecuteTemplate(w, "partials/search_results.html", viewBooks)
		if err != nil {
			log.Printf("❌ Template error: %v", err)
			http.Error(w, "Template render error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(viewBooks)
}


func (h *BookHandler) DetailsHandler(w http.ResponseWriter, r *http.Request) {
	workID := r.URL.Query().Get("id")
	if workID == "" {
		http.Error(w, "missing work ID", http.StatusBadRequest)
		return
	}

	details, err := h.GetBookDetails(workID)
	if err != nil {
		http.Error(w, "could not fetch book details", http.StatusInternalServerError)
		log.Printf("❌ error loading details for %s: %v", workID, err)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "partials/book_details.html", details)
	if err != nil {
		http.Error(w, "could not render template", http.StatusInternalServerError)
		log.Printf("❌ template execution failed: %v", err)
	}
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

	bookDetails := BookDetails{
		Title:       rawDetails.Title,
		Description: description,
		Subjects:    rawDetails.Subjects,
		CoverArtURL: coverArtURL,
		WorkID:      workID, // ✅ important for HTMX toggle target
	}

	for _, link := range rawDetails.Links {
		bookDetails.Links = append(bookDetails.Links, link.URL)
	}

	return bookDetails, nil
}
