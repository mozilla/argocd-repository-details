package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestUnifiedHandler(t *testing.T) {
	// Mock dependencies
	deps := &HandlerDeps{
		CommitsHandler:  mockCommitsHandler,
		ReleasesHandler: mockReleasesHandler,
	}

	tests := []struct {
		name           string
		repo           string
		gitRef         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid short commit SHA",
			repo:           "test/repo",
			gitRef:         "abc1234",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
		},
		{
			name:           "Valid full commit SHA",
			repo:           "test/repo",
			gitRef:         "abcdef1234567890abcdef1234567890abcdef12",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "commits"}`,
		},
		{
			name:           "Valid tag",
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
			name:           "Invalid gitRef format",
			repo:           "test/repo",
			gitRef:         "not-a-commit-or-tag",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"handler": "releases"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP request
			req := httptest.NewRequest("GET", "/api/releases", nil)
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
		})
	}
}
