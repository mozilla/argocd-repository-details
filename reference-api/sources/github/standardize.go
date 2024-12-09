package github

import "github.com/google/go-github/v67/github"

// Create a standardized struct for Commits and Releases
type StandardizedOutput struct {
	Latest  *StandardizedEntity `json:"latest"`
	Current *StandardizedEntity `json:"current"`
}

type StandardizedEntity struct {
	Ref         string `json:"ref"`          // Commit SHA or Release Tag
	URL         string `json:"url"`          // Commit URL or Release URL
	Message     string `json:"message"`      // Commit Message or Release Body
	Author      string `json:"author"`       // Commit Author Login or Release Author Login
	PublishedAt string `json:"published_at"` // Commit Date or Release Published At
}

// StandardizeCommit converts a RepositoryCommit into a StandardizedEntity
func StandardizeCommit(commit *github.RepositoryCommit) *StandardizedEntity {
	if commit == nil {
		return nil
	}

	return &StandardizedEntity{
		Ref:         *commit.SHA,
		URL:         *commit.HTMLURL,
		Message:     *commit.Commit.Message,
		Author:      *commit.Author.Login,
		PublishedAt: commit.Commit.Author.Date.String(),
	}
}

// StandardizeRelease converts a Release into a StandardizedEntity
func StandardizeRelease(release *github.RepositoryRelease) *StandardizedEntity {
	if release == nil {
		return nil
	}

	return &StandardizedEntity{
		Ref:         *release.TagName,
		URL:         *release.HTMLURL,
		Message:     *release.Body,
		Author:      *release.Author.Login,
		PublishedAt: release.PublishedAt.String(),
	}
}
