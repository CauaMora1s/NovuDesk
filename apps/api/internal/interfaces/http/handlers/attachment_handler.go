package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/attachment"
	"github.com/novudesk/novudesk/internal/domain/storage"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type AttachmentHandler struct {
	attachments attachment.Repository
	storage     storage.Provider
}

func NewAttachmentHandler(attachments attachment.Repository, store storage.Provider) *AttachmentHandler {
	return &AttachmentHandler{attachments: attachments, storage: store}
}

// List returns all attachments for a ticket.
func (h *AttachmentHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	ticketID := chi.URLParam(r, "id")

	items, err := h.attachments.ListByTicket(r.Context(), ticketID, claims.OrgID)
	if err != nil {
		respond.Error(w, apperrors.Internal(err))
		return
	}

	for _, a := range items {
		a.URL = h.storage.PublicURL(a.StorageKey)
	}

	respond.Ok(w, items)
}

// Upload handles multipart file upload for a ticket.
func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	ticketID := chi.URLParam(r, "id")

	if err := r.ParseMultipartForm(storage.MaxFileSizeBytes); err != nil {
		respond.Error(w, apperrors.BadRequest("file too large or invalid multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respond.Error(w, apperrors.BadRequest("missing file field"))
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !storage.AllowedMIMETypes[mimeType] {
		respond.Error(w, apperrors.BadRequest(fmt.Sprintf("file type %q is not allowed", mimeType)))
		return
	}

	if header.Size > storage.MaxFileSizeBytes {
		respond.Error(w, apperrors.BadRequest("file exceeds the 25 MB limit"))
		return
	}

	fileID := uuid.NewString()
	ext := filepath.Ext(header.Filename)
	storageKey := fmt.Sprintf("attachments/%s/%s/%s%s", claims.OrgID, ticketID, fileID, ext)

	if err := h.storage.Upload(r.Context(), storageKey, file, storage.UploadOptions{
		ContentType: mimeType,
		SizeBytes:   header.Size,
	}); err != nil {
		respond.Error(w, apperrors.Internal(err))
		return
	}

	commentID := r.FormValue("comment_id")
	a := &attachment.Attachment{
		ID:         fileID,
		TicketID:   ticketID,
		OrgID:      claims.OrgID,
		UploaderID: &claims.UserID,
		Filename:   header.Filename,
		MimeType:   mimeType,
		SizeBytes:  header.Size,
		StorageKey: storageKey,
	}
	if commentID != "" {
		a.CommentID = &commentID
	}

	if err := h.attachments.Create(r.Context(), a); err != nil {
		_ = h.storage.Delete(r.Context(), storageKey)
		respond.Error(w, apperrors.Internal(err))
		return
	}

	a.URL = h.storage.PublicURL(storageKey)
	respond.Created(w, a)
}
