package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/dcrespo1/book-list-app/handlers"
	"github.com/dcrespo1/book-list-app/pkg/database" // Update with your actual module path

	"github.com/dcrespo1/book-list-app/api"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var db_user = os.Getenv("POSTGRES_USER")
var db_password = os.Getenv("POSTGRES_PASSWORD")
var db_name = os.Getenv("POSTGRES_DB")
var db_port = os.Getenv("POSTGRES_PORT")

func main() {
	// PostgreSQL connection
	connStr := "postgres://" + db_user + ":" + db_password + "@localhost:" + db_port + "/" + db_name + "?sslmode=disable" // Update with your DB details
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Initialize sqlc queries
	dbQueries := database.New(db)

	// Initialize handlers
	ReadlistHandler := handlers.ReadlistHandler{
		DB:      db,
		Queries: dbQueries,
	}

	// HTTP routes
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
			return
		}

		books, err := api.SearchBooks(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(books)
	})

	http.HandleFunc("/details", func(w http.ResponseWriter, r *http.Request) {
		workID := r.URL.Query().Get("id")
		if workID == "" {
			http.Error(w, "Missing query parameter 'id'", http.StatusBadRequest)
			return
		}

		details, err := api.GetBookDetails(workID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(details)
	})

	http.HandleFunc("/readlist/add", ReadlistHandler.AddToReadlist)
	http.HandleFunc("/readlist", ReadlistHandler.GetReadlist)

	// Start server
	log.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
