package attachment

import (
	"context"
	"time"
)

type Attachment struct {
	ID         string    `db:"id"          json:"id"`
	TicketID   string    `db:"ticket_id"   json:"ticket_id"`
	OrgID      string    `db:"org_id"      json:"org_id"`
	CommentID  *string   `db:"comment_id"  json:"comment_id,omitempty"`
	UploaderID *string   `db:"uploader_id" json:"uploader_id,omitempty"`
	Filename   string    `db:"filename"    json:"filename"`
	MimeType   string    `db:"mime_type"   json:"mime_type"`
	SizeBytes  int64     `db:"size_bytes"  json:"size_bytes"`
	StorageKey string    `db:"storage_key" json:"-"`
	URL        string    `db:"-"           json:"url"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, a *Attachment) error
	ListByTicket(ctx context.Context, ticketID, orgID string) ([]*Attachment, error)
	Delete(ctx context.Context, id, orgID string) (*Attachment, error)
}
