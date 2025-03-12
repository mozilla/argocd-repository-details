package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"
)

// Mock handlers
func mockCommitsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"handler": "commits"}`)
}

func mockReleasesHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"handler": "releases"}`)
}

func TestUnifiedHandlerWithCache(t *testing.T) {
	// Create an LRU cache with 5 items max (to test eviction easily)
	cache, err := lru.NewWithEvict[string, []byte](5, onEvict)
	assert.NoError(t, err, "Failed to initialize cache")

	// Mock dependencies with cache
	deps := &HandlerDeps{
		CommitsHandler:  mockCommitsHandler,
		ReleasesHandler: mockReleasesHandler,
		cache:           cache,
	}

	tests := []struct {
		name           string
		repo           string
		gitRef         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid short commit SHA (should be cached)",
			repo:           "test/repo",
			gitRef:         "abc1234",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
		},
		{
			name:           "Valid full commit SHA (should be cached)",
			repo:           "test/repo",
			gitRef:         "abcdef1234567890abcdef1234567890abcdef12",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
		},
		{
			name:           "Valid tag (should be cached)",
			repo:           "test/repo",
			gitRef:         "v1.0.0",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
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
		{
			name:           "Invalid gitRef format (should be cached)",
			repo:           "test/repo",
			gitRef:         "not-a-commit-or-tag",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test: %s", tt.name) // Output test name

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
				assert.Equal(t, tt.expectedBody, string(cachedData))
			}
		})

	}

	// Additional test: Cache eviction after exceeding max size
	t.Run("Cache Eviction", func(t *testing.T) {
		// Add more entries to force eviction
		for i := 1; i <= 6; i++ { // Exceed cache limit of 5
			cacheKey := fmt.Sprintf("repo%d:gitRef%d", i, i)
			deps.storeInCache(cacheKey, []byte(fmt.Sprintf(`{"handler": "test%d"}`, i)))
			time.Sleep(10 * time.Millisecond) // Allow eviction logging to appear
		}

		// Check if the first entry was evicted
		_, found := deps.getFromCache("repo1:gitRef1")
		assert.False(t, found, "Expected first cached item to be evicted")
	})
}
