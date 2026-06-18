package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	appauth "github.com/dcrespo1/book-list-app/auth"
	"github.com/dcrespo1/book-list-app/pkg/database"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

// fakeStore is a test double for BookStore.
type fakeStore struct {
	books       []database.Book
	addedID     int32
	addErr      error
	getAllErr    error
	getOneErr   error
	getIDErr    error
	updateErr   error
	deleteErr   error
	updatedBook database.Book
}

func (f *fakeStore) AddBook(_ context.Context, _ database.AddBookParams) (int32, error) {
	return f.addedID, f.addErr
}

func (f *fakeStore) GetAllBooks(_ context.Context, _ string) ([]database.Book, error) {
	return f.books, f.getAllErr
}

func (f *fakeStore) GetBookByWorkID(_ context.Context, arg database.GetBookByWorkIDParams) (database.Book, error) {
	if f.getOneErr != nil {
		return database.Book{}, f.getOneErr
	}
	for _, b := range f.books {
		if b.WorkID == arg.WorkID && b.UserID == arg.UserID {
			return b, nil
		}
	}
	return database.Book{}, sql.ErrNoRows
}

func (f *fakeStore) GetBookByID(_ context.Context, arg database.GetBookByIDParams) (database.Book, error) {
	if f.getIDErr != nil {
		return database.Book{}, f.getIDErr
	}
	for _, b := range f.books {
		if b.ID == arg.ID && b.UserID == arg.UserID {
			return b, nil
		}
	}
	return database.Book{}, sql.ErrNoRows
}

func (f *fakeStore) UpdateBook(_ context.Context, _ database.UpdateBookParams) (database.Book, error) {
	return f.updatedBook, f.updateErr
}

func (f *fakeStore) DeleteBookByID(_ context.Context, _ database.DeleteBookByIDParams) error {
	return f.deleteErr
}

func newHandler(store BookStore) *ReadlistHandler {
	return &ReadlistHandler{Queries: store}
}

// withChiParam injects chi URL params into the request context.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withSub injects an authenticated user's subject claim into the request context.
func withSub(r *http.Request, sub string) *http.Request {
	return r.WithContext(appauth.SubToContext(r.Context(), sub))
}

const testSub = "user-sub-abc123"

// --- AddToReadlist ---

func TestAddToReadlist_Success(t *testing.T) {
	h := newHandler(&fakeStore{addedID: 42})

	body := `{"title":"Dune","authors":"Frank Herbert","work_id":"OL12345W"}`
	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodPost, "/readlist", bytes.NewBufferString(body)), testSub)
	h.AddToReadlist(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusCreated)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"] != float64(42) {
		t.Errorf("id: got %v, want 42", resp["id"])
	}
}

