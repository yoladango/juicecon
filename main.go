package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"juicecon-golang/internal/handler"
	"juicecon-golang/internal/middleware"
	"juicecon-golang/internal/weather"
)

var startTime time.Time

//go:embed static
var staticFiles embed.FS

func main() {
	startTime = time.Now()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Weather client (shared by API handler and healthz)
	weatherClient := weather.NewClient()

	// API handler
	apiHandler := handler.NewWithWeatherClient(weatherClient)

	// Static file server
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("failed to create static filesystem", slog.String("error", err.Error()))
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// API endpoint (primary)
	mux.Handle("/api/dewcon", apiHandler)
	// Backward-compatible alias
	mux.Handle("/api/juicecon", apiHandler)

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := struct {
			Status string              `json:"status"`
			Uptime string              `json:"uptime"`
			Cache  weather.CacheStats  `json:"cache"`
		}{
			Status: "ok",
			Uptime: fmt.Sprintf("%s", time.Since(startTime).Round(time.Second)),
			Cache:  weatherClient.CacheStats(),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("healthz: failed to write response", slog.String("error", err.Error()))
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
			slog.Error("root: failed to write response", slog.String("error", err.Error()))
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
		slog.Info("received shutdown signal", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", slog.String("error", err.Error()))
		}
	}()

	slog.Info("DEWCON system starting", slog.String("port", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("server stopped")
}
