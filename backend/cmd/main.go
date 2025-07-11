package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/dcrespo1/book-list-app/handlers"
	"github.com/dcrespo1/book-list-app/pkg/database" // Update with your actual module path

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

	BookHandler := handlers.BookHandler{}

	// HTTP routes
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Book List App API!"))
	})

	http.HandleFunc("/search", BookHandler.Search)
	http.HandleFunc("/details", BookHandler.Details)

	http.HandleFunc("/readlist/add", ReadlistHandler.AddToReadlist)
	http.HandleFunc("/readlist", ReadlistHandler.GetReadlist)

	// Start server
	log.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
