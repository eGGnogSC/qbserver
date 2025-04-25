// qbclient/client.go
package qbclient

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "strings"
    "time"
    
    "github.com/eGGnogSC/qbserver/auth"
)

// Client is the main QuickBooks API client
type Client struct {
    baseURL      string
    clientID     string
    clientSecret string
    authService  *auth.Service
    userID       string
    realmID      string
    httpClient   *http.Client
}

// NewClient creates a new QuickBooks API client
func NewClient(baseURL, clientID, clientSecret string, authService *auth.Service) *Client {
    return &Client{
        baseURL:      baseURL,
        clientID:     clientID,
        clientSecret: clientSecret,
        authService:  authService,
        httpClient:   &http.Client{Timeout: 30 * time.Second},
    }
}

// WithUser sets the user context for the client
func (c *Client) WithUser(userID string) *Client {
    client := *c
    client.userID = userID
    return &client
}

// WithRealmID sets the QuickBooks company ID for the client
func (c *Client) WithRealmID(realmID string) *Client {
    client := *c
    client.realmID = realmID
    return &client
}

// sendRequest makes an authenticated request to the QuickBooks API
func (c *Client) sendRequest(ctx context.Context, method, endpoint string, body []byte) (*http.Response, error) {
    // If userID is not set, try to get it from context
    userID := c.userID
    if userID == "" {
        userID = auth.GetUserID(ctx)
        if userID == "" {
            return nil, fmt.Errorf("user ID not provided")
        }
    }
    
    // If realmID is not set, try to get it from context
    realmID := c.realmID
    if realmID == "" {
        var err error
        realmID, err = auth.GetCompanyID(ctx)
        if err != nil {
            return nil, fmt.Errorf("company ID not provided")
        }
    }
    
    // Get valid token
    token, err := c.authService.GetValidToken(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get valid token: %w", err)
    }
    
    // Create request
    var reqBody *strings.Reader
    if body != nil {
        reqBody = strings.NewReader(string(body))
    }
    
    req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    // Set headers
    req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
    req.Header.Set("Accept", "application/json")
    
    if method == "POST" || method == "PUT" {
        req.Header.Set("Content-Type", "application/json")
    }
    
    // Add minor version
    query := req.URL.Query()
    query.Set("minorversion", "75") // Using the latest minor version
    req.URL.RawQuery = query.Encode()
    
    // Send request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    
    // Check for error responses
    if resp.StatusCode >= 400 {
        defer resp.Body.Close()
        body, _ := ioutil.ReadAll(resp.Body)
        
        var qbErr struct {
            Fault struct {
                Error []struct {
                    Message string `json:"Message"`
                    Code    string `json:"code"`
                } `json:"Error"`
            } `json:"Fault"`
        }
        
        if err := json.Unmarshal(body, &qbErr); err == nil && len(qbErr.Fault.Error) > 0 {
            return nil, fmt.Errorf("QuickBooks API error (%s): %s", 
                qbErr.Fault.Error[0].Code, qbErr.Fault.Error[0].Message)
        }
        
        return nil, fmt.Errorf("QuickBooks API returned status %d: %s", 
            resp.StatusCode, string(body))
    }
    
    return resp, nil
}
