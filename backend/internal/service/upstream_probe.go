package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

const (
	upstreamProbeDefaultTimeout     = 30 * time.Second
	upstreamProbeMaxLineSize        = 8 * 1024 * 1024
	upstreamProbeStartupDelay       = 15 * time.Second
	upstreamProbeSchedulerTick      = 10 * time.Second
	upstreamProbeWorkerConcurrency  = 3
	upstreamProbeNoCandidateReason  = "probe_no_candidate"
	upstreamProbeSuccessReason      = "probe_success"
	upstreamProbeHTTPErrorReason    = "probe_http_error"
	upstreamProbeRequestErrorReason = "probe_request_error"
	upstreamProbeStreamErrorReason  = "probe_stream_error"
	upstreamProbeTimeoutReason      = "probe_timeout"
	upstreamProbeDisabledReason     = "probe_disabled"
	upstreamProbeUpstreamOffReason  = "probe_upstream_inactive"
	upstreamProbeNoAPIKeyReason     = "probe_no_api_key"
	upstreamProbeNoBaseURLReason    = "probe_no_base_url"
)

func (s *UpstreamService) RunProbe(ctx context.Context, upstreamID int64) (*UpstreamProbeResult, error) {
	return s.runProbe(ctx, upstreamID, true)
}

func (s *UpstreamService) runProbe(ctx context.Context, upstreamID int64, respectEnabled bool) (*UpstreamProbeResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUpstreamNotFound
	}
	checkedAt := time.Now().UTC()
	upstream, err := s.repo.Get(ctx, upstreamID)
	if err != nil {
		return nil, err
	}
	if respectEnabled && !upstream.ProbeEnabled {
		return &UpstreamProbeResult{
			UpstreamID:   upstreamID,
			Success:      false,
			Ignored:      true,
			Reason:       upstreamProbeDisabledReason,
			ErrorMessage: "upstream probe is disabled",
			CheckedAt:    checkedAt,
		}, nil
	}
	switch upstream.Status {
	case UpstreamStatusDisabled, UpstreamStatusCircuitOpen:
		return &UpstreamProbeResult{
			UpstreamID:   upstreamID,
			Success:      false,
			Ignored:      true,
			Reason:       upstreamProbeUpstreamOffReason,
			ErrorMessage: "upstream is not active for probing",
			CheckedAt:    checkedAt,
		}, nil
	}

	account, err := s.selectProbeAccount(ctx, upstreamID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return s.recordProbeResult(ctx, nil, &UpstreamProbeResult{
			UpstreamID:   upstreamID,
			Success:      false,
			Ignored:      true,
			Reason:       upstreamProbeNoCandidateReason,
			ErrorMessage: "no configured upstream API key candidate",
			CheckedAt:    checkedAt,
		})
	}

	result := newProbeResult(upstreamID, account, checkedAt)
	apiKey := strings.TrimSpace(account.GetOpenAIApiKey())
	if apiKey == "" {
		result.Ignored = true
		result.Reason = upstreamProbeNoAPIKeyReason
		result.ErrorMessage = "upstream API key secret missing"
		return s.recordProbeResult(ctx, account, result)
	}

	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	if baseURL == "" {
		baseURL = upstreamRuntimeOpenAIAPIBaseURL(upstream)
	}
	if baseURL == "" {
		result.Ignored = true
		result.Reason = upstreamProbeNoBaseURLReason
		result.ErrorMessage = "upstream OpenAI base URL missing"
		return s.recordProbeResult(ctx, account, result)
	}
	model := selectUpstreamProbeModel(upstream, account)
	result.Model = model

	timeout := time.Duration(upstream.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = upstreamProbeDefaultTimeout
	}
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	probeResult := executeUpstreamStreamingProbe(probeCtx, baseURL, apiKey, model)
	result.Success = probeResult.Success
	result.Reason = probeResult.Reason
	result.StatusCode = probeResult.StatusCode
	result.FirstTokenMS = probeResult.FirstTokenMS
	result.DurationMS = time.Since(start).Milliseconds()
	result.ErrorMessage = probeResult.ErrorMessage
	if result.DurationMS <= 0 {
		result.DurationMS = probeResult.DurationMS
	}
	if result.DurationMS <= 0 {
		result.DurationMS = 1
	}
	return s.recordProbeResult(ctx, account, result)
}

