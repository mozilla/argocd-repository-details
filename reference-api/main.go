package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mozilla/argocd-repository-details/reference-api/sources/github"
)

func onEvict(key string, value CachedResponse) {
	// Function called during eviction, log key evicted from cache
	log.Printf("Evicted from cache: %s", key)
}

func main() {
	// Create an LRU cache with eviction callback, cacheSize configurable by environment variable (CacheSize)
	cacheSize := 1000
	if cs := os.Getenv("CacheSize"); cs != "" {
		if parsedSize, err := strconv.Atoi(cs); err == nil {
			cacheSize = parsedSize
		} else {
			log.Printf("Invalid CacheSize value: %s, using default: %d", cs, cacheSize)
		}
	}

	// Create the LRU cache with the correct type
	cache, err := lru.NewWithEvict[string, CachedResponse](cacheSize, onEvict)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	// Initialize dependencies with cache
	deps := &HandlerDeps{
		CommitsHandler:  github.CommitsHandler,
		ReleasesHandler: github.ReleasesHandler,
		cache:           cache,
	}

	// Unified handler for both releases and commits, may support additional sources in the future.
	http.HandleFunc("/api/references", deps.UnifiedHandler)

	// Determine the port
	port := "8000"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
