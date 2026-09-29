// Package github reads the public events of a GitHub user and turns them
// into activities for the stats package.
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/BertMarti/commitling/internal/stats"
)

// Event is the subset of a public GitHub event that commitling uses.
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Actor     struct {
		Login string `json:"login"`
	} `json:"actor"`
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
	Payload json.RawMessage `json:"payload"`
}

type pushPayload struct {
	Size         *int              `json:"size"`
	DistinctSize *int              `json:"distinct_size"`
	Commits      []json.RawMessage `json:"commits"`
}

type actionPayload struct {
	Action string `json:"action"`
}

// ParseEvents decodes a JSON array of events, like the one returned by
// GET /users/{user}/events/public or stored in testdata/events.json.
func ParseEvents(r io.Reader) ([]Event, error) {
	var events []Event
	if err := json.NewDecoder(r).Decode(&events); err != nil {
		return nil, fmt.Errorf("no se pudieron leer los eventos: %w", err)
	}
	return events, nil
}

// CommitCount returns how many commits a PushEvent carries. It prefers the
// "size" field, then the length of "commits". GitHub has been trimming push
// payloads, so a push without either still counts as one commit.
func CommitCount(e Event) int {
	var p pushPayload
	if len(e.Payload) > 0 {
		_ = json.Unmarshal(e.Payload, &p)
	}
	switch {
	case p.Size != nil && *p.Size > 0:
		return *p.Size
	case len(p.Commits) > 0:
		return len(p.Commits)
	default:
		return 1
	}
}

func action(e Event) string {
	var p actionPayload
	if len(e.Payload) > 0 {
		_ = json.Unmarshal(e.Payload, &p)
	}
	return p.Action
}

// Dedupe drops repeated events (same non-empty id), keeping the first one.
// Pages of the events API can overlap when new events arrive while paging.
func Dedupe(events []Event) []Event {
	seen := make(map[string]bool, len(events))
	out := make([]Event, 0, len(events))
	for _, e := range events {
		if e.ID != "" {
			if seen[e.ID] {
				continue
			}
			seen[e.ID] = true
		}
		out = append(out, e)
	}
	return out
}

// Activities converts events into stats activities. Repeated events (same
// id) are counted once.
//
//   - PushEvent: one commit activity with Count = number of commits.
//   - PullRequestEvent with action "opened" (or no action): a pull request.
//   - IssuesEvent with action "opened" (or no action): an issue.
//   - Anything else (stars, forks, comments, reviews, other PR/issue
//     actions...): "other" activity.
func Activities(events []Event) []stats.Activity {
	events = Dedupe(events)
	acts := make([]stats.Activity, 0, len(events))
	for _, e := range events {
		a := stats.Activity{At: e.CreatedAt, Repo: e.Repo.Name, Kind: stats.KindOther, Count: 1}
		switch e.Type {
		case "PushEvent":
			a.Kind = stats.KindCommit
			a.Count = CommitCount(e)
		case "PullRequestEvent":
			if act := action(e); act == "" || act == "opened" {
				a.Kind = stats.KindPullRequest
			}
		case "IssuesEvent":
			if act := action(e); act == "" || act == "opened" {
				a.Kind = stats.KindIssue
			}
		}
		acts = append(acts, a)
	}
	return acts
}

// Latest returns the time of the most recent event, or the zero time.
func Latest(events []Event) time.Time {
	var t time.Time
	for _, e := range events {
		if e.CreatedAt.After(t) {
			t = e.CreatedAt
		}
	}
	return t
}

// Login returns the most common actor login in the events, or "".
func Login(events []Event) string {
	counts := map[string]int{}
	for _, e := range events {
		if e.Actor.Login != "" {
			counts[e.Actor.Login]++
		}
	}
	logins := make([]string, 0, len(counts))
	for l := range counts {
		logins = append(logins, l)
	}
	sort.Slice(logins, func(i, j int) bool {
		if counts[logins[i]] != counts[logins[j]] {
			return counts[logins[i]] > counts[logins[j]]
		}
		return logins[i] < logins[j]
	})
	if len(logins) == 0 {
		return ""
	}
	return logins[0]
}
