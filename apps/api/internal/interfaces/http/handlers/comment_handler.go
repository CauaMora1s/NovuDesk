package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/audit"
	"github.com/novudesk/novudesk/internal/domain/comment"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
	"github.com/novudesk/novudesk/pkg/validator"
)

type CommentHandler struct {
	comments comment.Repository
	audits   audit.Repository
}

func NewCommentHandler(comments comment.Repository, audits audit.Repository) *CommentHandler {
	return &CommentHandler{comments: comments, audits: audits}
}

// TimelineItemType distinguishes comments from audit events in the unified timeline.
type TimelineItemType string

const (
	TimelineComment  TimelineItemType = "comment"
	TimelineActivity TimelineItemType = "activity"
)

// TimelineItem is the unified view returned by the timeline endpoint.
type TimelineItem struct {
	Type      TimelineItemType `json:"type"`
	ID        string           `json:"id"`
	CreatedAt time.Time        `json:"created_at"`

	// Comment fields (populated when Type == "comment")
	Body         *string `json:"body,omitempty"`
	IsInternal   *bool   `json:"is_internal,omitempty"`
	AuthorID     *string `json:"author_id,omitempty"`
	AuthorName   *string `json:"author_name,omitempty"`
	AuthorAvatar *string `json:"author_avatar,omitempty"`

	// Activity fields (populated when Type == "activity")
	Action    *string          `json:"action,omitempty"`
	ActorID   *string          `json:"actor_id,omitempty"`
	ActorType *audit.ActorType `json:"actor_type,omitempty"`
	ActorName *string          `json:"actor_name,omitempty"`
	Before    *json.RawMessage `json:"before,omitempty"`
	After     *json.RawMessage `json:"after,omitempty"`
}

// List returns the unified timeline (comments + audit events) for a ticket.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	ticketID := chi.URLParam(r, "id")

	includeInternal := false
	for _, p := range []string{"comments:create_internal", "comments:edit_own"} {
		for _, perm := range claims.Permissions {
			if perm == p {
				includeInternal = true
				break
			}
		}
	}

	comments, err := h.comments.ListByTicket(r.Context(), ticketID, claims.OrgID, includeInternal)
	if err != nil {
		respond.Error(w, apperrors.Internal(err))
		return
	}

	logs, _, err := h.audits.ListByResource(r.Context(), claims.OrgID, "ticket", ticketID, 200, 0)
	if err != nil {
		respond.Error(w, apperrors.Internal(err))
		return
	}

	items := make([]TimelineItem, 0, len(comments)+len(logs))

	for _, c := range comments {
		isInternal := c.IsInternal
		item := TimelineItem{
			Type:         TimelineComment,
			ID:           c.ID,
			CreatedAt:    c.CreatedAt,
			Body:         &c.Body,
			IsInternal:   &isInternal,
			AuthorID:     c.AuthorID,
			AuthorName:   c.AuthorName,
			AuthorAvatar: c.AuthorAvatar,
		}
		items = append(items, item)
	}

	for _, l := range logs {
		action := l.Action
		actorType := l.ActorType
		var before, after *json.RawMessage
		if l.Before != nil {
			b := json.RawMessage(l.Before)
			before = &b
		}
		if l.After != nil {
			a := json.RawMessage(l.After)
			after = &a
		}
		item := TimelineItem{
			Type:      TimelineActivity,
			ID:        l.ID,
			CreatedAt: l.CreatedAt,
			Action:    &action,
			ActorID:   l.ActorID,
			ActorType: &actorType,
			ActorName: l.ActorName,
			Before:    before,
			After:     after,
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	respond.Ok(w, items)
}

type createCommentRequest struct {
	Body       string `json:"body"        validate:"required,min=1"`
	IsInternal bool   `json:"is_internal"`
}

// Create adds a new comment to a ticket.
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	ticketID := chi.URLParam(r, "id")

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	// Internal comments require the specific permission.
	if req.IsInternal {
		hasInternal := false
		for _, p := range claims.Permissions {
			if p == "comments:create_internal" {
				hasInternal = true
				break
			}
		}
		if !hasInternal {
			respond.Forbidden(w, "you do not have permission to create internal notes")
			return
		}
	}

	c := &comment.Comment{
		ID:         uuid.NewString(),
		TicketID:   ticketID,
		OrgID:      claims.OrgID,
		AuthorID:   &claims.UserID,
		Body:       req.Body,
		IsInternal: req.IsInternal,
	}

	created, err := h.comments.CreateWithAuthor(r.Context(), c)
	if err != nil {
		respond.Error(w, apperrors.Internal(err))
		return
	}

	isInternal := created.IsInternal
	item := TimelineItem{
		Type:         TimelineComment,
		ID:           created.ID,
		CreatedAt:    created.CreatedAt,
		Body:         &created.Body,
		IsInternal:   &isInternal,
		AuthorID:     created.AuthorID,
		AuthorName:   created.AuthorName,
		AuthorAvatar: created.AuthorAvatar,
	}

	respond.Created(w, item)
}
