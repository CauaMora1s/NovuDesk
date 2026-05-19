package email

import "context"

// Sender abstracts email delivery. SMTP is the default implementation.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

// Templates for each notification type.
const (
	TplTicketCreated   = "ticket_created"
	TplTicketAssigned  = "ticket_assigned"
	TplTicketUpdated   = "ticket_updated"
	TplCommentAdded    = "comment_added"
	TplInvitation      = "invitation"
	TplSLABreach       = "sla_breach"
	TplPasswordReset   = "password_reset"
)

// TemplateData holds variables injected into email templates.
type TemplateData struct {
	OrgName      string
	RecipientName string
	TicketNumber int64
	TicketTitle  string
	TicketURL    string
	InviteURL    string
	ActorName    string
	ExtraData    map[string]any
}
