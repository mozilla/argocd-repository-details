package main

import (
	"fmt"
	"net/http"
	"regexp"
)

type HandlerDeps struct {
	CommitsHandler  http.HandlerFunc
	ReleasesHandler http.HandlerFunc
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// UnifiedHandler decides between using the CommitsHandler or ReleasesHandler based on gitRef format
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

	// Return an error if gitRef is "latest"
	if gitRef == "latest" {
		w.WriteHeader(http.StatusBadRequest)
		errMsg := "'latest' is not a valid value for 'gitRef' please use an immutable image that matches a gitRef on" +
			" your application repository"
		_, err := fmt.Fprintln(w, errMsg)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Determine if gitRef looks like a commit hash (short or full)
	commitHashRegex := regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`) // Match 7 to 40 hexadecimal characters
	if commitHashRegex.MatchString(gitRef) {
		// Handle as a commit
		deps.CommitsHandler(w, r)
		return
	}

	// Attempt to handle as a release
	releaseRecorder := &responseRecorder{ResponseWriter: w, statusCode: 200}
	deps.ReleasesHandler(releaseRecorder, r)

	// Fall back to commits if no release is found
	if releaseRecorder.statusCode == http.StatusNotFound {
		deps.CommitsHandler(w, r)
	}
}
