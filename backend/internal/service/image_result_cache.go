package service

import (
	"fmt"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

const (
	imageResultCacheTTL     = 10 * time.Minute
	imageResultCacheCleanup = 15 * time.Minute
)

type imageResultEntry struct {
	statusCode  int
	contentType string
	body        []byte
}

type imageResultCache struct {
	c *gocache.Cache
}

func newImageResultCache() *imageResultCache {
	return &imageResultCache{c: gocache.New(imageResultCacheTTL, imageResultCacheCleanup)}
}

func (r *imageResultCache) key(userID int64, bodyHash, endpoint string) string {
	return fmt.Sprintf("img:%d:%s:%s", userID, bodyHash, endpoint)
}

func (r *imageResultCache) get(userID int64, bodyHash, endpoint string) (*imageResultEntry, bool) {
	v, ok := r.c.Get(r.key(userID, bodyHash, endpoint))
	if !ok {
		return nil, false
	}
	e, ok := v.(*imageResultEntry)
	return e, ok
}

func (r *imageResultCache) set(userID int64, bodyHash, endpoint string, e *imageResultEntry) {
	r.c.Set(r.key(userID, bodyHash, endpoint), e, gocache.DefaultExpiration)
}
