package github

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

const (
	githubAPIURL = "https://api.github.com"
)

var (
	appID          = os.Getenv("GITHUB_APP_ID")           // Your GitHub App ID
	privateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH") // Path to the private key file
)

// Release represents a GitHub release
type Release struct {
	TagName     string `json:"tag_name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
}

// ErrorResponse represents an error message
type ErrorResponse struct {
	Error string `json:"error"`
}

// LoadPrivateKey loads the RSA private key from the file
func LoadPrivateKey(filePath string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("WARNING: Private key not found or could not be read (%s). Falling back to unauthenticated mode.", err.Error())
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
	installationURL := fmt.Sprintf("%s/repos/%s/installation", githubAPIURL, repo)
	request, err := http.NewRequest("GET", installationURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+jwtToken)
	request.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get installation ID, status: %s", resp.Status)
	}

	var installation struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&installation); err != nil {
		return "", err
	}

	// Get the access token
	tokenURL := fmt.Sprintf("%s/app/installations/%d/access_tokens", githubAPIURL, installation.ID)
	request, err = http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+jwtToken)
	request.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err = client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create installation token, status: %s", resp.Status)
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	return tokenResp.Token, nil
}
