package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/attachment"
)

type attachmentRepo struct {
	db *sqlx.DB
}

func NewAttachmentRepo(db *sqlx.DB) attachment.Repository {
	return &attachmentRepo{db: db}
}

func (r *attachmentRepo) Create(ctx context.Context, a *attachment.Attachment) error {
	q := `INSERT INTO ticket_attachments
	        (id, ticket_id, org_id, comment_id, uploader_id, filename, mime_type, size_bytes, storage_key)
	      VALUES
	        (:id, :ticket_id, :org_id, :comment_id, :uploader_id, :filename, :mime_type, :size_bytes, :storage_key)`
	_, err := r.db.NamedExecContext(ctx, q, a)
	return err
}

func (r *attachmentRepo) ListByTicket(ctx context.Context, ticketID, orgID string) ([]*attachment.Attachment, error) {
	q := `SELECT * FROM ticket_attachments
	      WHERE ticket_id = $1 AND org_id = $2
	      ORDER BY created_at ASC`
	items := make([]*attachment.Attachment, 0)
	err := r.db.SelectContext(ctx, &items, q, ticketID, orgID)
	return items, err
}

func (r *attachmentRepo) Delete(ctx context.Context, id, orgID string) (*attachment.Attachment, error) {
	var a attachment.Attachment
	err := r.db.GetContext(ctx, &a,
		`DELETE FROM ticket_attachments WHERE id = $1 AND org_id = $2 RETURNING *`,
		id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &a, err
}