func (s *UpstreamService) selectProbeAccount(ctx context.Context, upstreamID int64) (*Account, error) {
	accounts, err := s.findRuntimeAccounts(ctx, upstreamID)
	if err != nil || len(accounts) == 0 {
		return nil, err
	}
	candidates := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform != PlatformOpenAI || account.Type != AccountTypeAPIKey {
			continue
		}
		if strings.TrimSpace(account.GetOpenAIApiKey()) == "" {
			continue
		}
		if strings.TrimSpace(account.GetCredential("base_url")) == "" {
			continue
		}
		candidates = append(candidates, account)
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		hi := parseAnyFloat(candidates[i].Extra["upstream_health_score"])
		hj := parseAnyFloat(candidates[j].Extra["upstream_health_score"])
		if hi != hj {
			return hi > hj
		}
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority < candidates[j].Priority
		}
		return candidates[i].ID < candidates[j].ID
	})
	account := candidates[0]
	return &account, nil
}

func newProbeResult(upstreamID int64, account *Account, checkedAt time.Time) *UpstreamProbeResult {
	result := &UpstreamProbeResult{
		UpstreamID: upstreamID,
		CheckedAt:  checkedAt,
	}
	if account == nil {
		return result
	}
	result.AccountID = account.ID
	result.RemoteAPIKeyID = strings.TrimSpace(anyToString(account.Extra["upstream_remote_api_key_id"]))
	result.RemoteAPIKeyName = strings.TrimSpace(anyToString(account.Extra["upstream_remote_api_key_name"]))
	result.RemoteGroupID = strings.TrimSpace(anyToString(account.Extra["upstream_remote_group_id"]))
	return result
}

