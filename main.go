package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"juicecon-golang/internal/handler"
	"juicecon-golang/internal/middleware"
)

var startTime time.Time

//go:embed static
var staticFiles embed.FS

func main() {
	startTime = time.Now()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// API handler
	apiHandler := handler.New()

	// Static file server
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// API endpoint
	mux.Handle("/api/juicecon", apiHandler)

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"uptime": fmt.Sprintf("%s", time.Since(startTime).Round(time.Second)),
		}); err != nil {
			log.Printf("healthz: failed to write response: %v", err)
		}
	})

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Root serves index.html
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write(data); err != nil {
			log.Printf("root: failed to write response: %v", err)
		}
	})

	// Rate limiter: 60 requests per minute, cleanup every 5 minutes.
	// Only applied to /api/ routes.
	rateLimiter := middleware.NewRateLimiter(60, time.Minute, 5*time.Minute)
	defer rateLimiter.Stop()

	// Apply middleware: security headers on all routes, rate limiting on /api/
	wrapped := middleware.SecurityHeaders(rateLimiter.Wrap("/api/", mux))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: wrapped,
	}

	// Listen for shutdown signals in a separate goroutine.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		log.Printf("Received signal %v, shutting down...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("DEWCON system starting on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped")
}
