package cache

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/src/core/cache"
)

var _ cache.TransactionalCache = (*MailMgmtCache)(nil)

type MailMgmtCache struct {
	Enabled    bool
	underlying cache.TransactionalCache
}

func NewMailMgmtCache(enabled bool, underlying cache.TransactionalCache) *MailMgmtCache {
	return &MailMgmtCache{
		Enabled:    enabled,
		underlying: underlying,
	}
}
func (m *MailMgmtCache) Get(c context.Context, key string) (interface{}, error) {
	if !m.Enabled {
		return nil, cache.ErrItemNotFound
	}
	return m.underlying.Get(c, key)
}
func (m *MailMgmtCache) GetDefault(c context.Context, key string, defaultValue interface{}) (interface{}, error) {
	if !m.Enabled {
		return defaultValue, nil
	}
	return m.underlying.GetDefault(c, key, defaultValue)
}
func (m *MailMgmtCache) Set(c context.Context, key string, value interface{}, ttl time.Duration) error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Set(c, key, value, ttl)
}
func (m *MailMgmtCache) TTL(c context.Context, key string) time.Duration {
	if !m.Enabled {
		return 0
	}
	return m.underlying.TTL(c, key)
}
func (m *MailMgmtCache) Has(c context.Context, key string) bool {
	if !m.Enabled {
		return false
	}
	return m.underlying.Has(c, key)
}
func (m *MailMgmtCache) Delete(c context.Context, key string) error {
	if !m.Enabled {
		return cache.ErrItemNotFound
	}
	return m.underlying.Delete(c, key)
}
func (m *MailMgmtCache) Keys(c context.Context) ([]string, error) {
	if !m.Enabled {
		return nil, nil
	}
	return m.underlying.Keys(c)
}
func (m *MailMgmtCache) Clear(c context.Context) error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Clear(c)
}
func (m *MailMgmtCache) Close(c context.Context) error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Close(c)
}
func (m *MailMgmtCache) RunInTx(ctx context.Context, fn func(ctx context.Context, txCache cache.Transaction) error) error {
	return m.underlying.RunInTx(ctx, fn)
}
func (m *MailMgmtCache) CounterValue(c context.Context, key string) (int64, error) {
	if !m.Enabled {
		return 0, nil
	}
	return m.underlying.CounterValue(c, key)
}
func (m *MailMgmtCache) Decrement(c context.Context, key string, amount int64) (int64, error) {
	if !m.Enabled {
		return 0, nil
	}
	return m.underlying.Decrement(c, key, amount)
}
func (m *MailMgmtCache) Increment(c context.Context, key string, amount int64) (int64, error) {
	if !m.Enabled {
		return 0, nil
	}
	return m.underlying.Increment(c, key, amount)
}
func (m *MailMgmtCache) Expire(c context.Context, key string, ttl cache.Duration) error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Expire(c, key, ttl)
}
