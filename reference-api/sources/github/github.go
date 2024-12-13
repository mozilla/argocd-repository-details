package github

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/go-github/v67/github"
)

var (
	appID          = os.Getenv("GITHUB_APP_ID")           // Your GitHub App ID
	privateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH") // Path to the private key file
)

// ErrorResponse represents an error message
type ErrorResponse struct {
	Error string `json:"error"`
}

func errorEncoder(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{Error: message})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func responseEncoder(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// LoadPrivateKey loads the RSA private key from the file
func LoadPrivateKey(filePath string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(filePath)
	if err != nil {
		errMsg := "WARNING: Private key not found or could not be read (%s). Falling back to unauthenticated mode."
		log.Printf(errMsg, err.Error())
		return nil, nil // Continue without breaking
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		log.Printf("WARNING: Failed to parse private key (%s). Falling back to unauthenticated mode.", err.Error())
		return nil, nil // Continue without breaking
	}

	return privateKey, nil
}

// GenerateJWT generates a JWT for the GitHub App
func GenerateJWT(privateKey *rsa.PrivateKey) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"iss": appID,
	})
	return token.SignedString(privateKey)
}

// GetInstallationToken fetches an Installation Access Token
func GetInstallationToken(jwtToken string, repo string) (string, error) {
	client := github.NewClient(nil).WithAuthToken(jwtToken)
	owner, repoName, _ := strings.Cut(repo, "/")
	ctx := context.Background()
	installation, _, err := client.Apps.FindRepositoryInstallation(ctx, owner, repoName)
	if err != nil {
		return "", err
	}
	token, _, err := client.Apps.CreateInstallationToken(ctx, *installation.ID, nil)
	if err != nil {
		return "", err
	}
	return *token.Token, nil
}

func GenerateAuthToken(repo string) string {
	if privateKeyPath == "" {
		return ""
	}
	privateKey, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		log.Println("WARNING: Failed to load private key. Falling back to unauthenticated mode.")
		return ""
	}
	// Generate JWT for GitHub App
	jwtToken, err := GenerateJWT(privateKey)
	if err != nil {
		log.Println("WARNING: Failed to generate JWT. Falling back to unauthenticated mode.")
		return ""
	}
	// Get installation token using the JWT
	accessToken, err := GetInstallationToken(jwtToken, repo)
	if err != nil {
		log.Println(err)
		log.Println("WARNING: Failed to get installation token. Falling back to unauthenticated mode.")
		return ""
	}
	return accessToken
}

func NewGithubClient(repo string) *github.Client {
	authToken := GenerateAuthToken(repo)
	client := github.NewClient(nil)
	if authToken == "" {
		log.Println("WARNING: Failed to get installation token. Returning unathenticated client.")
		return client
	}
	return client.WithAuthToken(authToken)

}