func TestAddToReadlist_InvalidJSON(t *testing.T) {
	h := newHandler(&fakeStore{})

	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodPost, "/readlist", bytes.NewBufferString("not json")), testSub)
	h.AddToReadlist(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAddToReadlist_MissingRequiredFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing title", `{"authors":"Frank Herbert","work_id":"OL12345W"}`},
		{"missing authors", `{"title":"Dune","work_id":"OL12345W"}`},
		{"missing work_id", `{"title":"Dune","authors":"Frank Herbert"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(&fakeStore{})
			w := httptest.NewRecorder()
			r := withSub(httptest.NewRequest(http.MethodPost, "/readlist", bytes.NewBufferString(tc.body)), testSub)
			h.AddToReadlist(w, r)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
			}
		})
	}
}

func TestAddToReadlist_Duplicate(t *testing.T) {
	h := newHandler(&fakeStore{addErr: &pq.Error{Code: "23505"}})

	body := `{"title":"Dune","authors":"Frank Herbert","work_id":"OL12345W"}`
	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodPost, "/readlist", bytes.NewBufferString(body)), testSub)
	h.AddToReadlist(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestAddToReadlist_DBError(t *testing.T) {
	h := newHandler(&fakeStore{addErr: errors.New("connection lost")})

	body := `{"title":"Dune","authors":"Frank Herbert","work_id":"OL12345W"}`
	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodPost, "/readlist", bytes.NewBufferString(body)), testSub)
	h.AddToReadlist(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- GetReadlist ---

func TestGetReadlist_ReturnsSavedBooks(t *testing.T) {
	books := []database.Book{
		{ID: 1, Title: "Dune", Authors: "Frank Herbert", WorkID: "OL12345W", UserID: testSub, Status: "want_to_read"},
	}
	h := newHandler(&fakeStore{books: books})

	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodGet, "/readlist", nil), testSub)
	h.GetReadlist(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	var got []database.Book
	json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 1 || got[0].Title != "Dune" {
		t.Errorf("unexpected books: %v", got)
	}
}

// Verifies that an empty readlist returns [] (JSON array), not null.
func TestGetReadlist_EmptyReturnsArray(t *testing.T) {
	h := newHandler(&fakeStore{})

	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodGet, "/readlist", nil), testSub)
	h.GetReadlist(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	var got []database.Book
	json.NewDecoder(w.Body).Decode(&got)
	if got == nil {
		t.Error("expected empty JSON array [], got null")
	}
}

func TestGetReadlist_DBError(t *testing.T) {
	h := newHandler(&fakeStore{getAllErr: errors.New("db down")})

	w := httptest.NewRecorder()
	r := withSub(httptest.NewRequest(http.MethodGet, "/readlist", nil), testSub)
	h.GetReadlist(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- GetByWorkID ---

func TestGetByWorkID_Found(t *testing.T) {
	books := []database.Book{
		{ID: 1, Title: "Dune", Authors: "Frank Herbert", WorkID: "OL12345W", UserID: testSub, Status: "want_to_read"},
	}
	h := newHandler(&fakeStore{books: books})

	w := httptest.NewRecorder()
	r := withSub(withChiParam(httptest.NewRequest(http.MethodGet, "/readlist/OL12345W", nil), "workID", "OL12345W"), testSub)
	h.GetByWorkID(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	var got database.Book
	json.NewDecoder(w.Body).Decode(&got)
	if got.WorkID != "OL12345W" {
		t.Errorf("work_id: got %q, want %q", got.WorkID, "OL12345W")
	}
}

func TestGetByWorkID_NotFound(t *testing.T) {
	h := newHandler(&fakeStore{})

	w := httptest.NewRecorder()
	r := withSub(withChiParam(httptest.NewRequest(http.MethodGet, "/readlist/OL99999W", nil), "workID", "OL99999W"), testSub)
	h.GetByWorkID(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetByWorkID_DBError(t *testing.T) {
	h := newHandler(&fakeStore{getOneErr: errors.New("db down")})

	w := httptest.NewRecorder()
	r := withSub(withChiParam(httptest.NewRequest(http.MethodGet, "/readlist/OL12345W", nil), "workID", "OL12345W"), testSub)
	h.GetByWorkID(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- PatchReadlist ---

func seedBook() []database.Book {
	return []database.Book{
		{ID: 1, Title: "Dune", Authors: "Frank Herbert", WorkID: "OL12345W", UserID: testSub, Status: "want_to_read"},
	}
}

func patchRequest(id, body string) *http.Request {
	return withSub(
		withChiParam(httptest.NewRequest(http.MethodPatch, "/readlist/"+id, bytes.NewBufferString(body)), "id", id),
		testSub,
	)
}

func TestPatchReadlist_UpdateStatus(t *testing.T) {
	updated := database.Book{ID: 1, Status: "reading", UserID: testSub}
	h := newHandler(&fakeStore{books: seedBook(), updatedBook: updated})

	w := httptest.NewRecorder()
	h.PatchReadlist(w, patchRequest("1", `{"status":"reading"}`))

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	var got database.Book
	json.NewDecoder(w.Body).Decode(&got)
	if got.Status != "reading" {
		t.Errorf("status field: got %q, want %q", got.Status, "reading")
	}
}

func TestPatchReadlist_UpdateRating(t *testing.T) {
	updated := database.Book{ID: 1, Status: "want_to_read", Rating: sql.NullInt32{Int32: 5, Valid: true}, UserID: testSub}
	h := newHandler(&fakeStore{books: seedBook(), updatedBook: updated})

	w := httptest.NewRecorder()
	h.PatchReadlist(w, patchRequest("1", `{"rating":5}`))

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPatchReadlist_UpdateNotes(t *testing.T) {
	updated := database.Book{ID: 1, Status: "want_to_read", Notes: sql.NullString{String: "great book", Valid: true}, UserID: testSub}
	h := newHandler(&fakeStore{books: seedBook(), updatedBook: updated})

	w := httptest.NewRecorder()
	h.PatchReadlist(w, patchRequest("1", `{"notes":"great book"}`))

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPatchReadlist_InvalidStatus(t *testing.T) {
	h := newHandler(&fakeStore{books: seedBook()})

	w := httptest.NewRecorder()
	h.PatchReadlist(w, patchRequest("1", `{"status":"binge_read"}`))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestPatchReadlist_InvalidRating(t *testing.T) {
	cases := []int{0, 6}
	for _, rating := range cases {
		t.Run(fmt.Sprintf("rating_%d", rating), func(t *testing.T) {
			h := newHandler(&fakeStore{books: seedBook()})
			w := httptest.NewRecorder()
			h.PatchReadlist(w, patchRequest("1", fmt.Sprintf(`{"rating":%d}`, rating)))

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
			}
		})
	}
}

func TestPatchReadlist_NotFound(t *testing.T) {
	h := newHandler(&fakeStore{}) // empty store

	w := httptest.NewRecorder()
	h.PatchReadlist(w, patchRequest("99", `{"status":"reading"}`))

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPatchReadlist_DBError(t *testing.T) {
	h := newHandler(&fakeStore{books: seedBook(), updateErr: errors.New("db down")})

	w := httptest.NewRecorder()
	h.PatchReadlist(w, patchRequest("1", `{"status":"reading"}`))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- DeleteFromReadlist ---

func TestDeleteFromReadlist_Success(t *testing.T) {
	h := newHandler(&fakeStore{})

	w := httptest.NewRecorder()
	r := withSub(withChiParam(httptest.NewRequest(http.MethodDelete, "/readlist/1", nil), "id", "1"), testSub)
	h.DeleteFromReadlist(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteFromReadlist_InvalidID(t *testing.T) {
	h := newHandler(&fakeStore{})

	w := httptest.NewRecorder()
	r := withSub(withChiParam(httptest.NewRequest(http.MethodDelete, "/readlist/abc", nil), "id", "abc"), testSub)
	h.DeleteFromReadlist(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteFromReadlist_DBError(t *testing.T) {
	h := newHandler(&fakeStore{deleteErr: errors.New("db down")})

	w := httptest.NewRecorder()
	r := withSub(withChiParam(httptest.NewRequest(http.MethodDelete, "/readlist/1", nil), "id", "1"), testSub)
	h.DeleteFromReadlist(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
