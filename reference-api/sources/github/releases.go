package github

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v67/github"
)

// Release represents a GitHub release
type Release struct {
	TagName     string `json:"tag_name"`
	URL         string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
}
type MergedReleases struct {
	Latest  *Release `json:"latest"`
	Current *Release `json:"current"`
}

func FetchReleases(repo, gitRef string) (*StandardizedOutput, error) {
	owner, repoName, _ := strings.Cut(repo, "/")
	client := NewGithubClient(repo)
	ctx := context.Background()
	releases, _, err := client.Repositories.ListReleases(ctx, owner, repoName, nil)
	if err != nil {
		return nil, err
	}

	var latestRelease *github.RepositoryRelease
	var matchingRelease *github.RepositoryRelease

	// Iterate through releases to find the latest and the matching release
	for _, release := range releases {
		// Identify the latest release (first in the list, assuming GitHub returns them in descending order)
		if latestRelease == nil {
			latestRelease = release
		}

		// Identify the release matching the gitRef
		if *release.TagName == gitRef {
			matchingRelease = release
		}

		// If both latest and matching releases are found, exit the loop early
		if latestRelease != nil && matchingRelease != nil {
			break
		}
	}

	// Normalize the releases
	return &StandardizedOutput{
		Latest:  StandardizeRelease(latestRelease),
		Current: StandardizeRelease(matchingRelease),
	}, nil
}

// ReleasesHandler handles the API endpoint for fetching releases from target repositories
func ReleasesHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef")

	// Validate query parameters
	if repo == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Missing 'repo' query parameter"})
		return
	}
	if gitRef == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Missing 'gitRef' query parameter"})
		return
	}

	// Fetch release information
	releases, err := FetchReleases(repo, gitRef)
	if err != nil {
		log.Printf("Error fetching releases: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to fetch release information"})
		return
	}

	// Check if a release was found
	if releases == nil || (releases.Current == nil || releases.Latest == nil) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "No release found for the given repository and gitRef"})
		return
	}

	// Return release details
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(releases)
}
