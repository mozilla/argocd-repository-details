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

// StandardizeTag converts a Tag and its associated Commit into a StandardizedEntity
func StandardizeTag(tag *github.RepositoryTag, commit *github.RepositoryCommit) *StandardizedEntity {
	if tag == nil {
		return nil
	}

	// If we have the commit details, use them for richer information
	if commit != nil && commit.Commit != nil {
		author := ""
		if commit.Author != nil && commit.Author.Login != nil {
			author = *commit.Author.Login
		}

		message := ""
		if commit.Commit.Message != nil {
			message = *commit.Commit.Message
		}

		publishedAt := ""
		if commit.Commit.Author != nil && commit.Commit.Author.Date != nil {
			publishedAt = commit.Commit.Author.Date.String()
		}

		url := ""
		if commit.HTMLURL != nil {
			url = *commit.HTMLURL
		}

		return &StandardizedEntity{
			Ref:         *tag.Name,
			URL:         url,
			Message:     message,
			Author:      author,
			PublishedAt: publishedAt,
		}
	}

	// Fallback to basic tag info if commit details are unavailable
	return &StandardizedEntity{
		Ref:         *tag.Name,
		URL:         "",
		Message:     "",
		Author:      "",
		PublishedAt: "",
	}
}
