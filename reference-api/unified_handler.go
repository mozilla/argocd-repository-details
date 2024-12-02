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

// UnifiedHandler decides between using the CommitsHandler or ReleasesHandler based on gitRef format
func (deps *HandlerDeps) UnifiedHandler(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef")

	if repo == "" || gitRef == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Missing 'repo' or 'gitRef' query parameter")
		return
	}

	// Return an error if gitRef is "latest"
	if gitRef == "latest" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "'latest' is not a valid value for 'gitRef' please use an immutable image that matches a gitRef on your application repository")
		return
	}

	// Determine if gitRef looks like a commit hash (short or full)
	commitHashRegex := regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`) // Match 7 to 40 hexadecimal characters
	if commitHashRegex.MatchString(gitRef) {
		// Handle as a commit
		deps.CommitsHandler(w, r)
	} else {
		// Handle as a tag (release)
		deps.ReleasesHandler(w, r)
	}
}
