package service

import (
	"math"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type UpstreamScheduleRequest struct {
	Model         string
	RemoteGroupID string
	LocalGroupID  int64
	Mode          string
	RandomSeed    int64
	Candidates    []UpstreamCandidate
}

type UpstreamCandidate struct {
	ID                int64
	Name              string
	Status            string
	Priority          int
	Weight            int
	CostMultiplier    float64
	SupportedModels   []string
	RemoteGroupIDs    []string
	CapacityAvailable bool
	HealthScore       float64
	PerformanceScore  float64
	CapacityScore     float64
}

type UpstreamScoreWeights struct {
	Health      float64 `json:"health"`
	Performance float64 `json:"performance"`
	Cost        float64 `json:"cost"`
	Capacity    float64 `json:"capacity"`
	Priority    float64 `json:"priority"`
}

type UpstreamScoreBreakdown struct {
	UpstreamID       int64                             `json:"upstream_id"`
	Name             string                            `json:"name"`
	Score            float64                           `json:"score"`
	WeightedTicket   float64                           `json:"weighted_ticket"`
	HealthScore      float64                           `json:"health_score"`
	PerformanceScore float64                           `json:"performance_score"`
	CostScore        float64                           `json:"cost_score"`
	CapacityScore    float64                           `json:"capacity_score"`
	PriorityScore    float64                           `json:"priority_score"`
	Weights          UpstreamScoreWeights              `json:"weights"`
	Filtered         bool                              `json:"filtered"`
	FilterReason     string                            `json:"filter_reason,omitempty"`
	CandidateAPIKeys []UpstreamAPIKeyScheduleCandidate `json:"candidate_api_keys,omitempty"`
}

type UpstreamScheduleDecision struct {
	SelectedID             int64                             `json:"selected_id"`
	SelectedName           string                            `json:"selected_name"`
	SelectedRemoteAPIKeyID string                            `json:"selected_remote_api_key_id,omitempty"`
	SelectedRemoteGroupID  string                            `json:"selected_remote_group_id,omitempty"`
	LocalGroupID           int64                             `json:"local_group_id,omitempty"`
	Mode                   string                            `json:"mode"`
	Reason                 string                            `json:"reason"`
	CandidateScores        []UpstreamScoreBreakdown          `json:"candidate_scores"`
	Filtered               []UpstreamScoreBreakdown          `json:"filtered"`
	CandidateAPIKeys       []UpstreamAPIKeyScheduleCandidate `json:"candidate_api_keys,omitempty"`
	FilteredAPIKeys        []UpstreamAPIKeyScheduleCandidate `json:"filtered_api_keys,omitempty"`
}

type UpstreamAPIKeyScheduleCandidate struct {
	UpstreamID       int64   `json:"upstream_id"`
	UpstreamName     string  `json:"upstream_name"`
	RemoteAPIKeyID   string  `json:"remote_api_key_id"`
	RemoteAPIKeyName string  `json:"remote_api_key_name"`
	RemoteGroupID    string  `json:"remote_group_id"`
	LocalGroupIDs    []int64 `json:"local_group_ids,omitempty"`
	Status           string  `json:"status"`
	Schedulable      bool    `json:"schedulable"`
	FilterReason     string  `json:"filter_reason,omitempty"`
}

func SelectUpstreamCandidate(req UpstreamScheduleRequest) UpstreamScheduleDecision {
	mode := normalizeUpstreamRoutingMode(req.Mode)
	weights := weightsForUpstreamMode(mode)
	eligible := make([]UpstreamScoreBreakdown, 0, len(req.Candidates))
	filtered := make([]UpstreamScoreBreakdown, 0)

	for _, candidate := range req.Candidates {
		if reason := filterUpstreamCandidate(req, candidate); reason != "" {
			filtered = append(filtered, UpstreamScoreBreakdown{
				UpstreamID:   candidate.ID,
				Name:         candidate.Name,
				Filtered:     true,
				FilterReason: reason,
			})
			continue
		}
		score := scoreUpstreamCandidate(candidate, weights)
		eligible = append(eligible, score)
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Score == eligible[j].Score {
			return eligible[i].UpstreamID < eligible[j].UpstreamID
		}
		return eligible[i].Score > eligible[j].Score
	})

	decision := UpstreamScheduleDecision{
		Mode:            mode,
		LocalGroupID:    req.LocalGroupID,
		CandidateScores: eligible,
		Filtered:        filtered,
	}
	if len(eligible) == 0 {
		decision.Reason = "no eligible upstream candidates"
		return decision
	}

	selected := selectUpstreamForMode(mode, eligible, req.RandomSeed)
	decision.SelectedID = selected.UpstreamID
	decision.SelectedName = selected.Name
	if mode == UpstreamRoutingBalanced {
		decision.Reason = "selected by weighted score among eligible upstreams"
	} else {
		decision.Reason = "selected top-scored upstream for " + mode + " mode"
	}
	return decision
}

