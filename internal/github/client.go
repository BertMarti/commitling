package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
	// MaxRetries is how many times a failed request is repeated (so a page is
	// asked for at most 1+MaxRetries times).
	MaxRetries = 3
	// DefaultBaseDelay is the first exponential-backoff wait: 1 s, 2 s, 4 s.
	DefaultBaseDelay = time.Second
	// DefaultMaxWait is the longest wait commitling accepts, whether it comes
	// from the backoff or from Retry-After / X-RateLimit-Reset. A server that
	// asks for more is not retried: better to fail with a clear message than
	// to hang a scheduled workflow.
	DefaultMaxWait = 30 * time.Second
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

	// BaseDelay is the first backoff wait (default DefaultBaseDelay).
	BaseDelay time.Duration
	// MaxWait caps every wait between attempts (default DefaultMaxWait).
	MaxWait time.Duration
	// Sleep waits for d or until ctx ends; nil uses a real timer. Tests inject
	// a fake one.
	Sleep func(ctx context.Context, d time.Duration) error
	// Now gives the current time to interpret X-RateLimit-Reset and a
	// Retry-After date; nil uses time.Now.
	Now func() time.Time
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
	// Retries is how many times the request was repeated before giving up.
	Retries int
	// RateLimited is set on a 403 that carries rate-limit headers (a plain
	// 403 is a permissions problem).
	RateLimited bool
}

// IsTransient reports whether err is a failure that may pass by itself: no
// answer at all (network), a server error (5xx) or rate limiting (429, or 403
// with rate-limit headers). A missing user (404), bad credentials (401), 422,
// a plain 403 or unparsable data will fail the same way tomorrow.
func IsTransient(err error) bool {
	var apiErr *APIError
	var netErr *netError
	switch {
	case errors.As(err, &apiErr):
		return apiErr.Status >= 500 || apiErr.Status == http.StatusTooManyRequests || apiErr.RateLimited
	case errors.As(err, &netErr):
		return true
	}
	return false
}

// rateLimitHeaders reports whether the headers of a 403 say it is rate limiting.
func rateLimitHeaders(hdr http.Header) bool {
	return hdr.Get("Retry-After") != "" || hdr.Get("X-RateLimit-Remaining") == "0"
}

func (e *APIError) Error() string { return e.baseError() + retriesSuffix(e.Retries) }

func (e *APIError) baseError() string {
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
	return Dedupe(all), nil
}

// netError is a failure to get any answer at all (DNS, connection refused or
// reset, timeout). It is worth repeating: the requests are all GETs.
type netError struct {
	err     error
	Retries int
}

func (e *netError) Error() string {
	return e.err.Error() + retriesSuffix(e.Retries)
}

func (e *netError) Unwrap() error { return e.err }

func retriesSuffix(n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return " (tras 1 reintento)"
	default:
		return fmt.Sprintf(" (tras %d reintentos)", n)
	}
}

// fetchPage asks for one page, repeating the request up to MaxRetries times
// on 500, 502, 503 and 504, on rate limiting (429, or 403 with rate-limit
// headers) and on network errors (unless the context has ended). Anything
// else (404, 422, other 4xx) is final.
func (c *Client) fetchPage(ctx context.Context, user string, page int) ([]Event, error) {
	for attempt := 0; ; attempt++ {
		events, hdr, err := c.doPage(ctx, user, page)
		if err == nil {
			return events, nil
		}
		var (
			apiErr *APIError
			netErr *netError
			wait   time.Duration
			ok     bool
		)
		switch {
		case errors.As(err, &apiErr):
			apiErr.Retries = attempt
			wait, ok = c.retryWait(apiErr.Status, hdr, attempt)
		case errors.As(err, &netErr) && ctx.Err() == nil:
			netErr.Retries = attempt
			wait, ok = c.backoff(attempt), true
		default:
			return nil, err
		}
		if attempt >= MaxRetries || !ok {
			return nil, err
		}
		if serr := c.sleep(ctx, wait); serr != nil {
			return nil, fmt.Errorf("%w; reintento interrumpido: %v", err, serr)
		}
	}
}

// backoff is the exponential wait before retry number attempt+1: 1x, 2x, 4x.
func (c *Client) backoff(attempt int) time.Duration {
	base := c.BaseDelay
	if base <= 0 {
		base = DefaultBaseDelay
	}
	wait := base << uint(attempt)
	if wait > c.maxWait() || wait <= 0 {
		wait = c.maxWait()
	}
	return wait
}

func (c *Client) maxWait() time.Duration {
	if c.MaxWait > 0 {
		return c.MaxWait
	}
	return DefaultMaxWait
}

// retryWait reports whether a response with this status and headers is worth
// repeating and how long to wait first.
func (c *Client) retryWait(status int, hdr http.Header, attempt int) (time.Duration, bool) {
	limited := false
	switch status {
	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
	case http.StatusTooManyRequests:
		limited = true
	case http.StatusForbidden:
		// A plain 403 is a permissions problem; only rate limiting is retried.
		limited = rateLimitHeaders(hdr)
		if !limited {
			return 0, false
		}
	default:
		return 0, false
	}

	if hint, ok := c.serverHint(hdr, limited); ok {
		if hint > c.maxWait() {
			return 0, false // the server asks for more than we are willing to wait
		}
		return hint, true
	}
	return c.backoff(attempt), true
}

// serverHint reads how long the server asks to wait: Retry-After (seconds or
// an HTTP date) or, when rate limited, X-RateLimit-Reset (Unix seconds).
func (c *Client) serverHint(hdr http.Header, limited bool) (time.Duration, bool) {
	if v := strings.TrimSpace(hdr.Get("Retry-After")); v != "" {
		if secs, err := strconv.ParseInt(v, 10, 64); err == nil && secs >= 0 {
			return secondsToDuration(secs), true
		}
		if t, err := http.ParseTime(v); err == nil {
			return positive(t.Sub(c.now())), true
		}
	}
	if limited {
		if v := strings.TrimSpace(hdr.Get("X-RateLimit-Reset")); v != "" {
			if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
				now := c.now()
				if unix > now.Unix() && unix-now.Unix() > 1<<31 {
					return secondsToDuration(unix - now.Unix()), true // absurdly far: avoid time overflow
				}
				return positive(time.Unix(unix, 0).Sub(now)), true
			}
		}
	}
	return 0, false
}

// secondsToDuration converts without overflowing: anything beyond what a
// Duration can hold is "a very long time".
func secondsToDuration(secs int64) time.Duration {
	if secs > int64(math.MaxInt64/time.Second) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(secs) * time.Second
}

func positive(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) sleep(ctx context.Context, d time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) doPage(ctx context.Context, user string, page int) ([]Event, http.Header, error) {
	u := fmt.Sprintf("%s/users/%s/events/public?per_page=%d&page=%d",
		strings.TrimRight(c.BaseURL, "/"), url.PathEscape(user), perPage, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, &netError{err: fmt.Errorf("no se pudo contactar con la API de GitHub: %w", err)}
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
		return nil, resp.Header, &APIError{Status: resp.StatusCode, Message: msg.Message,
			RateLimited: resp.StatusCode == http.StatusForbidden && rateLimitHeaders(resp.Header)}
	}
	events, err := ParseEvents(resp.Body)
	return events, resp.Header, err
}
