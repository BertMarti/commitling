package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// recorder is a fake Sleep that records the waits instead of sleeping.
type recorder struct{ waits []time.Duration }

func (r *recorder) sleep(_ context.Context, d time.Duration) error {
	r.waits = append(r.waits, d)
	return nil
}

func newTestClient(url string) (*Client, *recorder) {
	rec := &recorder{}
	c := NewClient("")
	c.BaseURL = url
	c.Sleep = rec.sleep
	c.Now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	return c, rec
}

// flaky answers the given statuses (with headers) in order and then 200.
func flaky(t *testing.T, calls *atomic.Int32, headers http.Header, statuses ...int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1)) - 1
		if n < len(statuses) {
			for k, v := range headers {
				w.Header()[k] = v
			}
			w.WriteHeader(statuses[n])
			fmt.Fprint(w, `{"message":"boom"}`)
			return
		}
		fmt.Fprint(w, eventsJSON(2, 0))
	}))
}

func TestRetriesServerErrorsThenSucceeds(t *testing.T) {
	for _, status := range []int{500, 502, 503, 504} {
		var calls atomic.Int32
		srv := flaky(t, &calls, nil, status, status)
		c, rec := newTestClient(srv.URL)
		events, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		if err != nil {
			t.Fatalf("%d: %v", status, err)
		}
		if len(events) != 2 || calls.Load() != 3 {
			t.Fatalf("%d: events=%d calls=%d, want 2 and 3", status, len(events), calls.Load())
		}
		if fmt.Sprint(rec.waits) != "[1s 2s]" {
			t.Errorf("%d: waits = %v, want exponential [1s 2s]", status, rec.waits)
		}
	}
}

func TestGivesUpAfterThreeRetries(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, nil, 503, 503, 503, 503, 503, 503)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	_, err := c.FetchEvents(context.Background(), "octoexample")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 503 {
		t.Fatalf("err = %v, want APIError 503", err)
	}
	if calls.Load() != 4 || apiErr.Retries != 3 {
		t.Fatalf("calls=%d retries=%d, want 4 requests (1 + 3 retries)", calls.Load(), apiErr.Retries)
	}
	if fmt.Sprint(rec.waits) != "[1s 2s 4s]" {
		t.Errorf("waits = %v, want [1s 2s 4s]", rec.waits)
	}
	if !strings.Contains(err.Error(), "tras 3 reintentos") {
		t.Errorf("error %q should mention the retries", err)
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	cases := []struct {
		status  int
		headers http.Header
	}{
		{404, nil},
		{422, nil},
		{400, nil},
		{401, nil},
		{403, nil}, // plain 403: permissions, not rate limit
		{403, http.Header{"X-Ratelimit-Remaining": {"12"}}}, // limit not exhausted
		{501, nil},
	}
	for _, tc := range cases {
		var calls atomic.Int32
		srv := flaky(t, &calls, tc.headers, tc.status, tc.status, tc.status, tc.status)
		c, rec := newTestClient(srv.URL)
		_, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != tc.status {
			t.Fatalf("%d: err = %v", tc.status, err)
		}
		if calls.Load() != 1 || len(rec.waits) != 0 || apiErr.Retries != 0 {
			t.Errorf("%d %v: calls=%d waits=%v, want a single request", tc.status, tc.headers, calls.Load(), rec.waits)
		}
	}
}

func TestRetryAfterSecondsIsRespected(t *testing.T) {
	for _, status := range []int{503, 429, 403} {
		var calls atomic.Int32
		srv := flaky(t, &calls, http.Header{"Retry-After": {"7"}}, status)
		c, rec := newTestClient(srv.URL)
		_, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		if err != nil {
			t.Fatalf("%d: %v", status, err)
		}
		if fmt.Sprint(rec.waits) != "[7s]" {
			t.Errorf("%d: waits = %v, want [7s]", status, rec.waits)
		}
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	var calls atomic.Int32
	now := time.Unix(1_800_000_000, 0).UTC()
	srv := flaky(t, &calls, http.Header{"Retry-After": {now.Add(12 * time.Second).Format(http.TimeFormat)}}, 503)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rec.waits) != "[12s]" {
		t.Errorf("waits = %v, want [12s]", rec.waits)
	}
}

