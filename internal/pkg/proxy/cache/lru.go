package cache

import (
	"bufio"
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"oci-proxy/internal/pkg/logging"
)

// entry is used to hold a value in the cache.
type entry struct {
	Key        string    `json:"key"`
	Size       int64     `json:"size"`
	LastAccess time.Time `json:"last_access"`
}

// CacheStats provides statistics about cache usage.
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Items       int
	CurrentSize int64
	MaxSize     int64
}

type Cache struct {
	maxSize  int64
	size     atomic.Int64
	ll       *list.List
	cache    map[string]*list.Element
	mu       sync.RWMutex
	cacheDir string

	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64

	persistMu    sync.Mutex
	lastPersist  time.Time
	persistDirty atomic.Bool
}

func NewLRUCache(maxSize int64, cacheDir string) (*Cache, error) {
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	c := &Cache{
		maxSize:  maxSize,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
		cacheDir: cacheDir,
	}

	if err := c.load(); err != nil {
		logging.Logger.Warn("could not load cache persistence, starting fresh", "path", c.persistencePath(), "error", err)
	}

	return c, nil
}

func (c *Cache) persistencePath() string {
	if c.cacheDir == "" {
		return ""
	}
	return filepath.Join(c.cacheDir, ".lru_persistence")
}

func (c *Cache) GetReader(key string) (io.ReadCloser, int64, bool) {
	c.mu.Lock()
	ee, exists := c.cache[key]
	if !exists {
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, 0, false
	}

	c.ll.MoveToFront(ee)
	e := ee.Value.(*entry)
	e.LastAccess = time.Now()
	size := e.Size
	filePath := filepath.Join(c.cacheDir, key)
	c.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		logging.Logger.Warn("file in cache but not on disk, removing", "key", key, "path", filePath, "error", err)
		c.mu.Lock()
		if ee, exists := c.cache[key]; exists {
			c.removeElementLocked(ee)
		}
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, 0, false
	}

	c.hits.Add(1)
	c.persistDirty.Store(true)
	return file, size, true
}

func (c *Cache) Put(key string, reader io.Reader, expectedDigest string) error {
	if c.cacheDir == "" {
		_, err := io.Copy(io.Discard, reader)
		return err
	}

	tmpFile, err := os.CreateTemp(c.cacheDir, "blob-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	hasher := sha256.New()
	size, err := io.Copy(tmpFile, io.TeeReader(reader, hasher))
	if err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	actualDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualDigest != expectedDigest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", expectedDigest, actualDigest)
	}

	if c.maxSize > 0 && size > c.maxSize {
		logging.Logger.Warn("file size exceeds max cache size, skipping cache", "key", key, "size", size, "maxSize", c.maxSize)
		return nil
	}

	finalPath := filepath.Join(c.cacheDir, key)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to move cached file: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		e := ee.Value.(*entry)
		oldSize := e.Size
		e.Size = size
		e.LastAccess = time.Now()
		c.size.Add(size - oldSize)
	} else {
		e := &entry{
			Key:        key,
			Size:       size,
			LastAccess: time.Now(),
		}
		ee := c.ll.PushFront(e)
		c.cache[key] = ee
		c.size.Add(size)
	}

	c.evictIfNeeded()
	c.persistDirty.Store(true)
	return nil
}

func (c *Cache) evictIfNeeded() {
	if c.maxSize <= 0 {
		return
	}

	var toEvict []*entry
	for c.size.Load() > c.maxSize {
		oldest := c.ll.Back()
		if oldest == nil {
			break
		}
		removedEntry := c.removeElementLocked(oldest)
		toEvict = append(toEvict, removedEntry)
		c.evictions.Add(1)
	}

	if len(toEvict) > 0 {
		c.mu.Unlock()
		c.deleteFiles(toEvict)
		c.mu.Lock()
	}
}

func (c *Cache) deleteFiles(entries []*entry) {
	for _, entry := range entries {
		filePath := filepath.Join(c.cacheDir, entry.Key)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logging.Logger.Warn("failed to remove cache file", "path", filePath, "error", err)
		} else {
			logging.Logger.Debug("evicted cache file", "key", entry.Key, "size", entry.Size)
		}
	}
}

func (c *Cache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ee, ok := c.cache[key]; ok {
		c.removeElementLocked(ee)
		filePath := filepath.Join(c.cacheDir, key)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logging.Logger.Warn("failed to remove cache file", "path", filePath, "error", err)
		}
		c.persistDirty.Store(true)
	}
}

func (c *Cache) removeElementLocked(e *list.Element) *entry {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.Key)
	c.size.Add(-kv.Size)
	return kv
}

func (c *Cache) Persist() error {
	if !c.persistDirty.Load() {
		return nil
	}

	c.persistMu.Lock()
	defer c.persistMu.Unlock()

	path := c.persistencePath()
	if path == "" {
		return nil
	}

	c.mu.RLock()
	entries := make([]*entry, 0, c.ll.Len())
	for e := c.ll.Back(); e != nil; e = e.Prev() {
		entries = append(entries, e.Value.(*entry))
	}
	c.mu.RUnlock()

	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".lru_persistence.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp persistence file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	writer := bufio.NewWriter(tmpFile)
	encoder := json.NewEncoder(writer)

	for _, e := range entries {
		if err := encoder.Encode(e); err != nil {
			return fmt.Errorf("failed to encode entry: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	tmpFile.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename persistence file: %w", err)
	}

	c.persistDirty.Store(false)
	c.lastPersist = time.Now()
	return nil
}

func (c *Cache) load() error {
	path := c.persistencePath()
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var validEntries []*entry
	skippedEntries := 0

	for scanner.Scan() {
		var e entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			logging.Logger.Warn("failed to unmarshal cache entry, skipping", "error", err)
			skippedEntries++
			continue
		}

		filePath := filepath.Join(c.cacheDir, e.Key)
		stat, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				logging.Logger.Debug("file in persistence but not on disk, skipping", "key", e.Key)
			} else {
				logging.Logger.Warn("failed to stat cached file, skipping", "key", e.Key, "error", err)
			}
			skippedEntries++
			continue
		}

		if stat.Size() != e.Size {
			logging.Logger.Warn("cached file size mismatch, removing", "key", e.Key, "expected", e.Size, "actual", stat.Size())
			os.Remove(filePath)
			skippedEntries++
			continue
		}

		validEntries = append(validEntries, &e)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan persistence file: %w", err)
	}

	c.mu.Lock()
	var totalSize int64
	for _, e := range validEntries {
		element := c.ll.PushFront(e)
		c.cache[e.Key] = element
		totalSize += e.Size
	}
	c.size.Add(totalSize)
	c.mu.Unlock()

	logging.Logger.Info("loaded cache from persistence", "loaded", len(validEntries), "skipped", skippedEntries, "size", c.size.Load())
	return nil
}

func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:        c.hits.Load(),
		Misses:      c.misses.Load(),
		Evictions:   c.evictions.Load(),
		Items:       c.ll.Len(),
		CurrentSize: c.size.Load(),
		MaxSize:     c.maxSize,
	}
}

func (c *Cache) CurrentSize() int64 {
	return c.size.Load()
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ll.Len()
}

func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.cache {
		filePath := filepath.Join(c.cacheDir, key)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logging.Logger.Warn("failed to remove cache file during clear", "path", filePath, "error", err)
		}
	}

	c.ll.Init()
	c.cache = make(map[string]*list.Element)
	c.size.Store(0)
	c.persistDirty.Store(true)

	return nil
}
