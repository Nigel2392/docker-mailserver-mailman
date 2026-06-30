package cache

import (
	"time"

	"github.com/Nigel2392/go-django/src/core/cache"
)

var _ cache.Cache = (*MailMgmtCache)(nil)

type MailMgmtCache struct {
	Enabled    bool
	underlying cache.Cache
}

func NewMailMgmtCache(enabled bool, underlying cache.Cache) *MailMgmtCache {
	return &MailMgmtCache{
		Enabled:    enabled,
		underlying: underlying,
	}
}

func (m *MailMgmtCache) Get(key string) (interface{}, error) {
	if !m.Enabled {
		return nil, cache.ErrItemNotFound
	}
	return m.underlying.Get(key)
}
func (m *MailMgmtCache) GetDefault(key string, defaultValue interface{}) (interface{}, error) {
	if !m.Enabled {
		return defaultValue, nil
	}
	return m.underlying.GetDefault(key, defaultValue)
}
func (m *MailMgmtCache) Set(key string, value interface{}, ttl time.Duration) error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Set(key, value, ttl)
}
func (m *MailMgmtCache) TTL(key string) time.Duration {
	if !m.Enabled {
		return 0
	}
	return m.underlying.TTL(key)
}
func (m *MailMgmtCache) Has(key string) bool {
	if !m.Enabled {
		return false
	}
	return m.underlying.Has(key)
}
func (m *MailMgmtCache) Delete(key string) error {
	if !m.Enabled {
		return cache.ErrItemNotFound
	}
	return m.underlying.Delete(key)
}
func (m *MailMgmtCache) Keys() ([]string, error) {
	if !m.Enabled {
		return nil, nil
	}
	return m.underlying.Keys()
}
func (m *MailMgmtCache) Clear() error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Clear()
}
func (m *MailMgmtCache) Close() error {
	if !m.Enabled {
		return nil
	}
	return m.underlying.Close()
}
