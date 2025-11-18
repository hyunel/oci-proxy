package proxy

import (
	"sync"

	"oci-proxy/internal/pkg/config"
	"oci-proxy/internal/pkg/logging"
	"oci-proxy/internal/pkg/proxy/cache"
)

type CacheManager struct {
	cfg    *config.Config
	caches map[string]*cache.Cache
	mu     sync.RWMutex
}

func NewCacheManager(cfg *config.Config) *CacheManager {
	return &CacheManager{
		cfg:    cfg,
		caches: make(map[string]*cache.Cache),
	}
}

func (cm *CacheManager) GetCache(registryHost string) *cache.Cache {
	cm.mu.RLock()
	c, ok := cm.caches[registryHost]
	cm.mu.RUnlock()
	if ok {
		return c
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok = cm.caches[registryHost]
	if ok {
		return c
	}

	settings := cm.cfg.GetRegistrySettings(registryHost)
	newCache, err := cache.NewLRUCache(settings.CacheMaxSize.Bytes(), settings.CacheDir)
	if err != nil {
		logging.Logger.Error("failed to create cache for registry", "registry", registryHost, "error", err)
		newCache, _ = cache.NewLRUCache(0, "")
	}

	cm.caches[registryHost] = newCache
	logging.Logger.Debug("initialized cache for registry", "registry", registryHost)
	return newCache
}

func (cm *CacheManager) PersistAll() {
	cm.mu.RLock()
	caches := make([]*cache.Cache, 0, len(cm.caches))
	for _, c := range cm.caches {
		caches = append(caches, c)
	}
	cm.mu.RUnlock()

	for _, c := range caches {
		if err := c.Persist(); err != nil {
			logging.Logger.Error("failed to persist cache", "error", err)
		}
	}
}

func (cm *CacheManager) GetStats() map[string]cache.CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]cache.CacheStats, len(cm.caches))
	for host, c := range cm.caches {
		stats[host] = c.Stats()
	}
	return stats
}
