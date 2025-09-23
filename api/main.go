package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/jasperwreed/ai-memory/api/handlers"
	"github.com/jasperwreed/ai-memory/api/middleware"
	_ "modernc.org/sqlite"
)

type Server struct {
	db     *sql.DB
	router *http.ServeMux
	port   string
}

func main() {
	var port string
	var dbPath string

	flag.StringVar(&port, "port", "8080", "Port to run the API server on")
	flag.StringVar(&dbPath, "db", "", "Path to database file")
	flag.Parse()

	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		dbPath = filepath.Join(homeDir, ".ai-memory", "conversations.db")
	}

	db, err := openDatabase(dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	server := NewServer(db, port)

	log.Printf("Starting API server on port %s", port)
	log.Printf("Database: %s", dbPath)
	log.Println("Press Ctrl+C to stop")

	if err := server.Start(); err != nil {
		log.Fatal("Server error:", err)
	}
}

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func NewServer(db *sql.DB, port string) *Server {
	s := &Server{
		db:     db,
		router: http.NewServeMux(),
		port:   port,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	h := handlers.NewHandlers(s.db)

	// Apply middleware
	withCORS := middleware.CORS
	withLogging := middleware.Logging
	withJSON := middleware.JSON

	// Health check
	s.router.HandleFunc("GET /api/health", withLogging(withJSON(h.Health)))

	// Conversation routes
	s.router.HandleFunc("GET /api/conversations", withLogging(withCORS(withJSON(h.ListConversations))))
	s.router.HandleFunc("GET /api/conversations/{id}", withLogging(withCORS(withJSON(h.GetConversation))))
	s.router.HandleFunc("POST /api/conversations", withLogging(withCORS(withJSON(h.CreateConversation))))
	s.router.HandleFunc("DELETE /api/conversations/{id}", withLogging(withCORS(withJSON(h.DeleteConversation))))

	// Search route
	s.router.HandleFunc("GET /api/search", withLogging(withCORS(withJSON(h.Search))))

	// Statistics route
	s.router.HandleFunc("GET /api/stats", withLogging(withCORS(withJSON(h.GetStatistics))))

	// Messages routes
	s.router.HandleFunc("GET /api/conversations/{id}/messages", withLogging(withCORS(withJSON(h.GetMessages))))

	// Export route
	s.router.HandleFunc("GET /api/conversations/{id}/export", withLogging(withCORS(h.ExportConversation)))
}

func (s *Server) Start() error {
	srv := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server shutdown complete")
	return nil
}