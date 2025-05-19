package tfa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type orgAdd struct {
	Delta struct {
		Users map[string]string `json:"users"`
	} `json:"delta"`
}

type orgAccess struct {
	Users []struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Ref      string `json:"ref"`
		Access   string `json:"access"`
		IsMember bool   `json:"isMember"`
	} `json:"users"`
}

type Grist struct {
	mu        sync.Mutex
	port      int
	baseUrl   string
	apiKey    string
	orgName   string
	adminMail string
	knownOrg  orgAccess
}

// NewGrist creates a new grist api object
func NewGrist(port int, apiKey, orgName, adminMail string) *Grist {
	g := &Grist{port: port, apiKey: apiKey, orgName: orgName, adminMail: adminMail}
	g.baseUrl = "http://127.0.0.1"
	orgs, err := g.getOrgs()
	if err != nil {
		return g
	}
	g.mu.Lock()
	g.knownOrg = *orgs
	g.mu.Unlock()
	return g
}

func (g *Grist) AddToOrgWithCheck(email string) error {
	if email == g.adminMail {
		return nil
	}
	// check in known users
	for _, org := range g.knownOrg.Users {
		if org.Email == email {
			return nil
		}
	}
	orgs, err := g.getOrgs()
	if err != nil {
		return err
	}
	g.mu.Lock()
	g.knownOrg = *orgs
	g.mu.Unlock()
	// refresh and check in known users
	for _, org := range g.knownOrg.Users {
		if org.Email == email {
			return nil
		}
	}
	_, err = g.updateOrgAccess(email)
	if err != nil {
		return err
	}
	return nil
}

func (g *Grist) getOrgs() (*orgAccess, error) {
	// Create the request URL
	url := fmt.Sprintf("%s:%d/api/orgs/%s/access", g.baseUrl, g.port, g.orgName)
	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	// Configure HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode the response
	var access orgAccess
	if err = json.NewDecoder(resp.Body).Decode(&access); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &access, nil
}

func (g *Grist) updateOrgAccess(email string) (*http.Response, error) {
	// Prepare the request payload
	var requestData orgAdd

	requestData.Delta.Users = map[string]string{email: "editors"}
	// Marshal the struct to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Create the request URL
	url := fmt.Sprintf("%s:%d/api/orgs/%s/access", g.baseUrl, g.port, g.orgName)

	// Create the HTTP request
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("accept", "*/*")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Configure HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	return resp, nil
}
