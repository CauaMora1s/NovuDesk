package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/novudesk/novudesk/internal/infrastructure/redis"
	"github.com/novudesk/novudesk/pkg/events"
)

// Manager handles all active SSE connections and fan-out via Redis pub/sub.
type Manager struct {
	bus    *redis.EventBus
	logger *slog.Logger
	mu     sync.RWMutex
	conns  map[string]map[string]chan events.Event // orgID → connID → channel
}

func NewManager(bus *redis.EventBus, logger *slog.Logger) *Manager {
	return &Manager{
		bus:    bus,
		logger: logger,
		conns:  make(map[string]map[string]chan events.Event),
	}
}

// ServeHTTP serves a long-lived SSE stream for the authenticated user.
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request, orgID, connID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	ch := make(chan events.Event, 32)
	m.register(orgID, connID, ch)
	defer m.unregister(orgID, connID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send a connected confirmation event immediately.
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ctx := r.Context()

	// Subscribe to Redis channel in a goroutine.
	go func() {
		m.bus.Subscribe(ctx, orgID, func(event events.Event) error {
			select {
			case ch <- event:
			default:
				m.logger.Warn("SSE channel full, dropping event", "conn_id", connID)
			}
			return nil
		})
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}

func (m *Manager) register(orgID, connID string, ch chan events.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conns[orgID] == nil {
		m.conns[orgID] = make(map[string]chan events.Event)
	}
	m.conns[orgID][connID] = ch
}

func (m *Manager) unregister(orgID, connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if org, ok := m.conns[orgID]; ok {
		close(org[connID])
		delete(org, connID)
	}
}

// ConnectionCount returns active connections per org (useful for health/metrics).
func (m *Manager) ConnectionCount(orgID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns[orgID])
}
