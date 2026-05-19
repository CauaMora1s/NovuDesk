package team

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/team"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type Service struct {
	teams team.Repository
}

func NewService(teams team.Repository) *Service {
	return &Service{teams: teams}
}

func (s *Service) Create(ctx context.Context, orgID, name string, description *string) (*team.Team, error) {
	t := &team.Team{
		ID:          uuid.NewString(),
		OrgID:       orgID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.teams.Create(ctx, t); err != nil {
		return nil, apperrors.Internal(err)
	}
	return t, nil
}

func (s *Service) List(ctx context.Context, orgID string) ([]*team.Team, error) {
	teams, err := s.teams.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return teams, nil
}

func (s *Service) Get(ctx context.Context, id, orgID string) (*team.Team, error) {
	t, err := s.teams.FindByID(ctx, id, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if t == nil {
		return nil, apperrors.NotFound(apperrors.CodeTeamNotFound, "team not found")
	}
	return t, nil
}

func (s *Service) Update(ctx context.Context, id, orgID string, input team.UpdateInput) (*team.Team, error) {
	t, err := s.teams.Update(ctx, id, orgID, input)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if t == nil {
		return nil, apperrors.NotFound(apperrors.CodeTeamNotFound, "team not found")
	}
	return t, nil
}

func (s *Service) Delete(ctx context.Context, id, orgID string) error {
	if err := s.teams.Delete(ctx, id, orgID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) AddMember(ctx context.Context, teamID, userID string) error {
	if err := s.teams.AddMember(ctx, teamID, userID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, teamID, userID string) error {
	if err := s.teams.RemoveMember(ctx, teamID, userID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) ListMembers(ctx context.Context, teamID, orgID string) ([]*team.Member, error) {
	members, err := s.teams.ListMembers(ctx, teamID, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return members, nil
}
