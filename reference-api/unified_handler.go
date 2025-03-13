package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"

	lru "github.com/hashicorp/golang-lru/v2"
)

type HandlerDeps struct {
	CommitsHandler  http.HandlerFunc
	ReleasesHandler http.HandlerFunc
	cache           *lru.Cache[string, []byte]
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rec *responseRecorder) Write(p []byte) (int, error) {
	rec.body.Write(p)
	return rec.ResponseWriter.Write(p)
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// UnifiedHandler with in-memory caching
func (deps *HandlerDeps) UnifiedHandler(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef")

	if repo == "" || gitRef == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := fmt.Fprintln(w, "Missing 'repo' or 'gitRef' query parameter")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if gitRef == "latest" {
		w.WriteHeader(http.StatusBadRequest)
		errMsg := "'latest' is not a valid value for 'gitRef' please use an immutable image that matches a gitRef on" +
			" your application repository"
		_, err := fmt.Fprintln(w, errMsg)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}

	cacheKey := fmt.Sprintf("%s:%s", repo, gitRef)

	// Check cache
	if cachedResponse, ok := deps.getFromCache(cacheKey); ok {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cachedResponse)
		return
	}

	// Determine if gitRef looks like a commit hash
	commitHashRegex := regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
	if commitHashRegex.MatchString(gitRef) {
		rec := &responseRecorder{ResponseWriter: w, statusCode: 200}
		deps.CommitsHandler(rec, r)
		deps.storeInCache(cacheKey, rec.body.Bytes()) // Cache response
		return
	}

	// Try handling as a release first
	releaseRecorder := &responseRecorder{ResponseWriter: w, statusCode: 200}
	deps.ReleasesHandler(releaseRecorder, r)

	// If release is found, cache and return it
	if releaseRecorder.statusCode != http.StatusNotFound {
		deps.storeInCache(cacheKey, releaseRecorder.body.Bytes())
		return
	}

	// Fall back to commits if no release is found
	commitRecorder := &responseRecorder{ResponseWriter: w, statusCode: 200}
	deps.CommitsHandler(commitRecorder, r)

	// Cache the final commit response
	deps.storeInCache(cacheKey, commitRecorder.body.Bytes())
}

func (deps *HandlerDeps) getFromCache(key string) ([]byte, bool) {
	if value, ok := deps.cache.Get(key); ok {
		log.Printf("Cache hit for key: %s", key)
		return value, true
	}
	log.Printf("Cache miss for key: %s", key)
	return nil, false
}

func (deps *HandlerDeps) storeInCache(key string, value []byte) {
	deps.cache.Add(key, value)
	log.Printf("Cached response for key: %s", key)
}
