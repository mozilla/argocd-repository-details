package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-github/v67/github"
)

type MergedCommits struct {
	Latest  *github.RepositoryCommit `json:"latest"`
	Current *github.RepositoryCommit `json:"current"`
}

// FetchCommit fetches a specific commit by its Git reference
func FetchCommit(repo, gitRef string) (*github.RepositoryCommit, error) {
	owner, repoName, _ := strings.Cut(repo, "/")
	client := NewGithubClient((repo))
	ctx := context.Background()
	commit, _, err := client.Repositories.GetCommit(ctx, owner, repoName, gitRef, nil)
	if err != nil {
		return nil, err
	}
	return commit, nil
}

// FetchLatestCommit fetches the latest commit from the repository
func FetchLatestCommit(repo string) (*github.RepositoryCommit, error) {
	owner, repoName, _ := strings.Cut(repo, "/")
	client := NewGithubClient((repo))
	ctx := context.Background()
	commits, _, err := client.Repositories.ListCommits(ctx, owner, repoName, nil)
	if err != nil {
		return nil, err
	}

	if len(commits) > 0 {
		return commits[0], nil
	}

	return nil, fmt.Errorf("no commits found")
}

// FetchCommits fetches both the latest commit and the one matching the Git reference (gitRef)
func FetchCommits(repo, gitRef string) (*StandardizedOutput, int, error) {
	// Fetch the current commit
	currentCommit, err := FetchCommit(repo, gitRef)
	if err != nil {
		log.Printf("Error fetching current commit: %v", err)
		if strings.Contains(err.Error(), "404") {
			return nil, http.StatusNotFound, fmt.Errorf("commit not found for gitRef: %s", gitRef)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching commit: %v", err)
	}

	// Fetch the latest commit
	latestCommit, err := FetchLatestCommit(repo)
	if err != nil {
		log.Printf("Error fetching latest commit: %v", err)
		latestCommit = nil // Allow partial results
	}

	// Standardize the commit response
	return &StandardizedOutput{
		Latest:  StandardizeCommit(latestCommit),
		Current: StandardizeCommit(currentCommit),
	}, http.StatusOK, nil
}

// CommitsHandler handles the API endpoint for fetching the latest and current commits
func CommitsHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef")
	if repo == "" {
		errorEncoder(w, http.StatusBadRequest, "Missing 'repo' query parameter")
		return
	}
	if gitRef == "" {
		errorEncoder(w, http.StatusBadRequest, "Missing 'gitRef' query parameter")
		return
	}

	commitHashRegex := regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
	if !commitHashRegex.MatchString(gitRef) {
		errorEncoder(w, http.StatusBadRequest, "Invalid gitRef format: Expected a commit SHA")
		return
	}

	// Fetch the commits and get the response status
	commits, statusCode, err := FetchCommits(repo, gitRef)
	if err != nil {
		errorEncoder(w, statusCode, err.Error()) // Return correct status code
		return
	}

	// Set response headers and send response
	w.Header().Set("Content-Type", "application/json")
	responseEncoder(w, statusCode, commits)
}
