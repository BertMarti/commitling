package github

import "github.com/BertMarti/commitling/internal/events"

// The parsing of events lives in internal/events (no networking, so the
// WebAssembly build does not link net/http). These aliases keep the API of
// this package as it was.

// Event is a public GitHub event; see events.Event.
type Event = events.Event

// Functions that turn events into activities; see package events.
var (
	ParseEvents = events.ParseEvents
	CommitCount = events.CommitCount
	Dedupe      = events.Dedupe
	Activities  = events.Activities
	Latest      = events.Latest
	Login       = events.Login
)