func filterUpstreamCandidate(req UpstreamScheduleRequest, candidate UpstreamCandidate) string {
	status := strings.TrimSpace(candidate.Status)
	if status == "" {
		status = UpstreamStatusActive
	}
	switch status {
	case UpstreamStatusDisabled:
		return "upstream disabled"
	case UpstreamStatusCircuitOpen:
		return "circuit breaker open"
	}
	if candidate.Weight <= 0 {
		return "weight is zero"
	}
	if !candidate.CapacityAvailable {
		return "capacity unavailable"
	}
	if model := strings.TrimSpace(req.Model); model != "" && len(candidate.SupportedModels) > 0 && !matchAnyModelPattern(model, candidate.SupportedModels) {
		return "model not supported"
	}
	if groupID := strings.TrimSpace(req.RemoteGroupID); groupID != "" && len(candidate.RemoteGroupIDs) > 0 && !containsString(candidate.RemoteGroupIDs, groupID) {
		return "remote group unavailable"
	}
	return ""
}

func scoreUpstreamCandidate(candidate UpstreamCandidate, weights UpstreamScoreWeights) UpstreamScoreBreakdown {
	health := clampUpstreamScore01(defaultScore(candidate.HealthScore))
	performance := clampUpstreamScore01(defaultScore(candidate.PerformanceScore))
	capacity := clampUpstreamScore01(defaultScore(candidate.CapacityScore))
	cost := costScore(candidate.CostMultiplier)
	priority := priorityScore(candidate.Priority)
	score := health*weights.Health +
		performance*weights.Performance +
		cost*weights.Cost +
		capacity*weights.Capacity +
		priority*weights.Priority
	if score <= 0 {
		score = 0.0001
	}
	return UpstreamScoreBreakdown{
		UpstreamID:       candidate.ID,
		Name:             candidate.Name,
		Score:            roundScore(score),
		WeightedTicket:   roundScore(score * float64(candidate.Weight)),
		HealthScore:      roundScore(health),
		PerformanceScore: roundScore(performance),
		CostScore:        roundScore(cost),
		CapacityScore:    roundScore(capacity),
		PriorityScore:    roundScore(priority),
		Weights:          weights,
	}
}

func weightedRandomUpstream(candidates []UpstreamScoreBreakdown, seed int64) UpstreamScoreBreakdown {
	total := 0.0
	for _, candidate := range candidates {
		total += candidate.WeightedTicket
	}
	if total <= 0 {
		return candidates[0]
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(seed))
	pick := r.Float64() * total
	accumulated := 0.0
	for _, candidate := range candidates {
		accumulated += candidate.WeightedTicket
		if pick <= accumulated {
			return candidate
		}
	}
	return candidates[len(candidates)-1]
}

func selectUpstreamForMode(mode string, candidates []UpstreamScoreBreakdown, seed int64) UpstreamScoreBreakdown {
	if len(candidates) == 0 {
		return UpstreamScoreBreakdown{}
	}
	if normalizeUpstreamRoutingMode(mode) == UpstreamRoutingBalanced {
		return weightedRandomUpstream(candidates, seed)
	}
	return candidates[0]
}

func weightsForUpstreamMode(mode string) UpstreamScoreWeights {
	switch normalizeUpstreamRoutingMode(mode) {
	case UpstreamRoutingStability:
		return UpstreamScoreWeights{Health: 0.5, Performance: 0.25, Cost: 0.1, Capacity: 0.1, Priority: 0.05}
	case UpstreamRoutingCost:
		return UpstreamScoreWeights{Health: 0.25, Performance: 0.1, Cost: 0.45, Capacity: 0.1, Priority: 0.1}
	case UpstreamRoutingSpeed:
		return UpstreamScoreWeights{Health: 0.25, Performance: 0.45, Cost: 0.1, Capacity: 0.1, Priority: 0.1}
	case UpstreamRoutingManual:
		return UpstreamScoreWeights{Health: 0.15, Performance: 0.1, Cost: 0.05, Capacity: 0.1, Priority: 0.6}
	default:
		return UpstreamScoreWeights{Health: 0.35, Performance: 0.25, Cost: 0.2, Capacity: 0.15, Priority: 0.05}
	}
}

func normalizeUpstreamRoutingMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case UpstreamRoutingStability, UpstreamRoutingCost, UpstreamRoutingSpeed, UpstreamRoutingManual:
		return strings.TrimSpace(mode)
	default:
		return UpstreamRoutingBalanced
	}
}

func defaultScore(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

func costScore(multiplier float64) float64 {
	if multiplier <= 0 {
		return 1
	}
	return clampUpstreamScore01(1 / multiplier)
}

func priorityScore(priority int) float64 {
	if priority <= 0 {
		return 1
	}
	return clampUpstreamScore01(1 / (1 + float64(priority)/100))
}

func clampUpstreamScore01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func roundScore(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func matchAnyModelPattern(model string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == "*" || strings.EqualFold(pattern, model) {
			return true
		}
		if ok, _ := filepath.Match(pattern, model); ok {
			return true
		}
	}
	return false
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}
