# Releasing Guide

This document describes the process for releasing new versions of `argocd-repository-details`.

## Overview

1. Create a GitHub release (triggers Docker build and PRs for both sandbox and prod)
2. Merge sandbox PR and manually sync in Argo CD
3. Validate in sandbox
4. Merge production PR
5. Production auto-syncs

## Prerequisites

- Write access to this repository
- Access to merge PRs in [mozilla/global-platform-admin](https://github.com/mozilla/global-platform-admin)
- Access to Argo CD environments for manual verification:
  - [sandbox](https://sandbox.argocd.global.mozgcp.net/)
  - [webservices](https://webservices.argocd.global.mozgcp.net/)
  - [dataservices](https://dataservices.argocd.global.mozgcp.net/)

## Release Process

### 1. Merge Changes

Merge all PRs intended for the release into the `main` branch.

### 2. Create GitHub Release

1. Go to https://github.com/mozilla/argocd-repository-details/releases
2. Click "Draft a new release"
3. Create a new tag (e.g., `v1.2.3`) following semantic versioning
4. Publish the release

This triggers:
- Docker image build
- [dispatch-gpa-update.yaml](.github/workflows/dispatch-gpa-update.yaml) workflow that creates sandbox and prod PRs in `mozilla/global-platform-admin`

### 3. Deploy to Sandbox Environment

1. Find the PR in [global-platform-admin](https://github.com/mozilla/global-platform-admin/pulls) titled `chore: Update argocd-repository-details Sandbox to v1.2.3`
2. Review changes
3. Merge the PR
4. Manually sync the [argocd-repository-details-bootstrap](https://sandbox.argocd.global.mozgcp.net/applications/argocd/argocd-repository-details-bootstrap) application in Argo CD
5. Verify the extension loads correctly

### 4. Deploy to Production Environments

The production PR is created automatically when you publish the release (same workflow as sandbox).

1. Find the PR in [global-platform-admin](https://github.com/mozilla/global-platform-admin/pulls) titled `chore: Update argocd-repository-details Prod to v1.2.3`
2. Review changes
3. Merge the PR
4. Production environments auto-sync automatically
5. Verify in [webservices](https://webservices.argocd.global.mozgcp.net/applications/argocd-webservices/argocd-repository-details-webservices) and [dataservices](https://dataservices.argocd.global.mozgcp.net/applications/argocd-dataservices/argocd-repository-details-dataservices)

Manual trigger (if needed):
```bash
gh workflow run argocd-repository-details.yaml -R mozilla/global-platform-admin
```

## Automation Details

**Workflow**: [`argocd-repository-details.yaml`](https://github.com/mozilla/global-platform-admin/blob/main/.github/workflows/argocd-repository-details.yaml) in global-platform-admin uses a matrix strategy for `sandbox` and `prod` environments.

**Triggers**:
- Publishing a release: [dispatch-gpa-update.yaml](.github/workflows/dispatch-gpa-update.yaml) sends `repository_dispatch` event that creates both sandbox and prod PRs
- Daily cron (noon UTC): Creates PRs if new versions are available
- Manual dispatch

**Manual trigger**:
```bash
# Latest version
gh workflow run argocd-repository-details.yaml -R mozilla/global-platform-admin

# Specific version
gh workflow run argocd-repository-details.yaml -R mozilla/global-platform-admin -f version=v1.2.3
```

## Troubleshooting

**PRs not created after release:**
1. Check [dispatch-gpa-update.yaml runs](../../actions/workflows/dispatch-gpa-update.yaml)
2. Verify `REPOSITORY_DETAILS_DISPATCHER_PAT` secret exists
3. Check [workflow runs](https://github.com/mozilla/global-platform-admin/actions/workflows/argocd-repository-details.yaml) in global-platform-admin
4. Confirm version isn't already deployed (workflow skips if up-to-date)
5. Manually trigger the workflow