func (s *UpstreamService) recordProbeResult(ctx context.Context, account *Account, result *UpstreamProbeResult) (*UpstreamProbeResult, error) {
	if result == nil {
		return nil, nil
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
	if strings.TrimSpace(result.Reason) == "" {
		if result.Success {
			result.Reason = upstreamProbeSuccessReason
		} else {
			result.Reason = upstreamProbeStreamErrorReason
		}
	}
	result.ErrorMessage = truncateString(sanitizeUpstreamErrorMessage(result.ErrorMessage), 2048)
	if s.repo == nil || result.UpstreamID <= 0 {
		return result, nil
	}
	event := UpstreamRuntimeEvent{
		UpstreamID:   result.UpstreamID,
		AccountID:    result.AccountID,
		Success:      result.Success,
		FirstTokenMs: result.FirstTokenMS,
		DurationMs:   result.DurationMS,
		StatusCode:   result.StatusCode,
		ErrorMessage: result.ErrorMessage,
		Ignored:      result.Ignored,
		Reason:       result.Reason,
		ObservedAt:   result.CheckedAt,
	}
	snapshot, err := s.repo.RecordRuntimeEvent(ctx, event)
	if err != nil {
		return result, err
	}
	result.SchedulerSnapshot = snapshot
	if account != nil && snapshot != nil {
		s.refreshRuntimeAccountFromSnapshot(ctx, account, snapshot)
	}
	return result, nil
}

func (s *UpstreamService) refreshRuntimeAccountFromSnapshot(ctx context.Context, account *Account, snapshot *UpstreamSchedulerSnapshot) {
	if s == nil || s.accountRepo == nil || account == nil || snapshot == nil {
		return
	}
	next := *account
	next.Credentials = copyAnyMap(account.Credentials)
	next.Extra = copyAnyMap(account.Extra)
	if next.Extra == nil {
		next.Extra = map[string]any{}
	}
	next.Extra["upstream_health_score"] = snapshot.HealthScore
	next.Extra["upstream_performance_score"] = snapshot.PerformanceScore
	next.Extra["upstream_capacity_score"] = snapshot.CapacityScore

	health := clampUpstreamScore01(defaultScore(snapshot.HealthScore))
	if health < 0.2 {
		next.Status = StatusError
		next.Schedulable = false
		next.ErrorMessage = "upstream health too low"
	} else if next.Status == StatusError && strings.Contains(strings.ToLower(next.ErrorMessage), "upstream health too low") {
		next.Status = StatusActive
		next.Schedulable = true
		next.ErrorMessage = ""
	}

	basePriority := parseAnyInt64(next.Extra["upstream_base_priority"])
	if basePriority <= 0 {
		basePriority = int64(next.Priority)
	}
	if basePriority <= 0 {
		basePriority = 100
	}
	priority := int(basePriority)
	if health < 0.9 {
		priority += int((0.9 - health) * 100)
	}
	next.Priority = clampInt(priority, 1, 10000)

	baseLoad := parseAnyInt64(next.Extra["upstream_base_load_factor"])
	if baseLoad <= 0 {
		baseLoad = int64(next.EffectiveLoadFactor())
	}
	if baseLoad <= 0 {
		baseLoad = 1
	}
	loadFactor := clampInt(int(float64(baseLoad)*health), 1, int(baseLoad))
	next.LoadFactor = &loadFactor

	if err := s.accountRepo.Update(ctx, &next); err != nil {
		slog.Warn("upstream_probe: refresh runtime account failed", "upstream_id", resultAccountUpstreamID(account), "account_id", account.ID, "error", err)
	}
}

func executeUpstreamStreamingProbe(ctx context.Context, baseURL, apiKey, model string) *UpstreamProbeResult {
	start := time.Now()
	result := &UpstreamProbeResult{
		Model:     model,
		Reason:    upstreamProbeStreamErrorReason,
		CheckedAt: start.UTC(),
	}
	targetURL := buildOpenAIChatCompletionsURL(baseURL)
	payload := map[string]any{
		"model":       model,
		"stream":      true,
		"max_tokens":  8,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		result.Reason = upstreamProbeRequestErrorReason
		result.ErrorMessage = err.Error()
		result.DurationMS = time.Since(start).Milliseconds()
		return result
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		result.Reason = upstreamProbeRequestErrorReason
		result.ErrorMessage = err.Error()
		result.DurationMS = time.Since(start).Milliseconds()
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Reason = probeRequestErrorReason(err)
		result.ErrorMessage = err.Error()
		result.DurationMS = time.Since(start).Milliseconds()
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 400 {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		if readErr != nil {
			result.ErrorMessage = fmt.Sprintf("http %d; read error body failed: %v", resp.StatusCode, readErr)
		} else {
			message := strings.TrimSpace(extractUpstreamErrorMessage(bodyBytes))
			if message == "" {
				message = string(bodyBytes)
			}
			result.ErrorMessage = fmt.Sprintf("http %d: %s", resp.StatusCode, strings.TrimSpace(message))
		}
		result.Reason = upstreamProbeHTTPErrorReason
		result.DurationMS = time.Since(start).Milliseconds()
		return result
	}

	streamResult := readProbeStream(ctx, resp.Body, start)
	result.Success = streamResult.Success
	result.Reason = streamResult.Reason
	result.FirstTokenMS = streamResult.FirstTokenMS
	result.ErrorMessage = streamResult.ErrorMessage
	result.DurationMS = time.Since(start).Milliseconds()
	if result.DurationMS <= 0 {
		result.DurationMS = 1
	}
	return result
}

type probeStreamResult struct {
	Success      bool
	Reason       string
	FirstTokenMS *int
	ErrorMessage string
}

func readProbeStream(ctx context.Context, body io.Reader, start time.Time) probeStreamResult {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), upstreamProbeMaxLineSize)
	seenData := false
	seenFinish := false
	var firstTokenMS *int

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return probeStreamResult{Reason: probeRequestErrorReason(err), FirstTokenMS: firstTokenMS, ErrorMessage: err.Error()}
		}
		line := scanner.Text()
		payload, ok := extractOpenAISSEDataLine(line)
		if !ok {
			continue
		}
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue
		}
		seenData = true
		if payload == "[DONE]" {
			seenFinish = true
			break
		}
		if message := probePayloadErrorMessage(payload); message != "" {
			return probeStreamResult{
				Reason:       upstreamProbeStreamErrorReason,
				FirstTokenMS: firstTokenMS,
				ErrorMessage: message,
			}
		}
		if firstTokenMS == nil && probePayloadHasToken(payload) {
			ms := int(time.Since(start).Milliseconds())
			if ms <= 0 {
				ms = 1
			}
			firstTokenMS = &ms
		}
		if probePayloadHasFinish(payload) {
			seenFinish = true
		}
	}
	if err := scanner.Err(); err != nil {
		return probeStreamResult{
			Reason:       probeRequestErrorReason(err),
			FirstTokenMS: firstTokenMS,
			ErrorMessage: err.Error(),
		}
	}
	if firstTokenMS != nil || seenFinish {
		return probeStreamResult{
			Success:      true,
			Reason:       upstreamProbeSuccessReason,
			FirstTokenMS: firstTokenMS,
		}
	}
	if seenData {
		return probeStreamResult{
			Reason:       upstreamProbeStreamErrorReason,
			FirstTokenMS: firstTokenMS,
			ErrorMessage: "stream ended before first token or finish signal",
		}
	}
	return probeStreamResult{
		Reason:       upstreamProbeStreamErrorReason,
		FirstTokenMS: firstTokenMS,
		ErrorMessage: "empty stream response",
	}
}

