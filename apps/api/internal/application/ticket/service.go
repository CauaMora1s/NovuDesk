package ticket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	auditapp "github.com/novudesk/novudesk/internal/application/audit"
	"github.com/novudesk/novudesk/internal/domain/audit"
	"github.com/novudesk/novudesk/internal/domain/sla"
	"github.com/novudesk/novudesk/internal/domain/ticket"
	"github.com/novudesk/novudesk/pkg/events"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type Service struct {
	tickets  ticket.Repository
	slas     sla.Repository
	audits   audit.Repository
	eventBus events.Publisher
}

func NewService(
	tickets ticket.Repository,
	slas sla.Repository,
	audits audit.Repository,
	bus events.Publisher,
) *Service {
	return &Service{tickets: tickets, slas: slas, audits: audits, eventBus: bus}
}

func (s *Service) Create(ctx context.Context, input ticket.CreateInput) (*ticket.Ticket, error) {
	number, err := s.tickets.NextNumber(ctx, input.OrgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	requesterID := input.RequesterID
	if requesterID == nil && input.CreatedBy != "" {
		requesterID = &input.CreatedBy
	}

	t := &ticket.Ticket{
		ID:           uuid.NewString(),
		OrgID:        input.OrgID,
		Number:       number,
		Title:        input.Title,
		Description:  input.Description,
		Status:       ticket.StatusOpen,
		Priority:     input.Priority,
		AssigneeID:   input.AssigneeID,
		TeamID:       input.TeamID,
		CategoryID:   input.CategoryID,
		RequesterID:  requesterID,
		SLAPolicyID:  input.SLAPolicyID,
		Tags:         input.Tags,
		CustomFields: input.CustomFields,
	}

	if t.CustomFields == nil {
		t.CustomFields = json.RawMessage(`{}`)
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}

	// Apply SLA due dates if a policy is assigned.
	if input.SLAPolicyID != nil && s.slas != nil {
		policy, err := s.slas.FindByID(ctx, *input.SLAPolicyID, input.OrgID)
		if err == nil && policy != nil {
			resp, resol := policy.CalculateDueDates(time.Now())
			t.SLAResponseDueAt = &resp
			t.SLAResolutionDueAt = &resol
		}
	}

	if err := s.tickets.Create(ctx, t); err != nil {
		return nil, apperrors.Internal(err)
	}

	// Write audit log.
	auditapp.WriteEntry(ctx, s.audits, audit.CreateInput{
		OrgID:        input.OrgID,
		ActorID:      &input.CreatedBy,
		ActorType:    audit.ActorUser,
		ResourceType: "ticket",
		ResourceID:   t.ID,
		Action:       "ticket.created",
		After:        t,
	})

	// Publish domain event for SSE + async workers.
	payload, _ := events.MarshalPayload(t)
	s.eventBus.Publish(ctx, events.Event{
		ID:        uuid.NewString(),
		Type:      events.TicketCreated,
		OrgID:     input.OrgID,
		ActorID:   input.CreatedBy,
		Payload:   payload,
		CreatedAt: time.Now(),
	})

	return t, nil
}

func (s *Service) Get(ctx context.Context, id, orgID string) (*ticket.Ticket, error) {
	t, err := s.tickets.FindByID(ctx, id, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if t == nil {
		return nil, apperrors.NotFound(apperrors.CodeTicketNotFound, "ticket not found")
	}
	return t, nil
}

func (s *Service) List(ctx context.Context, orgID string, f ticket.Filter, limit, offset int) ([]*ticket.Ticket, int64, error) {
	tickets, total, err := s.tickets.List(ctx, orgID, f, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	return tickets, total, nil
}

func (s *Service) Update(ctx context.Context, id, orgID, actorID string, input ticket.UpdateInput) (*ticket.Ticket, error) {
	before, err := s.tickets.FindByID(ctx, id, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if before == nil {
		return nil, apperrors.NotFound(apperrors.CodeTicketNotFound, "ticket not found")
	}

	// Set resolved/closed timestamps based on status change.
	if input.Status != nil {
		now := time.Now()
		switch *input.Status {
		case ticket.StatusResolved:
			if before.ResolvedAt == nil {
				input.Status = input.Status // keep value
			}
		case ticket.StatusClosed:
			_ = now
		}
	}

	updated, err := s.tickets.Update(ctx, id, orgID, input)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	auditapp.WriteEntry(ctx, s.audits, audit.CreateInput{
		OrgID:        orgID,
		ActorID:      &actorID,
		ActorType:    audit.ActorUser,
		ResourceType: "ticket",
		ResourceID:   id,
		Action:       "ticket.updated",
		Before:       before,
		After:        updated,
	})

	payload, _ := events.MarshalPayload(updated)
	s.eventBus.Publish(ctx, events.Event{
		ID:        uuid.NewString(),
		Type:      events.TicketUpdated,
		OrgID:     orgID,
		ActorID:   actorID,
		Payload:   payload,
		CreatedAt: time.Now(),
	})

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id, orgID, actorID string) error {
	t, err := s.tickets.FindByID(ctx, id, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if t == nil {
		return apperrors.NotFound(apperrors.CodeTicketNotFound, "ticket not found")
	}

	if err := s.tickets.Delete(ctx, id, orgID); err != nil {
		return apperrors.Internal(err)
	}

	auditapp.WriteEntry(ctx, s.audits, audit.CreateInput{
		OrgID:        orgID,
		ActorID:      &actorID,
		ActorType:    audit.ActorUser,
		ResourceType: "ticket",
		ResourceID:   id,
		Action:       "ticket.deleted",
		Before:       t,
	})

	return nil
}
