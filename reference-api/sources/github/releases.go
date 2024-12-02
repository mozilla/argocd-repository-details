package github

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type MergedReleases struct {
	Latest  *Release `json:"latest"`
	Current *Release `json:"current"`
}

// FetchReleases fetches the latest release and the one matching the Git reference (gitRef) from GitHub
func FetchReleases(repo, accessToken, gitRef string) (*MergedReleases, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/releases", githubAPIURL, repo)

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

	// Decode the response body into a slice of Release
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var latestRelease *Release
	var matchingRelease *Release

	// Iterate through releases to find the latest and the matching release
	for _, release := range releases {
		// Identify the latest release (first in the list, assuming GitHub returns them in descending order)
		if latestRelease == nil {
			latestRelease = &release
		}

		// Identify the release matching the gitRef
		if release.TagName == gitRef {
			matchingRelease = &release
		}

		// If both latest and matching releases are found, exit the loop early
		if latestRelease != nil && matchingRelease != nil {
			break
		}
	}

	// Return the merged object
	return &MergedReleases{
		Latest:  latestRelease,
		Current: matchingRelease,
	}, nil
}

// ReleasesHandler handles the API endpoint for fetching releases from target repositories
func ReleasesHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	gitRef := r.URL.Query().Get("gitRef")
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

	// Fetch the release by Git reference
	releases, err := FetchReleases(repo, accessToken, gitRef)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Respond with the matching release
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(releases)
}
