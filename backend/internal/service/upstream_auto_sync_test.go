package service

import (
	"testing"
	"time"
)

func TestParseUpstreamAutoSyncConfigDisabledByDefault(t *testing.T) {
	cfg := parseUpstreamAutoSyncConfig(&Upstream{Metadata: map[string]any{}})
	if cfg.Enabled {
		t.Fatalf("Enabled = true, want false")
	}
	if cfg.Interval != upstreamAutoSyncDefaultInterval {
		t.Fatalf("Interval = %v, want %v", cfg.Interval, upstreamAutoSyncDefaultInterval)
	}
}

func TestParseUpstreamAutoSyncConfigReadsIntervalSeconds(t *testing.T) {
	cfg := parseUpstreamAutoSyncConfig(&Upstream{Metadata: map[string]any{
		"auto_sync_enabled":          "true",
		"auto_sync_interval_seconds": 900,
	}})
	if !cfg.Enabled {
		t.Fatalf("Enabled = false, want true")
	}
	if cfg.Interval != 15*time.Minute {
		t.Fatalf("Interval = %v, want 15m", cfg.Interval)
	}
}

func TestParseUpstreamAutoSyncConfigClampsTooSmallInterval(t *testing.T) {
	cfg := parseUpstreamAutoSyncConfig(&Upstream{Metadata: map[string]any{
		"auto_sync_enabled":          true,
		"auto_sync_interval_seconds": 10,
	}})
	if cfg.Interval != upstreamAutoSyncMinInterval {
		t.Fatalf("Interval = %v, want %v", cfg.Interval, upstreamAutoSyncMinInterval)
	}
}
