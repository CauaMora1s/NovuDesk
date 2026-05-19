package events

import (
	"context"
	"encoding/json"
	"time"
)

// Type identifies the kind of domain event.
type Type string

const (
	// Ticket events
	TicketCreated      Type = "ticket.created"
	TicketUpdated      Type = "ticket.updated"
	TicketAssigned     Type = "ticket.assigned"
	TicketStatusChanged Type = "ticket.status_changed"
	TicketClosed       Type = "ticket.closed"
	TicketReopened     Type = "ticket.reopened"

	// Comment events
	CommentCreated Type = "comment.created"

	// User events
	UserInvited   Type = "user.invited"
	UserJoined    Type = "user.joined"
	UserDeactivated Type = "user.deactivated"

	// Organization events
	OrgCreated Type = "organization.created"
	OrgUpdated Type = "organization.updated"

	// SLA events
	SLABreachWarning Type = "sla.breach_warning"
	SLABreached      Type = "sla.breached"
)

// Event is a domain event that flows through the EventBus.
type Event struct {
	ID        string          `json:"id"`
	Type      Type            `json:"type"`
	OrgID     string          `json:"org_id"`
	ActorID   string          `json:"actor_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// Publisher publishes domain events.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Subscriber receives events of specific types.
type Subscriber interface {
	Subscribe(ctx context.Context, types []Type, handler func(Event) error) error
}

// Bus combines publishing and subscribing.
type Bus interface {
	Publisher
	Subscriber
}

// NoopBus discards all events — useful in tests.
type NoopBus struct{}

func (NoopBus) Publish(_ context.Context, _ Event) error { return nil }
func (NoopBus) Subscribe(_ context.Context, _ []Type, _ func(Event) error) error {
	return nil
}

// MarshalPayload serializes an arbitrary value into the event payload.
func MarshalPayload(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}
