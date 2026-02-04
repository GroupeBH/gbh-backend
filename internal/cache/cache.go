package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type NoopCache struct{}

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return nil, false, nil
}

func (n *NoopCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (n *NoopCache) Delete(ctx context.Context, key string) error {
	return nil
}
