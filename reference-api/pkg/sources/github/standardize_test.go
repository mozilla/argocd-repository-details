package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
)

func TestStandardizeTag(t *testing.T) {
	tests := []struct {
		name   string
		tag    *github.RepositoryTag
		commit *github.RepositoryCommit
		want   *StandardizedEntity
	}{
		{
			name: "Nil tag returns nil",
			tag:  nil,
			want: nil,
		},
		{
			name: "Tag with full commit details",
			tag: &github.RepositoryTag{
				Name: github.String("v1.0.0"),
			},
			commit: &github.RepositoryCommit{
				SHA:     github.String("abc123"),
				HTMLURL: github.String("https://github.com/test/repo/commit/abc123"),
				Author: &github.User{
					Login: github.String("testuser"),
				},
				Commit: &github.Commit{
					Message: github.String("Test commit message"),
					Author: &github.CommitAuthor{
						Date: &github.Timestamp{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)},
					},
				},
			},
			want: &StandardizedEntity{
				Ref:         "v1.0.0",
				URL:         "https://github.com/test/repo/commit/abc123",
				Message:     "Test commit message",
				Author:      "testuser",
				PublishedAt: "2025-01-01 12:00:00 +0000 UTC",
			},
		},
		{
			name: "Tag without commit details",
			tag: &github.RepositoryTag{
				Name: github.String("v2.0.0"),
			},
			commit: nil,
			want: &StandardizedEntity{
				Ref:         "v2.0.0",
				URL:         "",
				Message:     "",
				Author:      "",
				PublishedAt: "",
			},
		},
		{
			name: "Tag with partial commit details",
			tag: &github.RepositoryTag{
				Name: github.String("v3.0.0"),
			},
			commit: &github.RepositoryCommit{
				SHA: github.String("def456"),
				Commit: &github.Commit{
					Message: github.String("Partial commit"),
				},
			},
			want: &StandardizedEntity{
				Ref:         "v3.0.0",
				URL:         "",
				Message:     "Partial commit",
				Author:      "",
				PublishedAt: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StandardizeTag(tt.tag, tt.commit)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardizeCommit(t *testing.T) {
	tests := []struct {
		name   string
		commit *github.RepositoryCommit
		want   *StandardizedEntity
	}{
		{
			name:   "Nil commit returns nil",
			commit: nil,
			want:   nil,
		},
		{
			name: "Complete commit",
			commit: &github.RepositoryCommit{
				SHA:     github.String("abc123def456"),
				HTMLURL: github.String("https://github.com/test/repo/commit/abc123"),
				Author: &github.User{
					Login: github.String("commitauthor"),
				},
				Commit: &github.Commit{
					Message: github.String("Fix bug in handler"),
					Author: &github.CommitAuthor{
						Date: &github.Timestamp{Time: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)},
					},
				},
			},
			want: &StandardizedEntity{
				Ref:         "abc123def456",
				URL:         "https://github.com/test/repo/commit/abc123",
				Message:     "Fix bug in handler",
				Author:      "commitauthor",
				PublishedAt: "2025-06-15 10:30:00 +0000 UTC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StandardizeCommit(tt.commit)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardizeRelease(t *testing.T) {
	tests := []struct {
		name    string
		release *github.RepositoryRelease
		want    *StandardizedEntity
	}{
		{
			name:    "Nil release returns nil",
			release: nil,
			want:    nil,
		},
		{
			name: "Complete release",
			release: &github.RepositoryRelease{
				TagName: github.String("v1.5.0"),
				HTMLURL: github.String("https://github.com/test/repo/releases/tag/v1.5.0"),
				Body:    github.String("## What's New\n- Feature A\n- Feature B"),
				Author: &github.User{
					Login: github.String("releaseauthor"),
				},
				PublishedAt: &github.Timestamp{Time: time.Date(2025, 12, 1, 14, 0, 0, 0, time.UTC)},
			},
			want: &StandardizedEntity{
				Ref:         "v1.5.0",
				URL:         "https://github.com/test/repo/releases/tag/v1.5.0",
				Message:     "## What's New\n- Feature A\n- Feature B",
				Author:      "releaseauthor",
				PublishedAt: "2025-12-01 14:00:00 +0000 UTC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StandardizeRelease(tt.release)
			assert.Equal(t, tt.want, got)
		})
	}
}
