package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v67/github"
)

// FetchTags fetches information about a specific git tag and the latest tag
func FetchTags(repo, gitRef string) (*StandardizedOutput, error) {
	owner, repoName, _ := strings.Cut(repo, "/")
	client := NewGithubClient(repo)
	ctx := context.Background()

	// List all tags
	tags, resp, err := client.Repositories.ListTags(ctx, owner, repoName, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("404 Not Found: No tags found for repo %s", repo)
		}
		return nil, fmt.Errorf("GitHub API error: %v", err)
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("404 Not Found: No tags found for repo %s", repo)
	}

	var matchingTag *github.RepositoryTag

	// Find the tag matching the gitRef
	for _, tag := range tags {
		tag := tag
		if *tag.Name == gitRef {
			matchingTag = tag
			break
		}
	}

	// If no matching tag found, return 404
	if matchingTag == nil {
		return nil, fmt.Errorf("404 Not Found: Tag %s not found in repo %s", gitRef, repo)
	}

	// Fetch commit details for matching tag
	matchingCommit, err := fetchCommitForTag(client, ctx, owner, repoName, matchingTag)
	if err != nil {
		log.Printf("Error: Failed to fetch commit for matching tag: %v", err)
		return nil, fmt.Errorf("failed to fetch commit for tag %s: %v", gitRef, err)
	}

	// Use unified latest that checks both releases and tags
	latest := FetchLatestReference(repo)

	return &StandardizedOutput{
		Latest:  latest,
		Current: StandardizeTag(matchingTag, matchingCommit),
	}, nil
}

// fetchCommitForTag fetches the commit that a tag points to
func fetchCommitForTag(
	client *github.Client, ctx context.Context, owner, repo string, tag *github.RepositoryTag,
) (*github.RepositoryCommit, error) {
	if tag == nil || tag.Commit == nil || tag.Commit.SHA == nil {
		return nil, fmt.Errorf("invalid tag data")
	}

	commit, _, err := client.Repositories.GetCommit(ctx, owner, repo, *tag.Commit.SHA, nil)
	if err != nil {
		return nil, err
	}

	return commit, nil
}

// TagsHandler handles the API endpoint for fetching git tags from target repositories
func TagsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Fetch tag information
	tags, err := FetchTags(repo, gitRef)
	if err != nil {
		log.Printf("Error fetching tags: %v", err)

		// If the error is due to GitHub returning a non-200 response, set the correct status
		if strings.Contains(err.Error(), "404") {
			errorEncoder(w, http.StatusNotFound, "Tag not found")
		} else {
			errorEncoder(w, http.StatusInternalServerError, "Failed to fetch tag information")
		}
		return
	}

	if tags == nil || tags.Current == nil {
		errorEncoder(w, http.StatusNotFound, "No tag found for the given repository and gitRef")
		return
	}

	responseEncoder(w, http.StatusOK, tags)
}
