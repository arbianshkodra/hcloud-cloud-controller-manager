//go:build eval

package hcloud

import (
	"context"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func init() {
	cacheInitializers[string(config.InstanceCacheModeEval)] = func(c *hcloud.Client, ttl time.Duration) ServerCache {
		return NewEvalCache(c, ttl)
	}
}

// ----- EvalCache -----

// EvalCache wraps a [PerServerCache], an [AllServerCache], and a [NoCache] and runs both in
// parallel for every lookup. Each underlying cache emits CacheRequests metrics
// under a distinct "cache" label, so the strategies can be compared side-by-side under a real workload.

var _ ServerCache = (*EvalCache)(nil)

type EvalCache struct {
	name   string
	caches []ServerCache
	client *hcloud.Client
}

func NewEvalCache(client *hcloud.Client, ttl time.Duration) *EvalCache {
	caches := []ServerCache{
		NewAllServerCache(client, ttl),
		NewPerServerCache(client, ttl),
		NewNoCache(client),
	}

	return &EvalCache{
		name:   "eval",
		caches: caches,
		client: client,
	}
}

func (c *EvalCache) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	return c.run(func(s ServerCache) (*hcloud.Server, error) { return s.ByID(ctx, id) })
}

func (c *EvalCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	return c.run(func(s ServerCache) (*hcloud.Server, error) { return s.ByName(ctx, name) })
}

func (c *EvalCache) run(lookup func(ServerCache) (*hcloud.Server, error)) (*hcloud.Server, error) {
	var server *hcloud.Server
	var err error

	for _, cache := range c.caches {
		server, err = lookup(cache)
		if err != nil {
			return nil, err
		}
		if server == nil {
			return nil, nil
		}
	}

	return server, err
}
