package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/novudesk/novudesk/internal/domain/ticket"
)

// ticketRow is a scan target that handles PostgreSQL array types.
type ticketRow struct {
	ID                 string          `db:"id"`
	OrgID              string          `db:"org_id"`
	Number             int64           `db:"number"`
	Title              string          `db:"title"`
	Description        *string         `db:"description"`
	Status             ticket.Status   `db:"status"`
	Priority           ticket.Priority `db:"priority"`
	AssigneeID         *string         `db:"assignee_id"`
	AssigneeName       *string         `db:"assignee_name"`
	TeamID             *string         `db:"team_id"`
	TeamName           *string         `db:"team_name"`
	RequesterID        *string         `db:"requester_id"`
	RequesterName      *string         `db:"requester_name"`
	CategoryID         *string         `db:"category_id"`
	CategoryName       *string         `db:"category_name"`
	SLAPolicyID        *string         `db:"sla_policy_id"`
	SLAResponseDueAt   *time.Time      `db:"sla_response_due_at"`
	SLAResolutionDueAt *time.Time      `db:"sla_resolution_due_at"`
	SLABreached        bool            `db:"sla_breached"`
	Tags               pq.StringArray  `db:"tags"`
	CustomFields       []byte          `db:"custom_fields"`
	CreatedAt          time.Time       `db:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at"`
	ResolvedAt         *time.Time      `db:"resolved_at"`
	ClosedAt           *time.Time      `db:"closed_at"`
}

