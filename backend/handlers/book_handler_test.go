package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestBookServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *BookHandler) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, &BookHandler{baseURL: srv.URL}
}

func TestSearchBooks_ParsesResponse(t *testing.T) {
	_, h := newTestBookServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if q := r.URL.Query().Get("q"); q != "dune" {
			t.Errorf("unexpected query param: %s", q)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"docs": []map[string]any{
				{
					"title":              "Dune",
					"author_name":        []string{"Frank Herbert"},
					"first_publish_year": 1965,
					"key":                "/works/OL12345W",
				},
			},
		})
	})

	books, err := h.SearchBooks("dune")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	b := books[0]
	if b.Title != "Dune" {
		t.Errorf("title: got %q, want %q", b.Title, "Dune")
	}
	if len(b.Authors) != 1 || b.Authors[0] != "Frank Herbert" {
		t.Errorf("authors: got %v", b.Authors)
	}
	if b.PublishYear != 1965 {
		t.Errorf("publish year: got %d, want 1965", b.PublishYear)
	}
	if b.WorkID != "OL12345W" {
		t.Errorf("work_id: got %q, want %q", b.WorkID, "OL12345W")
	}
}

func TestSearchBooks_NonOKResponse(t *testing.T) {
	_, h := newTestBookServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := h.SearchBooks("anything")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestGetBookDetails_StringDescription(t *testing.T) {
	_, h := newTestBookServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"title":       "Dune",
			"description": "A description string",
			"subjects":    []string{"Science Fiction"},
			"covers":      []int{12345},
		})
	})

	d, err := h.GetBookDetails("OL12345W")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Description != "A description string" {
		t.Errorf("description: got %q, want %q", d.Description, "A description string")
	}
	want := "https://covers.openlibrary.org/b/id/12345-L.jpg"
	if d.CoverArtURL != want {
		t.Errorf("cover art URL: got %q, want %q", d.CoverArtURL, want)
	}
}

// Open Library sometimes returns description as {"type":..., "value":"..."} instead of a plain string.
func TestGetBookDetails_ObjectDescription(t *testing.T) {
	_, h := newTestBookServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"title": "Foundation",
			"description": map[string]any{
				"type":  "/type/text",
				"value": "A nested description",
			},
		})
	})

	d, err := h.GetBookDetails("OL99W")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Description != "A nested description" {
		t.Errorf("description: got %q, want %q", d.Description, "A nested description")
	}
}

func TestGetBookDetails_NoCovers(t *testing.T) {
	_, h := newTestBookServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"title": "Book Without Cover",
		})
	})

	d, err := h.GetBookDetails("OL1W")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.CoverArtURL != "" {
		t.Errorf("expected empty cover art URL, got %q", d.CoverArtURL)
	}
}
