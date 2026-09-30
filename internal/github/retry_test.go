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

// dropper closes the connection without answering the first n requests and
// then answers 200 with two events.
func dropper(calls *atomic.Int32, n int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if int(calls.Add(1)) <= n {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		fmt.Fprint(w, eventsJSON(2, 0))
	}))
}

func TestRetriesNetworkErrors(t *testing.T) {
	var calls atomic.Int32
	srv := dropper(&calls, 2)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || calls.Load() != 3 || fmt.Sprint(rec.waits) != "[1s 2s]" {
		t.Fatalf("events=%d calls=%d waits=%v, want 2, 3 and [1s 2s]", len(events), calls.Load(), rec.waits)
	}
}

func TestNetworkErrorsGiveUpWithTheSameLimit(t *testing.T) {
	var calls atomic.Int32
	srv := dropper(&calls, 100)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	_, err := c.FetchEvents(context.Background(), "octoexample")
	if err == nil {
		t.Fatal("want an error")
	}
	if calls.Load() != 4 || fmt.Sprint(rec.waits) != "[1s 2s 4s]" {
		t.Errorf("calls=%d waits=%v, want 4 requests and [1s 2s 4s]", calls.Load(), rec.waits)
	}
	for _, want := range []string{"no se pudo contactar", "tras 3 reintentos"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
	}
}

func TestConnectionRefusedIsRetried(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nobody listens any more
	c, rec := newTestClient(url)
	_, err := c.FetchEvents(context.Background(), "octoexample")
	if err == nil || !strings.Contains(err.Error(), "tras 3 reintentos") {
		t.Fatalf("err = %v, want a failure after 3 retries", err)
	}
	if fmt.Sprint(rec.waits) != "[1s 2s 4s]" {
		t.Errorf("waits = %v", rec.waits)
	}
}

func TestRequestTimeoutIsRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			select { // hang until the client gives up
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
			return
		}
		fmt.Fprint(w, eventsJSON(2, 0))
	}))
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	c.HTTPClient.Timeout = 50 * time.Millisecond
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil || len(events) != 2 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
	if fmt.Sprint(rec.waits) != "[1s]" {
		t.Errorf("waits = %v, want [1s]", rec.waits)
	}
}

func TestNetworkErrorIsNotRetriedOnceContextEnded(t *testing.T) {
	var calls atomic.Int32
	srv := dropper(&calls, 100)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, rec := newTestClient(srv.URL)
	_, err := c.FetchEvents(ctx, "octoexample")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(rec.waits) != 0 {
		t.Errorf("waits = %v, want none", rec.waits)
	}
}

func TestNetworkRetryStopsWhenSleepIsInterrupted(t *testing.T) {
	var calls atomic.Int32
	srv := dropper(&calls, 100)
	defer srv.Close()
	c, _ := newTestClient(srv.URL)
	c.Sleep = func(ctx context.Context, d time.Duration) error { return context.Canceled }
	_, err := c.FetchEvents(context.Background(), "octoexample")
	if err == nil || !strings.Contains(err.Error(), "interrumpido") || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d", err, calls.Load())
	}
}

// A request that cannot even be built is a bug, not a flaky network.
func TestBadBaseURLIsNotRetried(t *testing.T) {
	c, rec := newTestClient("http://[::1")
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err == nil {
		t.Fatal("want an error")
	}
	if len(rec.waits) != 0 {
		t.Errorf("waits = %v, want none", rec.waits)
	}
}

// Retry-After values so large that seconds*time.Second overflows must be
// treated as "too long", not as a negative (immediate) wait.
func TestHugeRetryAfterIsNotRetried(t *testing.T) {
	for _, v := range []string{"10000000000", "9223372036", "9223372036854775807"} {
		var calls atomic.Int32
		srv := flaky(t, &calls, http.Header{"Retry-After": {v}}, 503, 503)
		c, rec := newTestClient(srv.URL)
		_, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		var apiErr *APIError
		if !errors.As(err, &apiErr) || calls.Load() != 1 || len(rec.waits) != 0 {
			t.Errorf("Retry-After %s: err=%v calls=%d waits=%v, want to give up at once", v, err, calls.Load(), rec.waits)
		}
	}
}

