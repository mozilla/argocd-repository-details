 # argocd-repository-details

## Overview
This project provides an Argo CD extension to view application repository details in the Argo CD UI.
This is intended to serve as a way to link a git reference from your source application repository to your
Argo CD application and assumes your infrastructure and application repositories are separate.

## Demo

## Requirements

The Repository Details extension requires your Argo CD Applications to have an Info annotation with
the application repository. ex. ``

It currently only support GitHub releases and Commits as information sources.

## Installation

### Installing the backend (reference-api)


### Enabling the RepositoryDetails extension in Argo CD

Argo CD needs to have the proxy extension feature enabled for the
RepositoryDetails extension to work. In order to do so add the following
entry in the `argocd-cmd-params-cm`:

```
server.enable.proxy.extension: "true"
```

The RepositoryDetails extension needs to be authorized in Argo CD API server. To
enable it for all users add the following entry in `argocd-rbac-cm`:

```
policy.csv: |-
   p, role:authenticated, extensions, invoke, repository-details, allow
```

**Note**: make sure to assign a proper role to the extension policy if you
want to restrict the access.

Finally Argo CD needs to be configured so it knows how to reach the
RepositoryDetails backend service. In order to do so, add the following
section in the `argocd-cm`.

```yaml
    - name: repository-details
      backend:
        services:
        - url: REPOSITORY_DETAILS_BACKEND_URL
```

**Attention**: Make sure to change the `REPOSITORY_DETAILS_BACKEND_URL`
to the URL where backend service is configured. The backend service
URL needs to be reacheable by the Argo CD API server.

## Testing

### Start the backend service:
```
cd reference-api
go run .
```
Use curl to check the references endpoint:
```
  curl "http://localhost:8000/api/references?repo=dlactin/test&gitRef=0.0.1"

or

  curl "http://localhost:8000/api/references?repo=dlactin/test&gitRef=035552e"

```

### Create the ArgoCD UI extensions tar file

Run the makefile in the root of the repository after cloning
```
make
```

Copy the extension.tar file somewhere it can be consumed by an Argo CD instance. GCS/S3/etc
```
cp ui/extension.tar
```

