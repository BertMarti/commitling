package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DefaultBaseURL is the public GitHub REST API.
const DefaultBaseURL = "https://api.github.com"

const (
	// DefaultUserAgent identifies the tool; GitHub requires a User-Agent.
	DefaultUserAgent = "commitling"
	// APIVersion is the pinned REST API version.
	APIVersion = "2022-11-28"
	// RequestTimeout bounds each HTTP request of the default client.
	RequestTimeout = 15 * time.Second
)

const (
	perPage = 100
	// maxPages * perPage is the 300 events limit of the public events API.
	maxPages = 3
)

var loginRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)

// ValidLogin reports whether s looks like a GitHub username.
func ValidLogin(s string) bool { return loginRe.MatchString(s) }

// Client fetches public events. The zero value is not usable; use NewClient.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient returns a client for the public API. token may be empty.
func NewClient(token string) *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		Token:      token,
		HTTPClient: &http.Client{Timeout: RequestTimeout},
		UserAgent:  DefaultUserAgent,
	}
}

// APIError is a non-successful answer from the API.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	switch {
	case e.Status == http.StatusNotFound:
		return "usuario no encontrado en GitHub (404)"
	case e.Status == http.StatusForbidden || e.Status == http.StatusTooManyRequests:
		return fmt.Sprintf("la API de GitHub ha rechazado la petición (%d): %s; si es el límite de peticiones, define GITHUB_TOKEN", e.Status, e.Message)
	default:
		return fmt.Sprintf("la API de GitHub respondió %d: %s", e.Status, e.Message)
	}
}

// FetchEvents downloads up to 300 public events (the API maximum, about
// the last 90 days) of user, newest first.
func (c *Client) FetchEvents(ctx context.Context, user string) ([]Event, error) {
	if !ValidLogin(user) {
		return nil, fmt.Errorf("nombre de usuario no válido: %q", user)
	}
	var all []Event
	for page := 1; page <= maxPages; page++ {
		events, err := c.fetchPage(ctx, user, page)
		if err != nil {
			var apiErr *APIError
			// The API answers 422 when asking beyond its pagination limit.
			if page > 1 && errors.As(err, &apiErr) && apiErr.Status == http.StatusUnprocessableEntity {
				break
			}
			return nil, err
		}
		all = append(all, events...)
		if len(events) < perPage {
			break
		}
	}
	return all, nil
}

func (c *Client) fetchPage(ctx context.Context, user string, page int) ([]Event, error) {
	u := fmt.Sprintf("%s/users/%s/events/public?per_page=%d&page=%d",
		strings.TrimRight(c.BaseURL, "/"), url.PathEscape(user), perPage, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", APIVersion)
	ua := c.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: RequestTimeout}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo contactar con la API de GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var msg struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &msg)
		if msg.Message == "" {
			msg.Message = http.StatusText(resp.StatusCode)
		}
		return nil, &APIError{Status: resp.StatusCode, Message: msg.Message}
	}
	return ParseEvents(resp.Body)
}
