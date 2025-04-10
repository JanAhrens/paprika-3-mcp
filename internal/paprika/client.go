package paprika

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// userAgent is required by the Paprika API
	userAgent = "paprika-3-mcp/1.0 (golang; " + runtime.Version() + ")"
)

// roundTripper is a wrapper around http.RoundTripper
// that adds the specified headers to each request
type roundTripper struct {
	headers   map[string]string
	transport http.RoundTripper
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range r.headers {
		// Only set if not already present
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	return r.transport.RoundTrip(req)
}

func NewClient(username, password string, logger *slog.Logger) (*Client, error) {
	// Create the http client & login to retrieve an authentication token
	t := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return d.DialContext(ctx, network, addr)
		},
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	token, err := login(ctx, *client, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	client.Transport = &roundTripper{
		transport: t,
		headers: map[string]string{
			"Accept":        "*/*",
			"Authorization": fmt.Sprintf("Bearer %s", token),
			"Connection":    "keep-alive",
			"User-Agent":    userAgent,
		},
	}

	l := logger
	if l == nil {
		l = slog.Default()
	}

	return &Client{
		client: client,
		logger: l,
	}, nil
}

type Client struct {
	client *http.Client
	logger *slog.Logger
}

type loginResponse struct {
	Result struct {
		Token string `json:"token"`
	} `json:"result"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// login authenticates with the Paprika API and returns an authentication token
// The token is used for all subsequent requests to the API. As far as I can tell, this is a JWT with no expiration.
func login(ctx context.Context, client http.Client, username, password string) (string, error) {
	body := fmt.Sprintf("email=%s&password=%s", username, password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://paprikaapp.com/api/v1/account/login", bytes.NewBufferString(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to login: %s", resp.Status)
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var loginResp loginResponse
	if err := json.Unmarshal(rawBytes, &loginResp); err != nil {
		return "", err
	}

	if loginResp.Result.Token == "" {
		return "", fmt.Errorf("failed to get token: %s", string(rawBytes))
	}

	return loginResp.Result.Token, nil
}

type RecipeList struct {
	Result []struct {
		UID  string `json:"uid"`
		Hash string `json:"hash"`
	} `json:"result"`
}

// ListRecipes retrieves a list of recipes from the Paprika API - the response objects
// only contain the UID and hash of each recipe, not the full recipe object
func (c *Client) ListRecipes(ctx context.Context) (*RecipeList, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://paprikaapp.com/api/v2/sync/recipes", nil)
	if err != nil {
		c.logger.Error("failed to create request", "error", err)
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("failed to get recipes", "error", err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to get recipes", "status", resp.Status)
		return nil, fmt.Errorf("failed to get recipes: %s", resp.Status)
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err)
		return nil, err
	}
	var recipeList RecipeList
	if err := json.Unmarshal(rawBytes, &recipeList); err != nil {
		c.logger.Error("failed to unmarshal response", "error", err)
		return nil, err
	}

	c.logger.Info("got recipes", "count", len(recipeList.Result))
	return &recipeList, nil
}

type Recipe struct {
	UID             string   `json:"uid"`
	Name            string   `json:"name"`
	Ingredients     string   `json:"ingredients"`
	Directions      string   `json:"directions"`
	Description     string   `json:"description"`
	Notes           string   `json:"notes"`
	NutritionalInfo string   `json:"nutritional_info"`
	Servings        string   `json:"servings"`
	Difficulty      string   `json:"difficulty"`
	PrepTime        string   `json:"prep_time"`
	CookTime        string   `json:"cook_time"`
	TotalTime       string   `json:"total_time"`
	Source          string   `json:"source"`
	SourceURL       string   `json:"source_url"`
	ImageURL        string   `json:"image_url"`
	Photo           string   `json:"photo"`
	PhotoHash       string   `json:"photo_hash"`
	PhotoLarge      string   `json:"photo_large"`
	Scale           string   `json:"scale"`
	Hash            string   `json:"hash"`
	Categories      []string `json:"categories"`
	Rating          int      `json:"rating"`
	InTrash         bool     `json:"in_trash"`
	IsPinned        bool     `json:"is_pinned"`
	OnFavorites     bool     `json:"on_favorites"`
	OnGroceryList   bool     `json:"on_grocery_list"`
	Created         string   `json:"created"`
	PhotoURL        string   `json:"photo_url"`
}

func (r *Recipe) generateUUID() {
	// Generate a new UUID for the recipe
	if r.UID == "" {
		r.UID = strings.ToUpper(uuid.New().String())
		return
	}

	r.UID = strings.ToUpper(r.UID)
}

func (r *Recipe) updateCreated() {
	layout := "2006-01-02 15:04:05"
	r.Created = time.Now().Format(layout)
}

func (r *Recipe) asMap() (map[string]interface{}, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

func (r *Recipe) updateHash() error {
	fields, err := r.asMap()
	if err != nil {
		return err
	}

	// Remove the "hash" field
	delete(fields, "hash")

	// Sort keys manually to ensure consistent JSON output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build a sorted map for consistent hashing
	sorted := make(map[string]interface{}, len(fields))
	for _, k := range keys {
		sorted[k] = fields[k]
	}

	// Marshal the sorted map to JSON
	jsonBytes, err := json.Marshal(sorted)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(jsonBytes)
	r.Hash = hex.EncodeToString(hash[:])
	return nil
}

func (r *Recipe) asGzip() ([]byte, error) {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err = writer.Write(jsonBytes)
	if err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type GetRecipeResponse struct {
	Result Recipe `json:"result"`
}

func (c *Client) GetRecipe(ctx context.Context, uid string) (*Recipe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://paprikaapp.com/api/v2/sync/recipe/%s/", uid), nil)
	if err != nil {
		c.logger.Error("failed to create request", "error", err)
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("failed to get recipe", "error", err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to get recipe", "status", resp.Status)
		return nil, fmt.Errorf("failed to get recipe: %s", resp.Status)
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err)
		return nil, err
	}

	var recipeResp GetRecipeResponse
	if err := json.Unmarshal(rawBytes, &recipeResp); err != nil {
		c.logger.Error("failed to unmarshal response", "error", err)
		return nil, err
	}

	return &recipeResp.Result, nil
}

func (c *Client) DeleteRecipe(ctx context.Context, recipe Recipe) (*Recipe, error) {
	// Set the recipe to be in the trash
	// TODO: reverse-engineer full deletions; currently a user must go in-app to empty their trash and fully delete something
	recipe.InTrash = true
	return c.SaveRecipe(ctx, recipe)
}

// SaveRecipe saves a recipe to the Paprika API. If the recipe already exists, it will be updated.
// If the recipe does not exist, it will be created.
func (c *Client) SaveRecipe(ctx context.Context, recipe Recipe) (*Recipe, error) {
	// set the created timestamp
	recipe.updateCreated()
	// generate a new UUID if one doesn't exist
	recipe.generateUUID()
	// generate a hash of the recipe object
	if err := recipe.updateHash(); err != nil {
		return nil, err
	}

	// gzip the recipe
	fileData, err := recipe.asGzip()
	if err != nil {
		return nil, err
	}

	// Create a multipart form request
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("data", "data")
	if err != nil {
		c.logger.Error("failed to create form file", "error", err)
		return nil, err
	}

	// Write the gzipped JSON data to the form file
	if _, err := part.Write(fileData); err != nil {
		c.logger.Error("failed to write gzipped JSON data", "error", err)
		return nil, err
	}
	if err := writer.Close(); err != nil {
		c.logger.Error("failed to close multipart writer", "error", err)
		return nil, err
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://paprikaapp.com/api/v2/sync/recipe/%s/", recipe.UID), &body)
	if err != nil {
		c.logger.Error("failed to create request", "error", err)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ContentLength = int64(body.Len())

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("failed to create recipe", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to create recipe", "status", resp.Status)
		return nil, fmt.Errorf("failed to create recipe: %s", resp.Status)
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err)
		return nil, err
	}

	if err := isErrorResponse(rawBytes); err != nil {
		c.logger.Error("failed to create recipe", "error", err)
		return nil, err
	}

	defer c.notify(ctx)

	return &recipe, nil
}

// notify sends a POST to /v2/sync/notify, which tells all Paprika clients to sync.
// We usually defer this call after a recipe is created/updated/deleted, since we don't care whether it suceeds or not.
func (c *Client) notify(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://paprikaapp.com/api/v2/sync/notify", nil)
	if err != nil {
		c.logger.Error("failed to create request", "error", err)
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("failed to notify", "error", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to notify", "status", resp.Status)
		return fmt.Errorf("failed to notify: %s", resp.Status)
	}

	return nil
}

// isErrorResponse checks if the response body contains an error message
// and returns an error if it does. The Paprika API is very inconsistent with how it returns errors;
// sometimes a successful status code can be returned but an error is still returned in the body
func isErrorResponse(body []byte) error {
	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// Not even valid JSON
		return err
	}

	// Check if it's likely an error response
	if errResp.Error.Message != "" || errResp.Error.Code != 0 {
		return fmt.Errorf("error: %s (code: %d)", errResp.Error.Message, errResp.Error.Code)
	}

	return nil
}
