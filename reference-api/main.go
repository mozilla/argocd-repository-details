package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mozilla/argocd-repository-details/reference-api/sources/github"
)

func onEvict(key string, value CachedResponse) {
	log.Printf("Evicted from cache: %s", key)
}

func main() {
	cacheSize := 1000
	if cs := os.Getenv("CacheSize"); cs != "" {
		if parsedSize, err := strconv.Atoi(cs); err == nil {
			cacheSize = parsedSize
		} else {
			log.Printf("Invalid CacheSize value: %s, using default: %d", cs, cacheSize)
		}
	}

	cache, err := lru.NewWithEvict[string, CachedResponse](cacheSize, onEvict)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	// Default expiration durations
	const (
		defaultSuccessDuration = 24 * time.Hour
		defaultErrorDuration   = 1 * time.Hour
	)

	cacheConfig := cacheConfiguration{
		SuccessCacheDuration: defaultSuccessDuration,
		ErrorCacheDuration:   defaultErrorDuration,
	}

	if scd := os.Getenv("CACHE_SUCCESS_DURATION"); scd != "" {
		duration, err := strconv.Atoi(scd)
		if err != nil || duration < 0 {
			log.Printf("Warning: Invalid CACHE_SUCCESS_DURATION: %s. Using default: %v", scd, defaultSuccessDuration)
		} else {
			cacheConfig.SuccessCacheDuration = time.Duration(duration) * time.Hour
		}
	}

	if ecd := os.Getenv("CACHE_ERROR_DURATION"); ecd != "" {
		duration, err := strconv.Atoi(ecd)
		if err != nil || duration < 0 {
			log.Printf("Warning: Invalid CACHE_ERROR_DURATION: %s. Using default: %v", ecd, defaultErrorDuration)
		} else {
			cacheConfig.ErrorCacheDuration = time.Duration(duration) * time.Hour
		}
	}

	deps := &HandlerDeps{
		CommitsHandler:  github.CommitsHandler,
		ReleasesHandler: github.ReleasesHandler,
		cache:           cache,
		config:          cacheConfig,
	}

	// Unified handler for both releases and commits, may support additional sources in the future.
	http.HandleFunc("/api/references", deps.UnifiedHandler)

	port := "8000"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