// Every request of the client is a GET, so repeating one is always safe.
func TestOnlyGETRequests(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if len(methods) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, eventsJSON(1, 0))
	}))
	defer srv.Close()
	c, _ := newTestClient(srv.URL)
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	for _, m := range methods {
		if m != http.MethodGet {
			t.Errorf("method %s, want GET", m)
		}
	}
}

// A zero or negative Retry-After and a date in the past mean "now".
func TestRetryAfterZeroAndPastDate(t *testing.T) {
	past := time.Unix(1_800_000_000, 0).UTC().Add(-time.Hour).Format(http.TimeFormat)
	for _, v := range []string{"0", past} {
		var calls atomic.Int32
		srv := flaky(t, &calls, http.Header{"Retry-After": {v}}, 503)
		c, rec := newTestClient(srv.URL)
		_, err := c.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		if err != nil || fmt.Sprint(rec.waits) != "[0s]" {
			t.Errorf("Retry-After %q: err=%v waits=%v, want [0s]", v, err, rec.waits)
		}
	}
}

// An unreadable Retry-After falls back to the exponential backoff.
func TestGarbageRetryAfterUsesBackoff(t *testing.T) {
	var calls atomic.Int32
	srv := flaky(t, &calls, http.Header{"Retry-After": {"mañana"}}, 503)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rec.waits) != "[1s]" {
		t.Errorf("waits = %v, want [1s]", rec.waits)
	}
}

func TestHugeRateLimitResetIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	h := http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"9223372036854775807"}}
	srv := flaky(t, &calls, h, 403, 403)
	defer srv.Close()
	c, rec := newTestClient(srv.URL)
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err == nil || calls.Load() != 1 || len(rec.waits) != 0 {
		t.Fatalf("err=%v calls=%d waits=%v, want to give up at once", err, calls.Load(), rec.waits)
	}
}

// IsTransient tells a passing failure (worth keeping the last good result
// for) from one that will not fix itself.
func TestIsTransient(t *testing.T) {
	limit := http.Header{"X-RateLimit-Remaining": {"0"}}
	cases := []struct {
		name    string
		status  int
		headers http.Header
		want    bool
	}{
		{"500", 500, nil, true},
		{"502", 502, nil, true},
		{"503", 503, nil, true},
		{"504", 504, nil, true},
		{"599", 599, nil, true},
		{"429", 429, nil, true},
		{"403 rate limit remaining 0", 403, limit, true},
		{"403 Retry-After", 403, http.Header{"Retry-After": {"1"}}, true},
		{"403 without headers", 403, nil, false},
		{"404", 404, nil, false},
		{"401", 401, nil, false},
		{"422", 422, nil, false},
	}
	for _, c := range cases {
		var calls atomic.Int32
		srv := flaky(t, &calls, c.headers, c.status, c.status, c.status, c.status) // every retry fails too
		cl, _ := newTestClient(srv.URL)
		_, err := cl.FetchEvents(context.Background(), "octoexample")
		if err == nil {
			t.Fatalf("%s: expected an error", c.name)
		}
		if got := IsTransient(err); got != c.want {
			t.Errorf("%s: IsTransient = %v, want %v (%v)", c.name, got, c.want, err)
		}
	}

	// Network failures are transient, also once wrapped.
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	cl, _ := newTestClient(srv.URL)
	_, err := cl.FetchEvents(context.Background(), "octoexample")
	if !IsTransient(err) || !IsTransient(fmt.Errorf("envuelto: %w", err)) {
		t.Errorf("a network error must be transient: %v", err)
	}
	// Anything else (bad input, unparsable body) is not.
	if IsTransient(errors.New("otra cosa")) || IsTransient(nil) {
		t.Error("unknown errors and nil are not transient")
	}
}
