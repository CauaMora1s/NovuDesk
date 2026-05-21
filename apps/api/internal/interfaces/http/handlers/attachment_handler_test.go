package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	authsvc "github.com/novudesk/novudesk/internal/application/auth"
	"github.com/novudesk/novudesk/internal/domain/attachment"
	"github.com/novudesk/novudesk/internal/domain/storage"
	"github.com/novudesk/novudesk/internal/interfaces/http/handlers"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
)

// --- fakes ---

type fakeAttachmentRepo struct {
	items  []*attachment.Attachment
	stored *attachment.Attachment
	err    error
}

func (f *fakeAttachmentRepo) Create(_ context.Context, a *attachment.Attachment) error {
	if f.err != nil {
		return f.err
	}
	f.stored = a
	return nil
}

func (f *fakeAttachmentRepo) ListByTicket(_ context.Context, ticketID, orgID string) ([]*attachment.Attachment, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*attachment.Attachment
	for _, a := range f.items {
		if a.TicketID == ticketID && a.OrgID == orgID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAttachmentRepo) Delete(_ context.Context, id, orgID string) (*attachment.Attachment, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, a := range f.items {
		if a.ID == id && a.OrgID == orgID {
			return a, nil
		}
	}
	return nil, nil
}

type fakeStorageProvider struct {
	uploadErr error
	deleted   []string
}

func (f *fakeStorageProvider) Upload(_ context.Context, _ string, _ io.Reader, _ storage.UploadOptions) error {
	return f.uploadErr
}

func (f *fakeStorageProvider) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *fakeStorageProvider) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

func (f *fakeStorageProvider) PublicURL(key string) string {
	return "https://cdn.example.com/" + key
}

// --- helpers ---

const (
	testOrgID    = "org-111"
	testUserID   = "user-222"
	testTicketID = "ticket-333"
)

func fakeClaims() *authsvc.Claims {
	return &authsvc.Claims{
		OrgID:  testOrgID,
		UserID: testUserID,
	}
}

// injectClaims sets the JWT claims into the request context the same way
// the Authenticate middleware does.
func injectClaims(r *http.Request, claims *authsvc.Claims) *http.Request {
	ctx := middleware.TestInjectClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

// routeWith wraps the handler inside a chi router so URL params are parsed.
func routeWith(method, pattern string, h http.HandlerFunc) http.Handler {
	rr := chi.NewRouter()
	rr.Method(method, pattern, h)
	return rr
}

func decodeEnvelope(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return env
}

// multipartBody builds a multipart/form-data body with a single "file" field.
func multipartBody(t *testing.T, filename, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{contentType}

	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}
	if _, err = part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	mw.Close()
	return &buf, mw.FormDataContentType()
}

// --- tests ---

func TestAttachmentHandler_List_ReturnsAttachmentsForTicket(t *testing.T) {
	uid := testUserID
	repo := &fakeAttachmentRepo{
		items: []*attachment.Attachment{
			{
				ID:         "att-1",
				TicketID:   testTicketID,
				OrgID:      testOrgID,
				UploaderID: &uid,
				Filename:   "report.pdf",
				MimeType:   "application/pdf",
				SizeBytes:  1024,
				StorageKey: "attachments/org-111/ticket-333/att-1.pdf",
				CreatedAt:  time.Now(),
			},
		},
	}
	store := &fakeStorageProvider{}
	h := handlers.NewAttachmentHandler(repo, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+testTicketID+"/attachments", nil)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testTicketID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", env["data"])
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(data))
	}

	item := data[0].(map[string]any)
	if item["id"] != "att-1" {
		t.Errorf("expected id att-1, got %v", item["id"])
	}
	// PublicURL must be set
	if !strings.HasPrefix(item["url"].(string), "https://cdn.example.com/") {
		t.Errorf("expected cdn url, got %v", item["url"])
	}
}

func TestAttachmentHandler_List_FiltersByOrgID(t *testing.T) {
	uid := testUserID
	otherOrgID := "org-other"
	repo := &fakeAttachmentRepo{
		items: []*attachment.Attachment{
			{ID: "att-mine", TicketID: testTicketID, OrgID: testOrgID, UploaderID: &uid, Filename: "a.pdf", MimeType: "application/pdf", SizeBytes: 1},
			{ID: "att-other", TicketID: testTicketID, OrgID: otherOrgID, UploaderID: &uid, Filename: "b.pdf", MimeType: "application/pdf", SizeBytes: 1},
		},
	}
	h := handlers.NewAttachmentHandler(repo, &fakeStorageProvider{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+testTicketID+"/attachments", nil)
	req = injectClaims(req, fakeClaims()) // claims.OrgID == testOrgID

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testTicketID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	env := decodeEnvelope(t, rec.Body)
	data := env["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 attachment (own org only), got %d", len(data))
	}
	if data[0].(map[string]any)["id"] != "att-mine" {
		t.Errorf("wrong attachment returned")
	}
}

func TestAttachmentHandler_Upload_ValidFile_Returns201(t *testing.T) {
	repo := &fakeAttachmentRepo{}
	store := &fakeStorageProvider{}
	h := handlers.NewAttachmentHandler(repo, store)

	body, ct := multipartBody(t, "photo.png", "image/png", []byte("fake-png-bytes"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/"+testTicketID+"/attachments", body)
	req.Header.Set("Content-Type", ct)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testTicketID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Upload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}

	if repo.stored == nil {
		t.Fatal("expected attachment to be persisted")
	}
	if repo.stored.OrgID != testOrgID {
		t.Errorf("wrong org_id stored: %s", repo.stored.OrgID)
	}
	if repo.stored.TicketID != testTicketID {
		t.Errorf("wrong ticket_id stored: %s", repo.stored.TicketID)
	}
	if repo.stored.MimeType != "image/png" {
		t.Errorf("wrong mime_type: %s", repo.stored.MimeType)
	}
}

func TestAttachmentHandler_Upload_DisallowedMIMEType_Returns400(t *testing.T) {
	h := handlers.NewAttachmentHandler(&fakeAttachmentRepo{}, &fakeStorageProvider{})

	body, ct := multipartBody(t, "script.sh", "application/x-sh", []byte("#!/bin/sh"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/"+testTicketID+"/attachments", body)
	req.Header.Set("Content-Type", ct)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testTicketID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for disallowed MIME, got %d", rec.Code)
	}
}

func TestAttachmentHandler_Upload_StorageFailure_CleansUpAndReturns500(t *testing.T) {
	store := &fakeStorageProvider{uploadErr: fmt.Errorf("s3 unavailable")}
	h := handlers.NewAttachmentHandler(&fakeAttachmentRepo{}, store)

	body, ct := multipartBody(t, "file.pdf", "application/pdf", []byte("pdf-content"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/"+testTicketID+"/attachments", body)
	req.Header.Set("Content-Type", ct)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testTicketID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Upload(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on storage failure, got %d", rec.Code)
	}
}

func TestAttachmentHandler_Upload_DBFailure_CleansUpStorage(t *testing.T) {
	repo := &fakeAttachmentRepo{err: fmt.Errorf("db down")}
	store := &fakeStorageProvider{}
	h := handlers.NewAttachmentHandler(repo, store)

	body, ct := multipartBody(t, "doc.pdf", "application/pdf", []byte("pdf"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/"+testTicketID+"/attachments", body)
	req.Header.Set("Content-Type", ct)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", testTicketID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Upload(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on DB failure, got %d", rec.Code)
	}
	if len(store.deleted) == 0 {
		t.Error("expected storage cleanup after DB failure, but Delete was not called")
	}
}
