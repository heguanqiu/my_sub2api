package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReadProbeStreamSuccessWithFirstToken(t *testing.T) {
	start := time.Now().Add(-25 * time.Millisecond)
	stream := strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")

	got := readProbeStream(context.Background(), stream, start)

	if !got.Success {
		t.Fatalf("success = false, error = %q", got.ErrorMessage)
	}
	if got.Reason != upstreamProbeSuccessReason {
		t.Fatalf("reason = %q, want %q", got.Reason, upstreamProbeSuccessReason)
	}
	if got.FirstTokenMS == nil || *got.FirstTokenMS <= 0 {
		t.Fatalf("first token = %v, want positive", got.FirstTokenMS)
	}
}

func TestReadProbeStreamReturnsUpstreamError(t *testing.T) {
	stream := strings.NewReader("data: {\"error\":{\"message\":\"bad upstream\"}}\n\n")

	got := readProbeStream(context.Background(), stream, time.Now())

	if got.Success {
		t.Fatal("success = true, want false")
	}
	if got.Reason != upstreamProbeStreamErrorReason {
		t.Fatalf("reason = %q, want %q", got.Reason, upstreamProbeStreamErrorReason)
	}
	if got.ErrorMessage != "bad upstream" {
		t.Fatalf("error = %q, want bad upstream", got.ErrorMessage)
	}
}

func TestReadProbeStreamEmptyFails(t *testing.T) {
	got := readProbeStream(context.Background(), strings.NewReader(""), time.Now())

	if got.Success {
		t.Fatal("success = true, want false")
	}
	if got.Reason != upstreamProbeStreamErrorReason {
		t.Fatalf("reason = %q, want %q", got.Reason, upstreamProbeStreamErrorReason)
	}
	if !strings.Contains(got.ErrorMessage, "empty stream") {
		t.Fatalf("error = %q, want empty stream", got.ErrorMessage)
	}
}

func TestReadProbeStreamReadTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := readProbeStream(ctx, strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"), time.Now())

	if got.Success {
		t.Fatal("success = true, want false")
	}
	if got.Reason != upstreamProbeTimeoutReason {
		t.Fatalf("reason = %q, want %q", got.Reason, upstreamProbeTimeoutReason)
	}
}

func TestProbePayloadHelpers(t *testing.T) {
	if !probePayloadHasToken(`{"choices":[{"delta":{"reasoning_content":"thinking"}}]}`) {
		t.Fatal("reasoning_content should count as first token")
	}
	if !probePayloadHasFinish(`{"choices":[{"finish_reason":"stop"}]}`) {
		t.Fatal("finish_reason should count as finish signal")
	}
	if got := probePayloadErrorMessage(`{"response":{"error":{"message":"stream failed"}}}`); got != "stream failed" {
		t.Fatalf("error message = %q, want stream failed", got)
	}
}

func TestRunProbeNoConfiguredCandidateIsIgnored(t *testing.T) {
	repo := &upstreamProbeRepo{
		upstream: &Upstream{
			ID:                   7,
			Name:                 "up",
			Type:                 UpstreamTypeSub2API,
			BaseURL:              "https://up.example.com",
			Status:               UpstreamStatusActive,
			ProbeEnabled:         true,
			ProbeIntervalSeconds: 60,
		},
	}
	accountRepo := &upstreamProbeAccountRepo{
		accounts: []Account{{
			ID:          11,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusError,
			Schedulable: false,
			Credentials: map[string]any{"base_url": "https://up.example.com/v1"},
			Extra:       map[string]any{upstreamRuntimeManagedExtraKey: true, "upstream_id": int64(7)},
		}},
	}
	svc := NewUpstreamService(repo, accountRepo, nil, nil, nil)

	result, err := svc.RunProbe(context.Background(), 7)
	if err != nil {
		t.Fatalf("RunProbe() error = %v", err)
	}
	if result == nil || !result.Ignored {
		t.Fatalf("result ignored = %v, want true", result)
	}
	if result.Reason != upstreamProbeNoCandidateReason {
		t.Fatalf("reason = %q, want %q", result.Reason, upstreamProbeNoCandidateReason)
	}
	if repo.recorded == nil || !repo.recorded.Ignored {
		t.Fatalf("RecordRuntimeEvent ignored event = %+v, want ignored", repo.recorded)
	}
}

type upstreamProbeRepo struct {
	UpstreamRepository
	upstream *Upstream
	recorded *UpstreamRuntimeEvent
}

func (r *upstreamProbeRepo) Get(ctx context.Context, id int64) (*Upstream, error) {
	if r.upstream == nil || r.upstream.ID != id {
		return nil, ErrUpstreamNotFound
	}
	copy := *r.upstream
	return &copy, nil
}

func (r *upstreamProbeRepo) RecordRuntimeEvent(ctx context.Context, event UpstreamRuntimeEvent) (*UpstreamSchedulerSnapshot, error) {
	copy := event
	r.recorded = &copy
	if event.Ignored {
		return nil, nil
	}
	return &UpstreamSchedulerSnapshot{HealthScore: 1, PerformanceScore: 1, CapacityScore: 1}, nil
}

type upstreamProbeAccountRepo struct {
	AccountRepository
	accounts []Account
}

func (r *upstreamProbeAccountRepo) FindByExtraField(ctx context.Context, key string, value any) ([]Account, error) {
	out := make([]Account, len(r.accounts))
	copy(out, r.accounts)
	return out, nil
}

func (r *upstreamProbeAccountRepo) Update(ctx context.Context, account *Account) error {
	return nil
}
