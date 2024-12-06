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
	"github.com/google/go-github/v67/github"
)

const (
	githubAPIURL = "https://api.github.com"
)

var (
	appID          = os.Getenv("GITHUB_APP_ID")           // Your GitHub App ID
	privateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH") // Path to the private key file
)

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

func GenerateAuthToken(repo string) string {
	// Initialize accessToken for optional authentication
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
					log.Println(err)
					log.Println("WARNING: Failed to get installation token. Falling back to unauthenticated mode.")
					accessToken = ""
				}
			}
		}
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
