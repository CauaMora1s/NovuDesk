package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/comment"
)

type commentRepo struct {
	db *sqlx.DB
}

func NewCommentRepo(db *sqlx.DB) comment.Repository {
	return &commentRepo{db: db}
}

func (r *commentRepo) Create(ctx context.Context, c *comment.Comment) error {
	q := `INSERT INTO ticket_comments (id, ticket_id, org_id, author_id, body, is_internal)
	      VALUES (:id, :ticket_id, :org_id, :author_id, :body, :is_internal)`
	_, err := r.db.NamedExecContext(ctx, q, c)
	return err
}

func (r *commentRepo) CreateWithAuthor(ctx context.Context, c *comment.Comment) (*comment.Comment, error) {
	q := `WITH ins AS (
	        INSERT INTO ticket_comments (id, ticket_id, org_id, author_id, body, is_internal)
	        VALUES ($1, $2, $3, $4, $5, $6)
	        RETURNING *
	      )
	      SELECT ins.*, u.full_name AS author_name, u.avatar_url AS author_avatar
	      FROM ins LEFT JOIN users u ON u.id = ins.author_id`
	var result comment.Comment
	err := r.db.GetContext(ctx, &result, q,
		c.ID, c.TicketID, c.OrgID, c.AuthorID, c.Body, c.IsInternal)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *commentRepo) FindByID(ctx context.Context, id, orgID string) (*comment.Comment, error) {
	var c comment.Comment
	err := r.db.GetContext(ctx, &c,
		`SELECT * FROM ticket_comments WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`,
		id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *commentRepo) ListByTicket(ctx context.Context, ticketID, orgID string, includeInternal bool) ([]*comment.Comment, error) {
	q := `SELECT c.*, u.full_name AS author_name, u.avatar_url AS author_avatar
	      FROM ticket_comments c
	      LEFT JOIN users u ON u.id = c.author_id
	      WHERE c.ticket_id = $1 AND c.org_id = $2 AND c.deleted_at IS NULL`
	if !includeInternal {
		q += ` AND c.is_internal = FALSE`
	}
	q += ` ORDER BY c.created_at ASC`

	comments := make([]*comment.Comment, 0)
	err := r.db.SelectContext(ctx, &comments, q, ticketID, orgID)
	return comments, err
}

func (r *commentRepo) Update(ctx context.Context, id, orgID string, input comment.UpdateInput) (*comment.Comment, error) {
	q := `UPDATE ticket_comments SET body = $3, updated_at = NOW()
	      WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
	      RETURNING *`
	var c comment.Comment
	err := r.db.GetContext(ctx, &c, q, id, orgID, input.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *commentRepo) SoftDelete(ctx context.Context, id, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ticket_comments SET deleted_at = NOW() WHERE id = $1 AND org_id = $2`,
		id, orgID)
	return err
}
