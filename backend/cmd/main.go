package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appauth "github.com/dcrespo1/book-list-app/auth"
	"github.com/dcrespo1/book-list-app/handlers"
	"github.com/dcrespo1/book-list-app/pkg/database"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/coreos/go-oidc/v3/oidc"
	_ "github.com/lib/pq"
)

type config struct {
	dbURL                string
	port                 string
	keycloakIssuer       string // iss claim in tokens; what clients (Bruno/browser) use
	keycloakDiscoveryURL string // where the server fetches OIDC config (differs inside devcontainer)
}

func loadConfig() config {
	issuer := getEnv("KEYCLOAK_ISSUER", "http://localhost:8180/realms/booklist")
	return config{
		dbURL: "postgres://" +
			getEnv("POSTGRES_USER", "app") + ":" +
			getEnv("POSTGRES_PASSWORD", "app") + "@" +
			getEnv("POSTGRES_HOST", "localhost") + ":" +
			getEnv("POSTGRES_PORT", "5432") + "/" +
			getEnv("POSTGRES_DB", "app") + "?sslmode=disable",
		port:                 getEnv("PORT", "8080"),
		keycloakIssuer:       issuer,
		keycloakDiscoveryURL: getEnv("KEYCLOAK_DISCOVERY_URL", issuer),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := loadConfig()

	// --- Database ---
	db, err := sql.Open("postgres", cfg.dbURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// --- OIDC / Keycloak ---
	// KEYCLOAK_DISCOVERY_URL may differ from KEYCLOAK_ISSUER when the server runs inside
	// the devcontainer network and needs to reach Keycloak by service name (keycloak:8080)
	// while tokens carry the public-facing issuer (localhost:8180).
	oidcCtx := context.Background()
	if cfg.keycloakDiscoveryURL != cfg.keycloakIssuer {
		oidcCtx = oidc.InsecureIssuerURLContext(oidcCtx, cfg.keycloakDiscoveryURL)
	}

	var provider *oidc.Provider
	for i := range 10 {
		provider, err = oidc.NewProvider(oidcCtx, cfg.keycloakIssuer)
		if err == nil {
			break
		}
		slog.Warn("Keycloak not ready, retrying...", "attempt", i+1, "error", err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		slog.Error("failed to connect to Keycloak", "issuer", cfg.keycloakIssuer, "error", err)
		os.Exit(1)
	}

	verifier := appauth.NewOIDCVerifier(provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	}))

	// --- Handlers ---
	queries := database.New(db)
	bookHandler := &handlers.BookHandler{}
	readlistHandler := &handlers.ReadlistHandler{Queries: queries}

	// --- Router ---
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}))
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			handlers.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
			return
		}
		handlers.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public — read-only Open Library proxies, no auth needed
	r.Get("/search", bookHandler.Search)
	r.Get("/details", bookHandler.Details)

	// Protected — all readlist routes require a valid Keycloak token
	r.Route("/readlist", func(r chi.Router) {
		r.Use(appauth.AuthMiddleware(verifier))
		r.Get("/", readlistHandler.GetReadlist)
		r.Post("/", readlistHandler.AddToReadlist)
		r.Get("/{workID}", readlistHandler.GetByWorkID)
		r.Patch("/{id}", readlistHandler.PatchReadlist)
		r.Delete("/{id}", readlistHandler.DeleteFromReadlist)
	})

	// --- Server ---
	srv := &http.Server{
		Addr:         "0.0.0.0:" + cfg.port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
