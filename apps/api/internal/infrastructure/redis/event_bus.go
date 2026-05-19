package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/novudesk/novudesk/pkg/events"
)

// EventBus implements events.Bus using Redis Pub/Sub for SSE fan-out
// and Redis Streams for durable async job queues.
type EventBus struct {
	rdb    *redis.Client
	logger *slog.Logger
}

func NewEventBus(rdb *redis.Client, logger *slog.Logger) *EventBus {
	return &EventBus{rdb: rdb, logger: logger}
}

// Publish serializes the event and publishes it to the org SSE channel.
// Workers that need durable delivery use XAdd on dedicated stream keys instead.
func (b *EventBus) Publish(ctx context.Context, event events.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	channel := SSEChannelKey(event.OrgID)
	if err := b.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		b.logger.Error("redis publish failed", "channel", channel, "error", err)
		return err
	}

	return nil
}

// Subscribe listens on the Redis pub/sub channel for this org.
// Intended for the SSE handler only.
func (b *EventBus) Subscribe(ctx context.Context, orgID string, handler func(events.Event) error) error {
	channel := SSEChannelKey(orgID)
	sub := b.rdb.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			var event events.Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				b.logger.Warn("failed to unmarshal SSE event", "error", err)
				continue
			}
			if err := handler(event); err != nil {
				b.logger.Warn("SSE handler returned error", "error", err)
			}
		}
	}
}

// QueueJob pushes a job to a Redis stream for durable async processing.
func (b *EventBus) QueueJob(ctx context.Context, stream string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}
	return b.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"data": string(data)},
	}).Err()
}

// Stream key constants for all async queues.
const (
	StreamAutomations = "queue:automations"
	StreamWebhooks    = "queue:webhooks"
	StreamEmails      = "queue:emails"
)
