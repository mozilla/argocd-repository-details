package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"
)

// Mock handlers for testing
func mockCommitsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, `{"handler": "commits"}`); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func mockReleasesHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, `{"handler": "releases"}`); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func mockTagsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, `{"handler": "tags"}`); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// Mock release handler that simulates a 404 (to trigger tag fallback)
func mockReleasesHandler404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if _, err := fmt.Fprint(w, `Not Found`); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// Mock tags handler that simulates a 404 (to trigger commit fallback)
func mockTagsHandler404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if _, err := fmt.Fprint(w, `Not Found`); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func TestUnifiedHandlerWithCache(t *testing.T) {
	// Create an LRU cache with 5 items max (to test eviction easily)
	cache, err := lru.NewWithEvict[string, CachedResponse](5, onEvict)
	assert.NoError(t, err, "Failed to initialize cache")

	// Test cases
	tests := []struct {
		name             string
		repo             string
		gitRef           string
		expectedStatus   int
		expectedBody     string
		use404Releases   bool // Whether to simulate a 404 response from ReleasesHandler
		use404Tags       bool // Whether to simulate a 404 response from TagsHandler
		expectedCacheKey string
	}{
		{
			name:           "Valid release tag (should be cached from releases)",
			repo:           "test/repo",
			gitRef:         "v1.0.0",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
			use404Releases: false,
			use404Tags:     false,
		},
		{
			name:           "Valid git tag (releases 404, should be served from tags)",
			repo:           "test/repo",
			gitRef:         "v2.0.0",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "tags"}`,
			use404Releases: true,
			use404Tags:     false,
		},
		{
			name:           "Valid short commit SHA (releases and tags 404, should fall back to commits)",
			repo:           "test/repo",
			gitRef:         "abc1234",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
			use404Releases: true,
			use404Tags:     true,
		},
		{
			name:           "Valid full commit SHA (releases and tags 404, should fall back to commits)",
			repo:           "test/repo",
			gitRef:         "abcdef1234567890abcdef1234567890abcdef12",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
			use404Releases: true,
			use404Tags:     true,
		},
		{
			name:           "Invalid gitRef format (should still be processed by ReleasesHandler)",
			repo:           "test/repo",
			gitRef:         "not-a-commit-or-tag",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
			use404Releases: false,
			use404Tags:     false,
		},
		{
			name:           "Missing repo parameter",
			repo:           "",
			gitRef:         "v1.0.0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'repo' or 'gitRef' query parameter\n",
		},
		{
			name:           "Missing gitRef parameter",
			repo:           "test/repo",
			gitRef:         "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'repo' or 'gitRef' query parameter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test: %s", tt.name)

			// Mock dependencies based on whether we want a 404 for releases
			releaseHandler := mockReleasesHandler
			if tt.use404Releases {
				releaseHandler = mockReleasesHandler404
			}

			// Mock dependencies based on whether we want a 404 for tags
			tagsHandler := mockTagsHandler
			if tt.use404Tags {
				tagsHandler = mockTagsHandler404
			}

			// Initialize dependencies with the selected handlers
			deps := &HandlerDeps{
				CommitsHandler:  mockCommitsHandler,
				ReleasesHandler: releaseHandler,
				TagsHandler:     tagsHandler,
				cache:           cache,
				config: cacheConfiguration{
					SuccessCacheDuration: 24 * time.Hour,
					ErrorCacheDuration:   1 * time.Hour,
				},
			}

			// Create a mock HTTP request
			req := httptest.NewRequest("GET", "/api/references", nil)
			q := req.URL.Query()
			q.Add("repo", tt.repo)
			q.Add("gitRef", tt.gitRef)
			req.URL.RawQuery = q.Encode()

			// Create a ResponseRecorder to capture the response
			rr := httptest.NewRecorder()

			// Call the UnifiedHandler
			handler := http.HandlerFunc(deps.UnifiedHandler)
			handler.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check the response body
			assert.Equal(t, tt.expectedBody, rr.Body.String())

			// Verify the response is cached
			cacheKey := fmt.Sprintf("%s:%s", tt.repo, tt.gitRef)
			cachedData, found := deps.getFromCache(cacheKey)
			if tt.expectedStatus == http.StatusOK {
				assert.True(t, found, "Expected response to be cached")
				assert.Equal(t, tt.expectedStatus, cachedData.StatusCode, "Cached status code mismatch")
				assert.Equal(t, tt.expectedBody, string(cachedData.Body), "Cached body mismatch")
			} else {
				assert.False(t, found, "Did not expect response to be cached for error status")
			}
		})
	}

	// Cache expiration test
	t.Run("Cache Expiration", func(t *testing.T) {
		// Initialize cache with a test-specific instance
		cache, err := lru.NewWithEvict[string, CachedResponse](5, onEvict)
		assert.NoError(t, err, "Failed to initialize cache")

		// Initialize a fresh HandlerDeps struct for this test
		deps := &HandlerDeps{
			CommitsHandler:  mockCommitsHandler,
			ReleasesHandler: mockReleasesHandler,
			TagsHandler:     mockTagsHandler,
			cache:           cache,
			config: cacheConfiguration{
				SuccessCacheDuration: 24 * time.Hour,
				ErrorCacheDuration:   1 * time.Hour,
			},
		}

		cacheKey := "test/repo:expired-entry"
		currentTime := time.Now().Unix()

		// Store a 200 response (should expire in 24 hours)
		deps.storeInCache(cacheKey, http.StatusOK, []byte(`{"handler": "commits"}`))

		// Ensure the response is stored
		cachedResponse, found := deps.cache.Get(cacheKey)
		assert.True(t, found, "Expected cached item to be present before expiration")

		// Manually modify the timestamp to simulate expiration (set to 25 hours ago)
		cachedResponse.Timestamp = currentTime - int64(25*time.Hour.Seconds())
		deps.cache.Add(cacheKey, cachedResponse) // Re-insert the modified entry

		// Ensure cache entry is now expired
		_, found = deps.getFromCache(cacheKey)
		assert.False(t, found, "Expected cached item to be evicted after 24 hours")
	})

	// Cache size-based eviction test (unchanged)
	t.Run("Cache Eviction (Size-Based)", func(t *testing.T) {
		for i := 1; i <= 6; i++ { // Exceed cache limit of 5
			cacheKey := fmt.Sprintf("repo%d:gitRef%d", i, i)
			cache.Add(cacheKey, CachedResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(fmt.Sprintf(`{"handler": "test%d"}`, i)),
				Timestamp:  time.Now().Unix(),
			})
			time.Sleep(10 * time.Millisecond) // Allow eviction logging to appear
		}

		// Check if the first entry was evicted
		_, found := cache.Get("repo1:gitRef1")
		assert.False(t, found, "Expected first cached item to be evicted")
	})
}

// TestUnifiedHandlerWithMetadataSuffix verifies that gitRefs with metadata suffixes are handled correctly
func TestUnifiedHandlerWithMetadataSuffix(t *testing.T) {
	// Create an LRU cache
	cache, err := lru.NewWithEvict[string, CachedResponse](10, onEvict)
	assert.NoError(t, err, "Failed to initialize cache")

	deps := &HandlerDeps{
		CommitsHandler:  mockCommitsHandler,
		ReleasesHandler: mockReleasesHandler,
		TagsHandler:     mockTagsHandler,
		cache:           cache,
		config: cacheConfiguration{
			SuccessCacheDuration: 24 * time.Hour,
			ErrorCacheDuration:   1 * time.Hour,
		},
	}

	repo := "test/repo"
	baseGitRef := "v1.2.3"

	// First request with metadata suffix
	req1 := httptest.NewRequest("GET", "/api/references", nil)
	q1 := req1.URL.Query()
	q1.Add("repo", repo)
	q1.Add("gitRef", "v1.2.3--release")
	req1.URL.RawQuery = q1.Encode()

	rr1 := httptest.NewRecorder()
	handler := http.HandlerFunc(deps.UnifiedHandler)
	handler.ServeHTTP(rr1, req1)

	assert.Equal(t, http.StatusOK, rr1.Code)
	assert.Equal(t, `{"handler": "releases"}`, rr1.Body.String())

	// Verify cache key uses base gitRef (without metadata)
	expectedCacheKey := fmt.Sprintf("%s:%s", repo, baseGitRef)
	cachedData, found := deps.getFromCache(expectedCacheKey)
	assert.True(t, found, "Expected response to be cached with key %s", expectedCacheKey)

	// Second request with different metadata suffix should hit cache
	req2 := httptest.NewRequest("GET", "/api/references", nil)
	q2 := req2.URL.Query()
	q2.Add("repo", repo)
	q2.Add("gitRef", "v1.2.3--stage")
	req2.URL.RawQuery = q2.Encode()

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	// Should get same cached response
	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Equal(t, `{"handler": "releases"}`, rr2.Body.String())
	assert.Equal(t, string(cachedData.Body), rr2.Body.String(), "Different metadata suffixes should share cache")
}

// TestUnifiedHandlerTagFallback verifies that the handler correctly falls back from releases to tags to commits
func TestUnifiedHandlerTagFallback(t *testing.T) {
	cache, err := lru.NewWithEvict[string, CachedResponse](10, onEvict)
	assert.NoError(t, err, "Failed to initialize cache")

	tests := []struct {
		name           string
		gitRef         string
		expectedBody   string
		expectedStatus int
		description    string
	}{
		{
			name:           "Release exists - serve from releases",
			gitRef:         "v1.0.0-release",
			expectedBody:   `{"handler": "releases"}`,
			expectedStatus: http.StatusOK,
			description:    "Should be served from ReleasesHandler",
		},
		{
			name:           "Release 404, tag exists - serve from tags",
			gitRef:         "v1.0.0-tag",
			expectedBody:   `{"handler": "tags"}`,
			expectedStatus: http.StatusOK,
			description:    "Should fall back to TagsHandler",
		},
		{
			name:           "Release and tag 404, commit exists - serve from commits",
			gitRef:         "abc123commit",
			expectedBody:   `{"handler": "commits"}`,
			expectedStatus: http.StatusOK,
			description:    "Should fall back to CommitsHandler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Custom release handler that only succeeds for "-release" suffix
			releaseHandler := func(w http.ResponseWriter, r *http.Request) {
				gitRef := r.URL.Query().Get("gitRef")
				if len(gitRef) > 8 && gitRef[len(gitRef)-8:] == "-release" {
					w.WriteHeader(http.StatusOK)
					if _, err := fmt.Fprint(w, `{"handler": "releases"}`); err != nil {
						log.Printf("Error writing response: %v", err)
					}
				} else {
					w.WriteHeader(http.StatusNotFound)
					if _, err := fmt.Fprint(w, `Not Found`); err != nil {
						log.Printf("Error writing response: %v", err)
					}
				}
			}

			// Custom tags handler that only succeeds for "-tag" suffix
			tagsHandler := func(w http.ResponseWriter, r *http.Request) {
				gitRef := r.URL.Query().Get("gitRef")
				if len(gitRef) > 4 && gitRef[len(gitRef)-4:] == "-tag" {
					w.WriteHeader(http.StatusOK)
					if _, err := fmt.Fprint(w, `{"handler": "tags"}`); err != nil {
						log.Printf("Error writing response: %v", err)
					}
				} else {
					w.WriteHeader(http.StatusNotFound)
					if _, err := fmt.Fprint(w, `Not Found`); err != nil {
						log.Printf("Error writing response: %v", err)
					}
				}
			}

			// Custom commits handler that only succeeds for "commit" substring
			commitsHandler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := fmt.Fprint(w, `{"handler": "commits"}`); err != nil {
					log.Printf("Error writing response: %v", err)
				}
			}

			deps := &HandlerDeps{
				CommitsHandler:  commitsHandler,
				ReleasesHandler: releaseHandler,
				TagsHandler:     tagsHandler,
				cache:           cache,
				config: cacheConfiguration{
					SuccessCacheDuration: 24 * time.Hour,
					ErrorCacheDuration:   1 * time.Hour,
				},
			}

			req := httptest.NewRequest("GET", "/api/references", nil)
			q := req.URL.Query()
			q.Add("repo", "test/repo")
			q.Add("gitRef", tt.gitRef)
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(deps.UnifiedHandler)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.description)
			assert.Equal(t, tt.expectedBody, rr.Body.String(), tt.description)
		})
	}
}

// TestCacheConfigurationEnvVars verifies that cache durations are correctly parsed from environment variables.
func TestCacheConfigurationEnvVars(t *testing.T) {
	// Save existing environment variables
	origSuccess := os.Getenv("CACHE_SUCCESS_DURATION")
	origError := os.Getenv("CACHE_ERROR_DURATION")

	// Restore the original environment variables after test
	defer func() {
		if origSuccess != "" {
			_ = os.Setenv("CACHE_SUCCESS_DURATION", origSuccess)
		} else {
			_ = os.Unsetenv("CACHE_SUCCESS_DURATION")
		}
		if origError != "" {
			_ = os.Setenv("CACHE_ERROR_DURATION", origError)
		} else {
			_ = os.Unsetenv("CACHE_ERROR_DURATION")
		}
	}()

	// Define test cases
	tests := []struct {
		name               string
		successDuration    string
		errorDuration      string
		expectedSuccessDur time.Duration
		expectedErrorDur   time.Duration
	}{
		{
			name:               "Valid durations",
			successDuration:    "48",
			errorDuration:      "2",
			expectedSuccessDur: 48 * time.Hour,
			expectedErrorDur:   2 * time.Hour,
		},
		{
			name:               "Invalid durations (fallback to default)",
			successDuration:    "invalid",
			errorDuration:      "-6",
			expectedSuccessDur: 24 * time.Hour,
			expectedErrorDur:   1 * time.Hour,
		},
		{
			name:               "Empty env vars (fallback to default)",
			successDuration:    "",
			errorDuration:      "",
			expectedSuccessDur: 24 * time.Hour,
			expectedErrorDur:   1 * time.Hour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set test environment variables
			_ = os.Setenv("CACHE_SUCCESS_DURATION", tc.successDuration)
			_ = os.Setenv("CACHE_ERROR_DURATION", tc.errorDuration)

			// Call the logic to initialize cache configuration
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

			// Validate cache durations
			if cacheConfig.SuccessCacheDuration != tc.expectedSuccessDur {
				t.Errorf("Expected SuccessCacheDuration = %v, got %v", tc.expectedSuccessDur, cacheConfig.SuccessCacheDuration)
			}
			if cacheConfig.ErrorCacheDuration != tc.expectedErrorDur {
				t.Errorf("Expected ErrorCacheDuration = %v, got %v", tc.expectedErrorDur, cacheConfig.ErrorCacheDuration)
			}
		})
	}
}
