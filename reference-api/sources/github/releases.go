package github

import (
	"context"
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
		release := release
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
		errorEncoder(w, http.StatusBadRequest, "Missing 'repo' query parameter")
		return
	}
	if gitRef == "" {
		errorEncoder(w, http.StatusBadRequest, "Missing 'gitRef' query parameter")
		return
	}

	// Fetch release information
	releases, err := FetchReleases(repo, gitRef)
	if err != nil {
		log.Printf("Error fetching releases: %v", err)
		errorEncoder(w, http.StatusInternalServerError, "Failed to fetch release information")
	}

	// Check if a release was found
	if releases == nil || (releases.Current == nil || releases.Latest == nil) {
		errorEncoder(w, http.StatusNotFound, "No release found for the given repository and gitRef")
	}

	// Return release details
	responseEncoder(w, http.StatusOK, releases)

}
