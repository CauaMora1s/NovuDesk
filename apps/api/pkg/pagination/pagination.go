package pagination

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPerPage = 25
	MaxPerPage     = 100
)

// Params holds parsed pagination parameters from the request.
type Params struct {
	PerPage int
	Cursor  *Cursor
	// Page-based fallback
	Page int
}

// Cursor encodes the last-seen position for cursor-based pagination.
type Cursor struct {
	ID        string
	CreatedAt time.Time
}

// Meta is included in list responses.
type Meta struct {
	Total      int64  `json:"total"`
	PerPage    int    `json:"per_page"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ParseRequest extracts pagination params from the HTTP request.
func ParseRequest(r *http.Request) Params {
	q := r.URL.Query()

	perPage := parseInt(q.Get("per_page"), DefaultPerPage)
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}

	p := Params{PerPage: perPage}

	if raw := q.Get("cursor"); raw != "" {
		if c, err := decodeCursor(raw); err == nil {
			p.Cursor = c
		}
	}

	if page := parseInt(q.Get("page"), 0); page > 0 {
		p.Page = page
	}

	return p
}

// EncodeCursor creates an opaque cursor string from the last item in a page.
func EncodeCursor(id string, createdAt time.Time) string {
	raw := fmt.Sprintf("%s|%d", id, createdAt.UnixMicro())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(encoded string) (*Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}

	us, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	return &Cursor{
		ID:        parts[0],
		CreatedAt: time.UnixMicro(us),
	}, nil
}

// Offset computes the SQL OFFSET from page-based params.
func (p *Params) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func parseInt(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
