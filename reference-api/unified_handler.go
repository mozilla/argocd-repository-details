package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type HandlerDeps struct {
	CommitsHandler  http.HandlerFunc
	ReleasesHandler http.HandlerFunc
	cache           *lru.Cache[string, CachedResponse]
	config          cacheConfiguration
}

type CachedResponse struct {
	StatusCode int
	Body       []byte
	Timestamp  int64 // Unix timestamp when stored
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

type cacheConfiguration struct {
	SuccessCacheDuration time.Duration
	ErrorCacheDuration   time.Duration
}

func (rec *responseRecorder) Write(p []byte) (int, error) {
	if rec.statusCode == 0 { // Default to 200 if WriteHeader was never called
		rec.statusCode = http.StatusOK
	}
	rec.body.Write(p)
	return rec.ResponseWriter.Write(p)
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// UnifiedHandler with in-memory caching
func (deps *HandlerDeps) UnifiedHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef")

	if repo == "" || gitRef == "" {
		http.Error(w, "Missing 'repo' or 'gitRef' query parameter", http.StatusBadRequest)
		return
	}

	if gitRef == "latest" {
		http.Error(w, "'latest' is not a valid value for 'gitRef'. Please use an immutable image.", http.StatusBadRequest)
		return
	}

	// Strip optional --<metadata> suffix from image tags
	//  "v1.2.3--release" → "v1.2.3"
	//  "dd295fd679--stage" → "dd295fd679"
	baseGitRef, _, _ := strings.Cut(gitRef, "--")

	// Use base gitRef for cache key (tags with different metadata share the same cache entry)
	cacheKey := fmt.Sprintf("%s:%s", repo, baseGitRef)

	// Check cache first
	if cachedResponse, ok := deps.getFromCache(cacheKey); ok {
		w.WriteHeader(cachedResponse.StatusCode)
		_, _ = w.Write(cachedResponse.Body)
		return
	}

	// Modify request to use base gitRef (without metadata) for GitHub queries
	q := r.URL.Query()
	q.Set("gitRef", baseGitRef)
	r.URL.RawQuery = q.Encode()

	// Try handling as a release first
	releaseRecorder := &responseRecorder{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     0, // Use a separate ResponseRecorder
	}
	deps.ReleasesHandler(releaseRecorder, r)

	// If release is found (not 404), cache and return
	if releaseRecorder.statusCode != http.StatusNotFound {
		w.WriteHeader(releaseRecorder.statusCode)
		if _, err := w.Write(releaseRecorder.body.Bytes()); err != nil {
			log.Printf("Error writing response: %v", err)
		}
		deps.storeInCache(cacheKey, releaseRecorder.statusCode, releaseRecorder.body.Bytes())
		return
	}

	// Clear previous response before falling back
	w.Header().Del("Content-Type") // Ensure correct content type is set by CommitsHandler
	w.WriteHeader(http.StatusOK)   // Reset status before falling back

	// Fall back to CommitsHandler
	deps.processAndCacheResponse(deps.CommitsHandler, w, r, cacheKey)
}

func (deps *HandlerDeps) processAndCacheResponse(
	handler http.HandlerFunc,
	w http.ResponseWriter,
	r *http.Request,
	cacheKey string,
) {
	rec := &responseRecorder{ResponseWriter: w, statusCode: 0} // Use 0 so that we can catch missing status updates
	handler(rec, r)

	// Store both status and response body
	deps.storeInCache(cacheKey, rec.statusCode, rec.body.Bytes())
}

func (deps *HandlerDeps) getFromCache(key string) (CachedResponse, bool) {
	if value, ok := deps.cache.Get(key); ok {
		currentTime := time.Now()

		// Determine expiration time based on status code
		var expirationTime time.Time
		if value.StatusCode == http.StatusOK {
			expirationTime = time.Unix(value.Timestamp, 0).Add(deps.config.SuccessCacheDuration)
		} else {
			expirationTime = time.Unix(value.Timestamp, 0).Add(deps.config.ErrorCacheDuration)
		}

		// Check if the cache entry has expired
		if currentTime.After(expirationTime) {
			log.Printf("Cache expired for key: %s (Stored at: %s, Expired at: %s)",
				key, time.Unix(value.Timestamp, 0), expirationTime)
			deps.cache.Remove(key)
			return CachedResponse{}, false
		}

		log.Printf("Cache hit for key: %s (Stored at: %s, Expires at: %s), Status: %d",
			key, time.Unix(value.Timestamp, 0), expirationTime, value.StatusCode)
		return value, true
	}

	log.Printf("Cache miss for key: %s", key)
	return CachedResponse{}, false
}

func (deps *HandlerDeps) storeInCache(key string, statusCode int, value []byte) {
	timestamp := time.Now().Unix() // Store current timestamp

	deps.cache.Add(key, CachedResponse{
		StatusCode: statusCode,
		Body:       value,
		Timestamp:  timestamp,
	})

	log.Printf("Cached response for key: %s, Status: %d (Stored at %d)", key, statusCode, timestamp)
}
