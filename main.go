package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"juicecon-golang/internal/handler"
)

//go:embed static
var staticFiles embed.FS

func main() {
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
		w.Write(data)
	})

	log.Printf("JUICECON server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