func probePayloadHasToken(payload string) bool {
	if strings.TrimSpace(payload) == "" || !gjson.Valid(payload) {
		return false
	}
	paths := []string{
		"choices.0.delta.content",
		"choices.0.delta.reasoning_content",
		"choices.0.delta.text",
		"choices.0.text",
		"delta",
		"content",
		"response.output_text.delta",
	}
	for _, path := range paths {
		value := gjson.Get(payload, path)
		if value.Exists() && strings.TrimSpace(value.String()) != "" {
			return true
		}
	}
	return false
}

func probePayloadHasFinish(payload string) bool {
	if strings.TrimSpace(payload) == "" || !gjson.Valid(payload) {
		return false
	}
	if finish := strings.TrimSpace(gjson.Get(payload, "choices.0.finish_reason").String()); finish != "" && finish != "null" {
		return true
	}
	switch strings.TrimSpace(gjson.Get(payload, "type").String()) {
	case "response.completed", "response.done", "done", "message_stop":
		return true
	default:
		return false
	}
}

func probePayloadErrorMessage(payload string) string {
	if strings.TrimSpace(payload) == "" || !gjson.Valid(payload) {
		return ""
	}
	for _, path := range []string{"error.message", "error", "message", "response.error.message"} {
		value := gjson.Get(payload, path)
		if !value.Exists() {
			continue
		}
		msg := strings.TrimSpace(value.String())
		if msg != "" {
			return msg
		}
	}
	return ""
}

func probeRequestErrorReason(err error) string {
	if err == nil {
		return upstreamProbeRequestErrorReason
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return upstreamProbeTimeoutReason
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return upstreamProbeTimeoutReason
	}
	return upstreamProbeRequestErrorReason
}