func TestRateLimitResetIsRespected(t *testing.T) {
	reset := strconv.FormatInt(1_800_000_000+9, 10)
	cases := []struct {
		name    string
		status  int
		headers http.Header
	}{
		{"403 sin cuota", 403, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {reset}}},
		{"429", 429, http.Header{"X-Ratelimit-Reset": {reset}}},
	}
	for _, tc := range cases {
		var calls atomic.Int32
		srv := flaky(t, &calls, tc.headers, tc.status)
		c, rec := newTestClient(srv.URL)
		_, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if fmt.Sprint(rec.waits) != "[9s]" {
			t.Errorf("%s: waits = %v, want [9s]", tc.name, rec.waits)
		}
	}
}

func TestRateLimitResetInThePastDoesNotWait(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1700000000"}}, 403)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rec.waits) != "[0s]" {
		t.Errorf("waits = %v, want [0s]", rec.waits)
	}
}

func TestRateLimitWithoutHintsUsesBackoff(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, nil, 429, 429)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rec.waits) != "[1s 2s]" {
		t.Errorf("waits = %v, want [1s 2s]", rec.waits)
	}
}

func TestDoesNotWaitLongerThanTheCap(t *testing.T) {
	cases := map[string]http.Header{
		"Retry-After":       {"Retry-After": {"3600"}},
		"X-RateLimit-Reset": {"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {strconv.FormatInt(1_800_000_000+3600, 10)}},
	}
	for name, h := range cases {
		var calls atomic.Int32
		srv := flaky(t, &calls, h, 403, 403)
		c, rec := newTestClient(srv.URL)
		_, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 403 {
			t.Fatalf("%s: err = %v, want APIError 403", name, err)
		}
		if calls.Load() != 1 || len(rec.waits) != 0 {
			t.Errorf("%s: calls=%d waits=%v, want to give up at once", name, calls.Load(), rec.waits)
		}
		if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
			t.Errorf("%s: error %q should still explain the rate limit", name, err)
		}
	}
}

func TestBackoffIsCappedByMaxWait(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, nil, 500, 500, 500)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	c.BaseDelay = 10 * time.Second
	c.MaxWait = 15 * time.Second
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rec.waits) != "[10s 15s 15s]" {
		t.Errorf("waits = %v, want [10s 15s 15s]", rec.waits)
	}
}

func TestRetryOnlyRepeatsTheFailedPage(t *testing.T) {
	var page1, page2 atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			page1.Add(1)
			fmt.Fprint(w, eventsJSON(100, 0))
			return
		}
		if page2.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, eventsJSON(5, 100))
	}))
	defer srv.Close()
	c, _ := newTestClient(srv.URL)
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 105 || page1.Load() != 1 || page2.Load() != 2 {
		t.Fatalf("events=%d page1=%d page2=%d", len(events), page1.Load(), page2.Load())
	}
}

func TestRetryStopsWhenContextEnds(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, nil, 500, 500, 500, 500)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	c, _ := newTestClient(srv.URL)
	c.Sleep = func(ctx context.Context, d time.Duration) error {
		cancel()
		return ctx.Err()
	}
	_, err := c.FetchEvents(ctx, "octoexample")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 500 {
		t.Fatalf("err = %v, want the last APIError", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
	if !strings.Contains(err.Error(), "interrumpido") {
		t.Errorf("error %q should say the retry was interrupted", err)
	}
}

func TestRealSleepHonoursContext(t *testing.T) {
	c := NewClient("")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := c.sleep(ctx, time.Minute); err == nil {
		t.Fatal("sleep should return the context error")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("sleep ignored the context")
	}
	if err := c.sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestRetryWithRealTimerAndTinyDelay(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, nil, 500, 500)
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL
	c.BaseDelay = time.Millisecond
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestAPIErrorMessageMentionsRetries(t *testing.T) {
	one := (&APIError{Status: 500, Message: "x", Retries: 1}).Error()
	if !strings.Contains(one, "tras 1 reintento") || strings.Contains(one, "reintentos") {
		t.Errorf("singular message = %q", one)
	}
	if strings.Contains((&APIError{Status: 404}).Error(), "reintento") {
		t.Error("a 404 has no retries to mention")
	}
}
