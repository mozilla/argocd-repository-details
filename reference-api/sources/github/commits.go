package github

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Commit struct {
	SHA    string `json:"sha"`
	URL    string `json:"html_url"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

type MergedCommits struct {
	Latest  *Commit `json:"latest"`
	Current *Commit `json:"current"`
}

// FetchCommit fetches a specific commit by its Git reference
func FetchCommit(repo, accessToken, gitRef string) (*Commit, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/commits/%s", githubAPIURL, repo, gitRef)

	// Create the request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Add authorization header if accessToken is available
	if accessToken != "" {
		req.Header.Set("Authorization", "token "+accessToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	// Decode the response body into a Commit
	var commit Commit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return nil, err
	}

	return &commit, nil
}

// FetchLatestCommit fetches the latest commit from the repository
func FetchLatestCommit(repo, accessToken string) (*Commit, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/commits", githubAPIURL, repo)

	// Create the request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Add authorization header if accessToken is available
	if accessToken != "" {
		req.Header.Set("Authorization", "token "+accessToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	// Decode the response body into a slice of Commit
	var commits []Commit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}

	// Return the first commit in the list (latest commit)
	if len(commits) > 0 {
		return &commits[0], nil
	}

	return nil, fmt.Errorf("no commits found")
}

// FetchCommits fetches both the latest commit and the one matching the Git reference (gitRef)
func FetchCommits(repo, accessToken, gitRef string) (*MergedCommits, error) {
	// Fetch the current commit
	currentCommit, err := FetchCommit(repo, accessToken, gitRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the current commit: %w", err)
	}

	// Fetch the latest commit
	latestCommit, err := FetchLatestCommit(repo, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the latest commit: %w", err)
	}

	return &MergedCommits{
		Latest:  latestCommit,
		Current: currentCommit,
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
