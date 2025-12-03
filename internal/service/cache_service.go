package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CacheService provides in-memory caching with TTL and invalidation support.
type CacheService struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewCacheService creates a new cache service.
func NewCacheService() *CacheService {
	cs := &CacheService{
		cache: make(map[string]*cacheEntry),
	}

	// Start background cleanup goroutine
	go cs.cleanup()

	return cs
}

// Get retrieves a value from cache.
func (cs *CacheService) Get(key string) (interface{}, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	entry, exists := cs.cache[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		// Don't delete here, let cleanup handle it
		return nil, false
	}

	return entry.data, true
}

// Set stores a value in cache with TTL.
func (cs *CacheService) Set(key string, value interface{}, ttl time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.cache[key] = &cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a key from cache.
func (cs *CacheService) Delete(key string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.cache, key)
}

// InvalidateByPrefix removes all keys with the given prefix.
func (cs *CacheService) InvalidateByPrefix(prefix string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for key := range cs.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(cs.cache, key)
		}
	}
}

// InvalidateUserCache removes all cache entries for a specific user.
func (cs *CacheService) InvalidateUserCache(userID uuid.UUID) {
	cs.InvalidateByPrefix("dashboard:" + userID.String() + ":")
	cs.InvalidateByPrefix("stats:" + userID.String() + ":")
	cs.InvalidateByPrefix("orders:" + userID.String() + ":")
	cs.InvalidateByPrefix("ai_recommendations:" + userID.String() + ":")
}

// InvalidateOrderCache removes cache entries related to an order.
func (cs *CacheService) InvalidateOrderCache(orderID uuid.UUID) {
	cs.InvalidateByPrefix("order:" + orderID.String() + ":")
	cs.InvalidateByPrefix("ai_recommendations:order:" + orderID.String() + ":")
	// Also invalidate all dashboard caches that might include this order
	cs.InvalidateByPrefix("dashboard:")
}

// cleanup removes expired entries periodically.
func (cs *CacheService) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cs.mu.Lock()
		now := time.Now()
		for key, entry := range cs.cache {
			if now.After(entry.expiresAt) {
				delete(cs.cache, key)
			}
		}
		cs.mu.Unlock()
	}
}

// Cache key generators
func DashboardCacheKey(userID uuid.UUID, includeAI bool) string {
	aiFlag := "false"
	if includeAI {
		aiFlag = "true"
	}
	return "dashboard:" + userID.String() + ":" + aiFlag
}

func StatsCacheKey(userID uuid.UUID) string {
	return "stats:" + userID.String()
}

func AIRecommendationsCacheKey(userID uuid.UUID, userRole string) string {
	return "ai_recommendations:" + userID.String() + ":" + userRole
}

func SuitableFreelancersCacheKey(orderID uuid.UUID) string {
	return "ai_recommendations:order:" + orderID.String() + ":suitable_freelancers"
}

// GetOrSet retrieves a value from cache or computes it if not found.
func (cs *CacheService) GetOrSet(
	ctx context.Context,
	key string,
	ttl time.Duration,
	fn func() (interface{}, error),
) (interface{}, error) {
	// Try to get from cache
	if value, found := cs.Get(key); found {
		return value, nil
	}

	// Compute value
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.Set(key, value, ttl)

	return value, nil
}

