package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/audit"
)

// WriteEntry converts a CreateInput into an audit.Log and persists it via the repository.
// Errors are intentionally ignored — audit failures must not block the main operation.
func WriteEntry(ctx context.Context, repo audit.Repository, input audit.CreateInput) {
	if repo == nil {
		return
	}
	beforeJSON, _ := json.Marshal(input.Before)
	afterJSON, _ := json.Marshal(input.After)
	metaJSON, _ := json.Marshal(input.Metadata)

	_ = repo.Create(ctx, &audit.Log{
		ID:           uuid.NewString(),
		OrgID:        input.OrgID,
		ActorID:      input.ActorID,
		ActorType:    input.ActorType,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Action:       input.Action,
		Before:       beforeJSON,
		After:        afterJSON,
		Metadata:     metaJSON,
	})
}
