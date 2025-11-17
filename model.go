package outbox

import (
	"encoding/json"
	"time"
)

type FindEventsFilters struct{}

type CreateEvent struct {
	AggregateID string
	Topic       string
	Type        string
	Payload     json.RawMessage
}

type SuccessEvent struct {
	ID   string
	Type string
}

type FailedEvent struct {
	ID    string
	Type  string
	Error error
}

type messageMetadata struct {
	ID   string
	Type string
}

type EventStatus string

func (es EventStatus) String() string {
	return string(es)
}

const EventStatusNew EventStatus = "new"
const EventStatusDone EventStatus = "done"

type Event struct {
	ID          string
	AggregateID string
	Topic       string
	Type        string
	Payload     json.RawMessage
	Status      EventStatus
	CreatedAt   time.Time
	ReservedTo  *time.Time
}
