package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v67/github"
)

type MergedCommits struct {
	Latest  *github.RepositoryCommit `json:"latest"`
	Current *github.RepositoryCommit `json:"current"`
}

// FetchCommit fetches a specific commit by its Git reference
func FetchCommit(repo, accessToken, gitRef string) (*github.RepositoryCommit, error) {
	owner, repoName, _ := strings.Cut(repo, "/")
	client := github.NewClient(nil).WithAuthToken(accessToken)
	ctx := context.Background()
	commit, _, err := client.Repositories.GetCommit(ctx, owner, repoName, gitRef, nil)
	if err != nil {
		return nil, err
	}
	return commit, nil
}

// FetchLatestCommit fetches the latest commit from the repository
func FetchLatestCommit(repo, accessToken string) (*github.RepositoryCommit, error) {
	owner, repoName, _ := strings.Cut(repo, "/")
	client := github.NewClient(nil).WithAuthToken(accessToken)
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
func FetchCommits(repo, accessToken, gitRef string) (*StandardizedOutput, error) {
	// Fetch the current commit
	currentCommit, err := FetchCommit(repo, accessToken, gitRef)
	if err != nil {
		log.Printf("Error fetching current commit: %v", err)
		currentCommit = nil // Allow partial results
	}

	// Fetch the latest commit
	latestCommit, err := FetchLatestCommit(repo, accessToken)
	if err != nil {
		log.Printf("Error fetching latest commit: %v", err)
		latestCommit = nil // Allow partial results
	}

	// Standardize the commit response
	return &StandardizedOutput{
		Latest:  StandardizeCommit(latestCommit),
		Current: StandardizeCommit(currentCommit),
	}, nil
}

// CommitsHandler handles the API endpoint for fetching the latest and current commits
func CommitsHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef") // Get the gitRef parameter
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

	// Initialize accessToken as empty for optional authentication
	var accessToken string

	// Check if private key path is defined for authentication
	if privateKeyPath != "" {
		privateKey, err := LoadPrivateKey(privateKeyPath)
		if err != nil {
			log.Println("WARNING: Failed to load private key. Falling back to unauthenticated mode.")
		} else {
			// Generate JWT for GitHub App
			jwtToken, err := GenerateJWT(privateKey)
			if err != nil {
				log.Println("WARNING: Failed to generate JWT. Falling back to unauthenticated mode.")
			} else {
				// Get installation token using the JWT
				accessToken, err = GetInstallationToken(jwtToken, repo)
				if err != nil {
					log.Println(err)
					log.Println("WARNING: Failed to get installation token. Falling back to unauthenticated mode.")
					accessToken = ""
				}
			}
		}
	}

	// Fetch the commits
	commits, err := FetchCommits(repo, accessToken, gitRef)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Respond with the merged commits
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}