func selectUpstreamProbeModel(upstream *Upstream, account *Account) string {
	if upstream != nil {
		if model := strings.TrimSpace(upstream.ProbeModel); model != "" {
			return model
		}
	}
	if account != nil {
		model := selectResponsesProbeModel(account)
		if strings.TrimSpace(model) != "" {
			return model
		}
	}
	return openai.DefaultTestModel
}

func resultAccountUpstreamID(account *Account) int64 {
	if account == nil {
		return 0
	}
	return accountUpstreamID(account)
}

func parseAnyFloat(raw any) float64 {
	switch v := raw.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		var f float64
		_, _ = fmt.Sscanf(strings.TrimSpace(v), "%f", &f)
		return f
	default:
		return 0
	}
}

type UpstreamProbeRunner struct {
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

func NewUpstreamProbeRunner(svc *UpstreamService) *UpstreamProbeRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &UpstreamProbeRunner{
		svc:      svc,
		ctx:      ctx,
		cancel:   cancel,
		nextDue:  map[int64]time.Time{},
		inFlight: map[int64]struct{}{},
		sem:      make(chan struct{}, upstreamProbeWorkerConcurrency),
	}
}

func (r *UpstreamProbeRunner) Start() {
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

func (r *UpstreamProbeRunner) Stop() {
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

func (r *UpstreamProbeRunner) loop() {
	defer r.wg.Done()
	timer := time.NewTimer(upstreamProbeStartupDelay)
	defer timer.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-timer.C:
			r.tick()
			timer.Reset(upstreamProbeSchedulerTick)
		}
	}
}

func (r *UpstreamProbeRunner) tick() {
	ctx, cancel := context.WithTimeout(r.ctx, 15*time.Second)
	defer cancel()
	items, _, err := r.svc.repo.List(ctx, UpstreamListParams{Page: 1, PageSize: 100})
	if err != nil {
		slog.Warn("upstream_probe: list upstreams failed", "error", err)
		return
	}
	now := time.Now()
	for _, upstream := range items {
		if upstream == nil || !upstream.ProbeEnabled {
			continue
		}
		switch upstream.Status {
		case UpstreamStatusDisabled, UpstreamStatusCircuitOpen:
			continue
		}
		if !r.isDue(upstream.ID, now, time.Duration(upstream.ProbeIntervalSeconds)*time.Second) {
			continue
		}
		r.submit(upstream.ID)
	}
}

func (r *UpstreamProbeRunner) isDue(upstreamID int64, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		interval = time.Minute
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.inFlight[upstreamID]; ok {
		return false
	}
	next := r.nextDue[upstreamID]
	if !next.IsZero() && now.Before(next) {
		return false
	}
	r.inFlight[upstreamID] = struct{}{}
	r.nextDue[upstreamID] = now.Add(interval)
	return true
}

func (r *UpstreamProbeRunner) submit(upstreamID int64) {
	select {
	case r.sem <- struct{}{}:
	case <-r.ctx.Done():
		r.release(upstreamID)
		return
	default:
		r.release(upstreamID)
		slog.Warn("upstream_probe: worker pool full, skip", "upstream_id", upstreamID)
		return
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			<-r.sem
			r.release(upstreamID)
			if rec := recover(); rec != nil {
				slog.Error("upstream_probe: panic", "upstream_id", upstreamID, "panic", rec)
			}
		}()
		ctx, cancel := context.WithTimeout(r.ctx, upstreamProbeDefaultTimeout+5*time.Second)
		defer cancel()
		if _, err := r.svc.runProbe(ctx, upstreamID, true); err != nil {
			slog.Warn("upstream_probe: run failed", "upstream_id", upstreamID, "error", err)
		}
	}()
}

func (r *UpstreamProbeRunner) release(upstreamID int64) {
	r.mu.Lock()
	delete(r.inFlight, upstreamID)
	r.mu.Unlock()
}

func ProvideUpstreamProbeRunner(svc *UpstreamService) *UpstreamProbeRunner {
	runner := NewUpstreamProbeRunner(svc)
	runner.Start()
	return runner
}
