package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BertMarti/commitling/internal/stats"
)

func loadFixture(t *testing.T) []Event {
	t.Helper()
	f, err := os.Open("../../testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	events, err := ParseEvents(f)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestParseFixture(t *testing.T) {
	events := loadFixture(t)
	if len(events) == 0 {
		t.Fatal("fixture has no events")
	}
	if got := Login(events); got != "octoexample" {
		t.Fatalf("Login = %q, want octoexample", got)
	}
	want := time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC)
	if got := Latest(events); got.Before(want) || got.After(want.Add(time.Hour)) {
		t.Fatalf("Latest = %s, want 2026-09-28 18:xx UTC", got)
	}

	acts := Activities(events)
	if len(acts) != len(events) {
		t.Fatalf("got %d activities for %d events", len(acts), len(events))
	}
	kinds := map[stats.Kind]int{}
	for _, a := range acts {
		kinds[a.Kind]++
	}
	// 2 PRs opened (one closed PR counts as other), 2 issues opened.
	if kinds[stats.KindPullRequest] != 2 || kinds[stats.KindIssue] != 2 {
		t.Fatalf("kinds = %v", kinds)
	}
	if kinds[stats.KindCommit] == 0 || kinds[stats.KindOther] == 0 {
		t.Fatalf("kinds = %v", kinds)
	}
}

func TestCommitCount(t *testing.T) {
	cases := []struct {
		payload string
		want    int
	}{
		{`{"size": 4, "commits": [{}, {}]}`, 4},
		{`{"commits": [{}, {}, {}]}`, 3},
		{`{"size": 0, "commits": [{}]}`, 1},
		{`{"push_id": 1, "ref": "refs/heads/main"}`, 1},
		{``, 1},
		{`"not an object"`, 1},
	}
	for _, c := range cases {
		e := Event{Type: "PushEvent", Payload: json.RawMessage(c.payload)}
		if got := CommitCount(e); got != c.want {
			t.Errorf("CommitCount(%s) = %d, want %d", c.payload, got, c.want)
		}
	}
}

func TestValidLogin(t *testing.T) {
	for _, ok := range []string{"octoexample", "BertMarti", "a", "a-b-c", strings.Repeat("x", 39)} {
		if !ValidLogin(ok) {
			t.Errorf("ValidLogin(%q) = false", ok)
		}
	}
	for _, bad := range []string{"", "-a", "a/b", "../x", "a b", strings.Repeat("x", 40), "a?b"} {
		if ValidLogin(bad) {
			t.Errorf("ValidLogin(%q) = true", bad)
		}
	}
}

func eventsJSON(n, offset int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf(`{"id":"%d","type":"WatchEvent","actor":{"login":"octoexample"},"repo":{"name":"x/y"},"payload":{},"created_at":"2026-09-01T10:00:00Z"}`, offset+i)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func TestFetchEventsPaginatesAndSendsToken(t *testing.T) {
	var pages []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/octoexample/events/public" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secreto" {
			t.Errorf("Authorization = %q", got)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page = %q", r.URL.Query().Get("per_page"))
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pages = append(pages, page)
		switch page {
		case 1, 2:
			fmt.Fprint(w, eventsJSON(100, page*1000))
		default:
			fmt.Fprint(w, eventsJSON(42, page*1000))
		}
	}))
	defer srv.Close()

	c := NewClient("secreto")
	c.BaseURL = srv.URL
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 242 {
		t.Fatalf("got %d events, want 242", len(events))
	}
	if fmt.Sprint(pages) != "[1 2 3]" {
		t.Fatalf("pages requested = %v", pages)
	}
}

func TestFetchEventsStopsOnShortPage(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected Authorization header without token")
		}
		fmt.Fprint(w, eventsJSON(3, 0))
	}))
	defer srv.Close()

	c := NewClient("")
	c.BaseURL = srv.URL
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || calls != 1 {
		t.Fatalf("events=%d calls=%d, want 3 and 1", len(events), calls)
	}
}

func TestFetchEventsPaginationLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, eventsJSON(100, 0))
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message":"In order to keep the API fast for everyone, pagination is limited for this resource."}`)
	}))
	defer srv.Close()

	c := NewClient("")
	c.BaseURL = srv.URL
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 100 {
		t.Fatalf("got %d events, want 100", len(events))
	}
}

func TestFetchEventsErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusNotFound, `{"message":"Not Found"}`, "no encontrado"},
		{http.StatusForbidden, `{"message":"API rate limit exceeded"}`, "GITHUB_TOKEN"},
		{http.StatusInternalServerError, `oops`, "500"},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.status)
			fmt.Fprint(w, c.body)
		}))
		cl := NewClient("")
		cl.BaseURL = srv.URL
		_, err := cl.FetchEvents(context.Background(), "octoexample")
		srv.Close()
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != c.status {
			t.Fatalf("status %d: err = %v", c.status, err)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("status %d: error %q does not mention %q", c.status, err, c.want)
		}
	}
}

func TestFetchEventsRejectsBadLogin(t *testing.T) {
	c := NewClient("")
	c.BaseURL = "http://127.0.0.1:1" // never contacted
	if _, err := c.FetchEvents(context.Background(), "../etc"); err == nil {
		t.Fatal("expected error for invalid login")
	}
}

func TestFetchEventsBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"not":"an array"}`)
	}))
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestFetchEventsSendsRequiredHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for header, want := range map[string]string{
			"User-Agent":           "commitling",
			"X-GitHub-Api-Version": "2022-11-28",
			"Accept":               "application/vnd.github+json",
		} {
			if got := r.Header.Get(header); got != want {
				t.Errorf("%s = %q, want %q", header, got, want)
			}
		}
		fmt.Fprint(w, eventsJSON(1, 0))
	}))
	defer srv.Close()

	c := NewClient("")
	c.BaseURL = srv.URL
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
	// A hand-built client without user agent or HTTP client still works.
	bare := &Client{BaseURL: srv.URL}
	if _, err := bare.FetchEvents(context.Background(), "octoexample"); err != nil {
		t.Fatal(err)
	}
}

func TestNewClientHasTimeout(t *testing.T) {
	c := NewClient("")
	if c.HTTPClient == nil || c.HTTPClient.Timeout != 15*time.Second {
		t.Fatalf("HTTPClient timeout = %v, want 15s", c.HTTPClient)
	}
	if c.UserAgent != "commitling" {
		t.Errorf("UserAgent = %q", c.UserAgent)
	}
}

func TestFetchEventsTimesOut(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	c := NewClient("")
	c.BaseURL = srv.URL
	c.HTTPClient.Timeout = 50 * time.Millisecond
	start := time.Now()
	_, err := c.FetchEvents(context.Background(), "octoexample")
	if err == nil || !strings.Contains(err.Error(), "no se pudo contactar") {
		t.Fatalf("err = %v, want a connection error", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("request did not time out")
	}
}

func TestFetchEventsPaginationLimitAfterPageTwo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "1":
			fmt.Fprint(w, eventsJSON(100, 0))
		case "2":
			fmt.Fprint(w, eventsJSON(100, 100))
		default:
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, `{"message":"pagination is limited for this resource"}`)
		}
	}))
	defer srv.Close()

	c := NewClient("")
	c.BaseURL = srv.URL
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatalf("a 422 past the last page is the end of the data, got %v", err)
	}
	if len(events) != 200 {
		t.Fatalf("got %d events, want 200", len(events))
	}
}

func TestFetchEvents422OnFirstPageIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message":"Validation Failed"}`)
	}))
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL
	if _, err := c.FetchEvents(context.Background(), "octoexample"); err == nil {
		t.Fatal("422 on the first page must not be swallowed")
	}
}

func TestDedupeAndActivitiesCountOnce(t *testing.T) {
	push := json.RawMessage(`{"size": 2}`)
	at := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "1", Type: "PushEvent", CreatedAt: at, Payload: push},
		{ID: "1", Type: "PushEvent", CreatedAt: at, Payload: push},
		{ID: "2", Type: "WatchEvent", CreatedAt: at},
		{ID: "", Type: "WatchEvent", CreatedAt: at}, // no id: never merged
		{ID: "", Type: "WatchEvent", CreatedAt: at},
	}
	if got := Dedupe(events); len(got) != 4 {
		t.Fatalf("Dedupe left %d events, want 4", len(got))
	}
	acts := Activities(events)
	if len(acts) != 4 {
		t.Fatalf("got %d activities, want 4", len(acts))
	}
	s := stats.Compute(acts, at.Add(time.Hour))
	if s.Commits != 2 {
		t.Fatalf("Commits = %d, want 2 (duplicate push counted twice)", s.Commits)
	}
}

func TestFetchEventsDropsOverlapBetweenPages(t *testing.T) {
	// A new event arrives between requests, so page 2 repeats the last
	// event of page 1.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, eventsJSON(100, 0)) // ids 0..99
			return
		}
		fmt.Fprint(w, eventsJSON(5, 99)) // ids 99..103
	}))
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL
	events, err := c.FetchEvents(context.Background(), "octoexample")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 104 {
		t.Fatalf("got %d events, want 104", len(events))
	}
}
