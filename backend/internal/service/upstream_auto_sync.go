package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const (
	upstreamAutoSyncStartupDelay      = 20 * time.Second
	upstreamAutoSyncSchedulerTick     = 30 * time.Second
	upstreamAutoSyncWorkerConcurrency = 2
	upstreamAutoSyncDefaultInterval   = 30 * time.Minute
	upstreamAutoSyncMinInterval       = time.Minute
	upstreamAutoSyncMaxInterval       = 24 * time.Hour
)

type upstreamAutoSyncConfig struct {
	Enabled  bool
	Interval time.Duration
}

type UpstreamAutoSyncRunner struct {
	svc *UpstreamService

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu       sync.Mutex
	nextDue  map[int64]time.Time
	inFlight map[int64]struct{}
	started  bool
	stopped  bool
	sem      chan struct{}
}

func NewUpstreamAutoSyncRunner(svc *UpstreamService) *UpstreamAutoSyncRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &UpstreamAutoSyncRunner{
		svc:      svc,
		ctx:      ctx,
		cancel:   cancel,
		nextDue:  map[int64]time.Time{},
		inFlight: map[int64]struct{}{},
		sem:      make(chan struct{}, upstreamAutoSyncWorkerConcurrency),
	}
}

func (r *UpstreamAutoSyncRunner) Start() {
	if r == nil || r.svc == nil {
		return
	}
	r.mu.Lock()
	if r.started || r.stopped {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.wg.Add(1)
	r.mu.Unlock()
	go r.loop()
}

func (r *UpstreamAutoSyncRunner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	r.cancel()
	r.mu.Unlock()
	r.wg.Wait()
}

func (r *UpstreamAutoSyncRunner) loop() {
	defer r.wg.Done()
	timer := time.NewTimer(upstreamAutoSyncStartupDelay)
	defer timer.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-timer.C:
			r.tick()
			timer.Reset(upstreamAutoSyncSchedulerTick)
		}
	}
}

func (r *UpstreamAutoSyncRunner) tick() {
	if r == nil || r.svc == nil || r.svc.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.ctx, 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	page := 1
	for {
		items, _, err := r.svc.repo.List(ctx, UpstreamListParams{Page: page, PageSize: 100})
		if err != nil {
			slog.Warn("upstream_auto_sync: list upstreams failed", "error", err)
			return
		}
		for _, upstream := range items {
			if upstream == nil {
				continue
			}
			cfg := parseUpstreamAutoSyncConfig(upstream)
			if !cfg.Enabled {
				r.forget(upstream.ID)
				continue
			}
			if upstream.Status == UpstreamStatusDisabled || upstream.Status == UpstreamStatusCircuitOpen {
				continue
			}
			if !r.isDue(upstream, cfg.Interval, now) {
				continue
			}
			r.submit(upstream.ID)
		}
		if len(items) < 100 {
			return
		}
		page++
	}
}

func (r *UpstreamAutoSyncRunner) isDue(upstream *Upstream, interval time.Duration, now time.Time) bool {
	if upstream == nil || upstream.ID <= 0 {
		return false
	}
	if interval <= 0 {
		interval = upstreamAutoSyncDefaultInterval
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.inFlight[upstream.ID]; ok {
		return false
	}
	next := r.nextDue[upstream.ID]
	if next.IsZero() {
		if upstream.LastSyncedAt != nil && !upstream.LastSyncedAt.IsZero() {
			next = upstream.LastSyncedAt.Add(interval)
		}
	}
	if !next.IsZero() && now.Before(next) {
		r.nextDue[upstream.ID] = next
		return false
	}
	r.inFlight[upstream.ID] = struct{}{}
	r.nextDue[upstream.ID] = now.Add(interval)
	return true
}

func (r *UpstreamAutoSyncRunner) submit(upstreamID int64) {
	select {
	case r.sem <- struct{}{}:
	case <-r.ctx.Done():
		r.release(upstreamID)
		return
	default:
		r.release(upstreamID)
		slog.Warn("upstream_auto_sync: worker pool full, skip", "upstream_id", upstreamID)
		return
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			<-r.sem
			r.release(upstreamID)
			if rec := recover(); rec != nil {
				slog.Error("upstream_auto_sync: panic", "upstream_id", upstreamID, "panic", rec)
			}
		}()
		ctx, cancel := context.WithTimeout(r.ctx, 2*time.Minute)
		defer cancel()
		if _, err := r.svc.SyncRemoteResources(ctx, upstreamID); err != nil {
			slog.Warn("upstream_auto_sync: sync failed", "upstream_id", upstreamID, "error", err)
		}
	}()
}

func (r *UpstreamAutoSyncRunner) release(upstreamID int64) {
	r.mu.Lock()
	delete(r.inFlight, upstreamID)
	r.mu.Unlock()
}

func (r *UpstreamAutoSyncRunner) forget(upstreamID int64) {
	r.mu.Lock()
	delete(r.nextDue, upstreamID)
	r.mu.Unlock()
}

func parseUpstreamAutoSyncConfig(upstream *Upstream) upstreamAutoSyncConfig {
	cfg := upstreamAutoSyncConfig{Interval: upstreamAutoSyncDefaultInterval}
	if upstream == nil || upstream.Metadata == nil {
		return cfg
	}
	cfg.Enabled = parseMetadataBool(upstream.Metadata["auto_sync_enabled"])
	if !cfg.Enabled {
		return cfg
	}
	seconds := parseAnyInt64(upstream.Metadata["auto_sync_interval_seconds"])
	if seconds <= 0 {
		minutes := parseAnyInt64(upstream.Metadata["auto_sync_interval_minutes"])
		if minutes > 0 {
			seconds = minutes * 60
		}
	}
	if seconds > 0 {
		cfg.Interval = time.Duration(seconds) * time.Second
	}
	if cfg.Interval < upstreamAutoSyncMinInterval {
		cfg.Interval = upstreamAutoSyncMinInterval
	}
	if cfg.Interval > upstreamAutoSyncMaxInterval {
		cfg.Interval = upstreamAutoSyncMaxInterval
	}
	return cfg
}

func parseMetadataBool(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "y", "on", "enabled":
			return true
		default:
			return false
		}
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func ProvideUpstreamAutoSyncRunner(svc *UpstreamService) *UpstreamAutoSyncRunner {
	runner := NewUpstreamAutoSyncRunner(svc)
	runner.Start()
	return runner
}
