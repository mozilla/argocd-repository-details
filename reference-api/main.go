package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dlactin/argocd-repository-details/reference-api/sources/github"
)

func main() {
	deps := &HandlerDeps{CommitsHandler: github.CommitsHandler, ReleasesHandler: github.ReleasesHandler}

	// Unified handler for both releases and commits, may support additional sources in the future.
	http.HandleFunc("/api/references", deps.UnifiedHandler)

	port := "8000"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