func (r *ticketRow) toTicket() *ticket.Ticket {
	t := &ticket.Ticket{
		ID:                 r.ID,
		OrgID:              r.OrgID,
		Number:             r.Number,
		Title:              r.Title,
		Description:        r.Description,
		Status:             r.Status,
		Priority:           r.Priority,
		AssigneeID:         r.AssigneeID,
		AssigneeName:       r.AssigneeName,
		TeamID:             r.TeamID,
		TeamName:           r.TeamName,
		RequesterID:        r.RequesterID,
		RequesterName:      r.RequesterName,
		CategoryID:         r.CategoryID,
		CategoryName:       r.CategoryName,
		SLAPolicyID:        r.SLAPolicyID,
		SLAResponseDueAt:   r.SLAResponseDueAt,
		SLAResolutionDueAt: r.SLAResolutionDueAt,
		SLABreached:        r.SLABreached,
		Tags:               []string(r.Tags),
		CustomFields:       json.RawMessage(r.CustomFields),
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
		ResolvedAt:         r.ResolvedAt,
		ClosedAt:           r.ClosedAt,
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	return t
}

type ticketRepo struct {
	db *sqlx.DB
}

func NewTicketRepo(db *sqlx.DB) ticket.Repository {
	return &ticketRepo{db: db}
}

func (r *ticketRepo) Create(ctx context.Context, t *ticket.Ticket) error {
	q := `INSERT INTO tickets
	        (id, org_id, number, title, description, status, priority,
	         assignee_id, team_id, category_id, requester_id, sla_policy_id,
	         sla_response_due_at, sla_resolution_due_at, tags, custom_fields)
	      VALUES
	        ($1, $2, $3, $4, $5, $6::ticket_status, $7::ticket_priority,
	         $8, $9, $10, $11, $12, $13, $14, $15, $16::jsonb)`
	_, err := r.db.ExecContext(ctx, q,
		t.ID, t.OrgID, t.Number, t.Title, t.Description, string(t.Status), string(t.Priority),
		t.AssigneeID, t.TeamID, t.CategoryID, t.RequesterID, t.SLAPolicyID,
		t.SLAResponseDueAt, t.SLAResolutionDueAt, pq.Array(t.Tags), string(t.CustomFields))
	return err
}

const ticketSelectJoins = `
	SELECT t.*,
	       ua.full_name  AS assignee_name,
	       tm.name       AS team_name,
	       ur.full_name  AS requester_name,
	       cat.name      AS category_name
	FROM tickets t
	LEFT JOIN users  ua  ON ua.id  = t.assignee_id
	LEFT JOIN teams  tm  ON tm.id  = t.team_id
	LEFT JOIN users  ur  ON ur.id  = t.requester_id
	LEFT JOIN categories cat ON cat.id = t.category_id`

func (r *ticketRepo) FindByID(ctx context.Context, id, orgID string) (*ticket.Ticket, error) {
	var row ticketRow
	err := r.db.GetContext(ctx, &row,
		ticketSelectJoins+` WHERE t.id = $1 AND t.org_id = $2`, id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toTicket(), nil
}

func (r *ticketRepo) FindByNumber(ctx context.Context, number int64, orgID string) (*ticket.Ticket, error) {
	var row ticketRow
	err := r.db.GetContext(ctx, &row,
		ticketSelectJoins+` WHERE t.number = $1 AND t.org_id = $2`, number, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toTicket(), nil
}

func (r *ticketRepo) List(ctx context.Context, orgID string, f ticket.Filter, limit, offset int) ([]*ticket.Ticket, int64, error) {
	conds := []string{"t.org_id = $1"}
	args := []any{orgID}
	idx := 2

	if len(f.Status) > 0 {
		conds = append(conds, fmt.Sprintf("t.status = ANY($%d)", idx))
		args = append(args, pq.Array(f.Status))
		idx++
	}
	if len(f.Priority) > 0 {
		conds = append(conds, fmt.Sprintf("t.priority = ANY($%d)", idx))
		args = append(args, pq.Array(f.Priority))
		idx++
	}
	if f.AssigneeID != nil {
		conds = append(conds, fmt.Sprintf("t.assignee_id = $%d", idx))
		args = append(args, *f.AssigneeID)
		idx++
	}
	if f.TeamID != nil {
		conds = append(conds, fmt.Sprintf("t.team_id = $%d", idx))
		args = append(args, *f.TeamID)
		idx++
	}
	if f.RequesterID != nil {
		conds = append(conds, fmt.Sprintf("t.requester_id = $%d", idx))
		args = append(args, *f.RequesterID)
		idx++
	}
	if f.CategoryID != nil {
		conds = append(conds, fmt.Sprintf("t.category_id = $%d", idx))
		args = append(args, *f.CategoryID)
		idx++
	}
	if len(f.Tags) > 0 {
		conds = append(conds, fmt.Sprintf("t.tags && $%d", idx))
		args = append(args, pq.Array(f.Tags))
		idx++
	}
	if f.Query != "" {
		conds = append(conds, fmt.Sprintf(
			"to_tsvector('portuguese', coalesce(t.title,'') || ' ' || coalesce(t.description,'')) @@ plainto_tsquery('portuguese', $%d)", idx))
		args = append(args, f.Query)
		idx++
	}
	if f.SLABreached != nil {
		conds = append(conds, fmt.Sprintf("t.sla_breached = $%d", idx))
		args = append(args, *f.SLABreached)
		idx++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int64
	r.db.GetContext(ctx, &total,
		fmt.Sprintf(`SELECT COUNT(1) FROM tickets t %s`, where), args...)

	q := fmt.Sprintf(`%s %s ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d`,
		ticketSelectJoins, where, idx, idx+1)
	args = append(args, limit, offset)

	var rows []ticketRow
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, 0, err
	}

	tickets := make([]*ticket.Ticket, len(rows))
	for i := range rows {
		tickets[i] = rows[i].toTicket()
	}
	return tickets, total, nil
}

func (r *ticketRepo) Update(ctx context.Context, id, orgID string, input ticket.UpdateInput) (*ticket.Ticket, error) {
	q := `UPDATE tickets SET
	          title        = COALESCE($3, title),
	          description  = COALESCE($4, description),
	          status       = COALESCE($5::text, status::text)::ticket_status,
	          priority     = COALESCE($6::text, priority::text)::ticket_priority,
	          assignee_id  = COALESCE($7, assignee_id),
	          team_id      = COALESCE($8, team_id),
	          category_id  = COALESCE($9, category_id),
	          custom_fields= COALESCE($10::jsonb, custom_fields),
	          tags         = COALESCE($11, tags),
	          updated_at   = NOW()
	      WHERE id = $1 AND org_id = $2
	      RETURNING id, org_id, number, title, description, status, priority,
	                assignee_id, team_id, category_id, requester_id, sla_policy_id,
	                sla_response_due_at, sla_resolution_due_at, sla_breached,
	                tags, custom_fields, created_at, updated_at, resolved_at, closed_at`

	var statusStr, priorityStr *string
	if input.Status != nil {
		s := string(*input.Status)
		statusStr = &s
	}
	if input.Priority != nil {
		p := string(*input.Priority)
		priorityStr = &p
	}

	var tagsArg any
	if input.Tags != nil {
		tagsArg = pq.Array(input.Tags)
	}

	var customFieldsStr *string
	if input.CustomFields != nil {
		s := string(input.CustomFields)
		customFieldsStr = &s
	}

	var row ticketRow
	err := r.db.GetContext(ctx, &row, q, id, orgID,
		input.Title, input.Description, statusStr, priorityStr,
		input.AssigneeID, input.TeamID, input.CategoryID, customFieldsStr, tagsArg)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toTicket(), nil
}

func (r *ticketRepo) Delete(ctx context.Context, id, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM tickets WHERE id = $1 AND org_id = $2`, id, orgID)
	return err
}

func (r *ticketRepo) NextNumber(ctx context.Context, orgID string) (int64, error) {
	var n int64
	err := r.db.GetContext(ctx, &n,
		`SELECT COALESCE(MAX(number), 0) + 1 FROM tickets WHERE org_id = $1`, orgID)
	return n, err
}

func (r *ticketRepo) UpdateSLABreach(ctx context.Context, orgID string) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tickets SET sla_breached = TRUE
		 WHERE org_id = $1
		   AND sla_breached = FALSE
		   AND sla_resolution_due_at < NOW()
		   AND status NOT IN ('resolved', 'closed')`,
		orgID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
