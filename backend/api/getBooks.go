package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Book represents a simplified structure for book data.
// Book represents a simplified structure for book data.
type Book struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	PublishYear []int    `json:"publish_year"`
	Subjects    []string `json:"subject"`
	WorkID      string   `json:"work_id"`
}

// SearchBooks searches for books by title or author using Open Library's Search API.
func SearchBooks(query string) ([]Book, error) {
	url := fmt.Sprintf("https://openlibrary.org/search.json?q=%s", query)

	resp, err := http.Get(url)
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
			PublishYear []int    `json:"publish_year"`
			Subject     []string `json:"subject"`
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
			Subjects:    doc.Subject,
			WorkID:      workID,
		})
	}

	return books, nil
}

type BookDetails struct {
	Title       string      `json:"title"`
	Description interface{} `json:"description"`
	Subjects    []string    `json:"subjects"`
	Links       []string    `json:"links"`
	CoverArtURL string      `json:"cover_art_link"`
}

// Link represents a structured link in the book details.
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// GetBookDetails fetches detailed information about a book by its Work ID.
func GetBookDetails(workID string) (BookDetails, error) {
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

	// Extract description (if it might be nested or a string)
	var description string
	switch v := rawDetails.Description.(type) {
	case string:
		description = v
	case map[string]interface{}:
		if val, ok := v["value"].(string); ok {
			description = val
		}
	}

	// Construct the cover art URL
	var coverArtURL string
	if len(rawDetails.Covers) > 0 {
		coverArtURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", rawDetails.Covers[0])
	}

	// Map the raw data to the BookDetails struct
	bookDetails := BookDetails{
		Title:       rawDetails.Title,
		Description: description,
		Subjects:    rawDetails.Subjects,
		CoverArtURL: coverArtURL,
	}

	// Map links
	for _, link := range rawDetails.Links {
		bookDetails.Links = append(bookDetails.Links, link.URL)
	}

	return bookDetails, nil
}
