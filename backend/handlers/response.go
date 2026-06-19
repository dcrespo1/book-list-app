package handlers

import "github.com/dcrespo1/book-list-app/pkg/database"

type BookResponse struct {
	ID          int32   `json:"id"`
	Title       string  `json:"title"`
	Authors     string  `json:"authors"`
	Subjects    *string `json:"subjects"`
	Description *string `json:"description"`
	CoverArtURL *string `json:"cover_art_url"`
	WorkID      string  `json:"work_id"`
	Status      string  `json:"status"`
	Rating      *int32  `json:"rating"`
	Notes       *string `json:"notes"`
}

func toBookResponse(b database.Book) BookResponse {
	r := BookResponse{
		ID:      b.ID,
		Title:   b.Title,
		Authors: b.Authors,
		WorkID:  b.WorkID,
		Status:  b.Status,
	}
	if b.Subjects.Valid {
		r.Subjects = &b.Subjects.String
	}
	if b.Description.Valid {
		r.Description = &b.Description.String
	}
	if b.CoverArtUrl.Valid {
		r.CoverArtURL = &b.CoverArtUrl.String
	}
	if b.Rating.Valid {
		r.Rating = &b.Rating.Int32
	}
	if b.Notes.Valid {
		r.Notes = &b.Notes.String
	}
	return r
}
