module github.com/mozilla/argocd-repository-details/reference-api

go 1.22.3

require (
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/go-github/v67 v67.0.1-0.20241202213040-cea0bba46cd1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mozilla/argocd-repository-details/reference-api/sources/github => ./sources/github
