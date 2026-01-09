package pocketbase

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type AuthMethod struct {
	Name                string `json:"name"`
	State               string `json:"state"`
	CodeVerifier        string `json:"codeVerifier"`
	CodeChallenge       string `json:"codeChallenge"`
	CodeChallengeMethod string `json:"codeChallengeMethod"`
	AuthURL             string `json:"authUrl"`
}

type AuthMethodsList struct {
	AuthProviders []AuthMethod `json:"authProviders"`
}

type AuthResponse struct {
	Token  string `json:"token"`
	Record struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"record"`
}

func TestConnection(pbURL string) error {
	resp, err := http.Get(pbURL + "/api/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	log.Printf("PocketBase connected: %s (status=%d)", pbURL, resp.StatusCode)
	return nil
}

func GetAuthMethods(pbURL string) ([]AuthMethod, error) {
	resp, err := http.Get(pbURL + "/api/collections/users/auth-methods")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get auth methods: %d", resp.StatusCode)
	}

	var list AuthMethodsList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	return list.AuthProviders, nil
}

func AuthWithOAuth2(pbURL, provider, code, codeVerifier, redirectURL string) (*AuthResponse, error) {
	data := url.Values{}
	data.Set("provider", provider)
	data.Set("code", code)
	data.Set("codeVerifier", codeVerifier)
	data.Set("redirectUrl", redirectURL)

	resp, err := http.Post(pbURL+"/api/collections/users/auth-with-oauth2", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("auth failed: %d", resp.StatusCode)
	}

	var auth AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, err
	}

	return &auth, nil
}
