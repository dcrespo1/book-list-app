package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/dcrespo1/book-list-app/handlers"
	"github.com/dcrespo1/book-list-app/pkg/database"

	_ "github.com/lib/pq"
)

var db_user = os.Getenv("POSTGRES_USER")
var db_password = os.Getenv("POSTGRES_PASSWORD")
var db_name = os.Getenv("POSTGRES_DB")
var db_port = os.Getenv("POSTGRES_PORT")

func main() {
	// PostgreSQL connection
	connStr := "postgres://" + db_user + ":" + db_password + "@localhost:" + db_port + "/" + db_name + "?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	// Parse all templates
	tmpl := template.Must(template.New("").ParseFiles(
		"templates/layout.html",
		"templates/index.html",
		"templates/partials/book_card.html",
		"templates/partials/book_details.html",
		"templates/partials/book_list.html",
		"templates/partials/search_results.html",
	))
	// Initialize handlers
	readlistHandler := handlers.ReadlistHandler{
		DB:      db,
		Queries: dbQueries,
		Tmpl:    tmpl,
	}

	bookHandler := handlers.NewBookHandler(tmpl)

	// Static assets (e.g., htmx.min.js)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Frontend HTML routes
	http.HandleFunc("/", bookHandler.Index)
	http.HandleFunc("/search", bookHandler.Search)

	// API routes
	http.HandleFunc("/details", bookHandler.DetailsHandler)
	http.HandleFunc("/readlist", readlistHandler.GetReadlist)
	http.HandleFunc("/readlist/add", readlistHandler.AddToReadlist)
	http.HandleFunc("/readlist/delete", readlistHandler.DeleteFromReadlist)


	log.Println("🌐 Server is running on http://localhost:8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
