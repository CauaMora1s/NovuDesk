package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/novudesk/novudesk/config"
)

func Connect(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return rdb, nil
}

// Key helpers keep all Redis key patterns in one place.

func RateLimitKey(ip, endpoint string) string {
	return fmt.Sprintf("ratelimit:%s:%s", ip, endpoint)
}

func SSEChannelKey(orgID string) string {
	return fmt.Sprintf("sse:org:%s", orgID)
}

func PermissionCacheKey(orgID, roleID string) string {
	return fmt.Sprintf("permissions:org:%s:role:%s", orgID, roleID)
}

func FeatureFlagsCacheKey(orgID string) string {
	return fmt.Sprintf("flags:org:%s", orgID)
}
