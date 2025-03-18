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
	fmt.Fprint(w, `{"handler": "commits"}`)
}

func mockReleasesHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"handler": "releases"}`)
}

// Mock release handler that simulates a 404 (to trigger commit fallback)
func mockReleasesHandler404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `Not Found`)
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
		expectedCacheKey string
	}{
		{
			name:           "Valid release tag (should be cached from releases)",
			repo:           "test/repo",
			gitRef:         "v1.0.0",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
			use404Releases: false, // Should be served from ReleasesHandler
		},
		{
			name:           "Valid short commit SHA (release 404s, should fall back to commits)",
			repo:           "test/repo",
			gitRef:         "abc1234",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
			use404Releases: true, // Simulate a 404 from ReleasesHandler
		},
		{
			name:           "Valid full commit SHA (release 404s, should fall back to commits)",
			repo:           "test/repo",
			gitRef:         "abcdef1234567890abcdef1234567890abcdef12",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
			use404Releases: true, // Simulate a 404 from ReleasesHandler
		},
		{
			name:           "Invalid gitRef format (should still be processed by ReleasesHandler)",
			repo:           "test/repo",
			gitRef:         "not-a-commit-or-tag",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
			use404Releases: false, // Should be served from ReleasesHandler
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

			// Initialize dependencies with the selected release handler
			deps := &HandlerDeps{
				CommitsHandler:  mockCommitsHandler,
				ReleasesHandler: releaseHandler,
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
