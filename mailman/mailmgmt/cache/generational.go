package cache

/*
 	Package cache provides generational caching utilities to solve the cache invalidation problem.

 	Generational caching allows you to instantly invalidate an entire group of cached items
 	without having to scan or delete individual keys. It works by appending a "generation" integer
 	to the cache key (e.g., "users.0.123"). When the group needs to be refreshed, the generation
 	number is incremented (e.g., "users.1.123"), instantly orphaning all old data.

 	# Example 1: Standard Key Generation and Retrieval

	func GetUser(ctx context.Context, userID string) (User, error) {
	    // 1. Generate the versioned key (e.g., "users.0.123")
	    key, err := cache.GenerationKey(ctx, "users", userID)
	    if err != nil {
	        return User{}, err
	    }

	    // 2. Attempt to fetch from cache
	    if val, err := cache.Get(ctx, key); err == nil {
	        return val.(User), nil
	    }

	    // 3. Cache miss: fetch from database and store
	    user := fetchUserFromDB(userID)
	    _ = cache.Set(ctx, key, user, time.Minute*15)

	    return user, nil
	}

 	# Example 2: Invalidating an Entire Group

	func InvalidateAllUsers(ctx context.Context) error {
	    // Increments the "users" generation counter (e.g., from 0 to 1).
	    // The 24h TTL ensures the counter itself doesn't cause a memory leak
	    // if the group is highly dynamic and rarely accessed.
	    _, err := cache.NextGeneration(ctx, "users", time.Hour*24)
	    return err
	}

 	# Example 3: Bulk Updating with the NextGeneration Closure

	func SeedNewProducts(ctx context.Context, products []Product) error {
	    // 1. Bump the generation to invalidate the old product catalog
	    genKeyFunc, err := cache.NextGeneration(ctx, "products", 0)
	    if err != nil {
	        return err
	    }

	    // 2. Use the returned closure to rapidly generate keys for the new generation
	    for _, p := range products {
	        // genKeyFunc generates "products.X.pid" without needing to hit the cache again
	        _ = cache.Set(ctx, genKeyFunc(p.ID), p, time.Hour)
	    }
	    return nil
	}

 	# Example 4: Using a Specific Cache Backend (Dependency Injection)

	func GetCustomData(ctx context.Context, myRedis cache.Cache, itemID string) {
	    // Use the ...FromCache variants to bypass cache.Default()
	    key, _ := cache.GenerationKeyFromCache(ctx, myRedis, "custom_group", itemID)
	    val, _ := myRedis.Get(ctx, key)
	    // ...
	}

*/

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/cache"
	"github.com/Nigel2392/go-django/src/core/errs"
)

type GenerationKey = string

type KeyGenerator = func(...string) GenerationKey

const (
	DEFAULT_GENERATION_KEY_PREFIX = "cache.generationalcache"
	KEY_DELIMITER                 = "."
)

func genGroupKey(s string) string {
	return fmt.Sprintf("%s.%s", DEFAULT_GENERATION_KEY_PREFIX, s)
}

func genKey(grp string, gen int64, key []string) GenerationKey {
	// return fmt.Sprintf("%s.%d.%s", grp, gen, strings.Join(key, "."))
	var sb = new(strings.Builder)
	var genStr = strconv.FormatInt(gen, 10)
	var l = len(grp) + len(genStr) + 1
	for _, k := range key {
		if k == "" {
			continue
		}
		l += len(k) + 1
	}
	sb.Grow(l)
	sb.WriteString(grp)
	sb.WriteString(KEY_DELIMITER)
	sb.WriteString(genStr)
	for _, k := range key {
		if k == "" {
			continue
		}
		sb.WriteString(KEY_DELIMITER)
		sb.WriteString(k)
	}
	return sb.String()
}

func genKeyFn(grp string, gen int64) KeyGenerator {
	return func(key ...string) GenerationKey {
		return genKey(grp, gen, key)
	}
}

func K(s ...string) []string {
	return s
}

func GetGenerationFromCache(ctx context.Context, c cache.Cache, group string) (int64, error) {
	var v, err = c.CounterValue(ctx, genGroupKey(group))
	if err != nil {
		if errors.Is(err, cache.ErrItemNotFound) {
			return 0, nil
		}
	}
	return v, err
}

func GenerationKeyFromCache(ctx context.Context, c cache.Cache, group string) (KeyGenerator, error) {
	var gen, err = GetGenerationFromCache(ctx, c, group)
	if err != nil {
		return nil, err
	}

	return genKeyFn(group, gen), nil
}

// RollOverFromCache increments the generation key in the cache.
// It also returns a GenerationKey.
func RollOverFromCache(ctx context.Context, c cache.Cache, group string, ttl cache.Duration) (KeyGenerator, error) {
	groupKey := genGroupKey(group)
	gen, err := c.Increment(ctx, groupKey, 1)
	if err != nil {
		return nil, err
	}

	if ttl > 0 {
		if err := c.Expire(ctx, groupKey, ttl); err != nil {
			return nil, err
		}
	}

	return genKeyFn(group, gen), nil
}

func GetItemFromCache[T any](ctx context.Context, c cache.Cache, ttl cache.Duration, gen int64, group string, key []string, get func(context.Context) (T, error)) (result T, err error) {
	if gen == -1 {
		gen, err = GetGenerationFromCache(ctx, c, group)
		if err != nil {
			return result, err
		}
	}

	var cacheKey = genKey(group, gen, key)
	v, err := cache.Get(ctx, cacheKey)
	isNotFound := errors.Is(err, cache.ErrItemNotFound)
	if err != nil && !isNotFound {
		return result, err
	}

	if v == nil || isNotFound {
		result, err = get(ctx)
		if err != nil {
			return result, err
		}

		err = c.Set(ctx, cacheKey, result, ttl)
		return result, err
	}

	result, ok := v.(T)
	if !ok {
		return result, errs.ErrInvalidType
	}
	return result, nil
}

func GetGen(ctx context.Context, group string) (int64, error) {
	return GetGenerationFromCache(ctx, cache.Default(), group)
}

func GetKey(ctx context.Context, group string, key ...string) (KeyGenerator, error) {
	return GenerationKeyFromCache(ctx, cache.Default(), group)
}

// RollOverFromCache increments the generation key in the cache.
// It also returns a GenerationKey generator.
func RollOver(ctx context.Context, group string, ttl cache.Duration) (KeyGenerator, error) {
	return RollOverFromCache(ctx, cache.Default(), group, ttl)
}

func GetItem[T any](ctx context.Context, ttl cache.Duration, gen int64, group string, key []string, get func(context.Context) (T, error)) (result T, err error) {
	return GetItemFromCache(ctx, cache.Default(), ttl, gen, group, key, get)
}
