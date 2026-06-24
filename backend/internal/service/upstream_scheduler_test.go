package service

import "testing"

func TestSelectUpstreamCandidateFiltersIneligibleCandidates(t *testing.T) {
	decision := SelectUpstreamCandidate(UpstreamScheduleRequest{
		Model:         "gpt-4o",
		RemoteGroupID: "vip",
		Mode:          UpstreamRoutingBalanced,
		RandomSeed:    7,
		Candidates: []UpstreamCandidate{
			{
				ID:                1,
				Name:              "disabled",
				Status:            UpstreamStatusDisabled,
				Weight:            100,
				CapacityAvailable: true,
			},
			{
				ID:                2,
				Name:              "open circuit",
				Status:            UpstreamStatusCircuitOpen,
				Weight:            100,
				CapacityAvailable: true,
			},
			{
				ID:                3,
				Name:              "zero weight",
				Status:            UpstreamStatusActive,
				Weight:            0,
				CapacityAvailable: true,
			},
			{
				ID:                4,
				Name:              "no capacity",
				Status:            UpstreamStatusActive,
				Weight:            100,
				CapacityAvailable: false,
			},
			{
				ID:                5,
				Name:              "wrong model",
				Status:            UpstreamStatusActive,
				Weight:            100,
				SupportedModels:   []string{"claude-*"},
				CapacityAvailable: true,
			},
			{
				ID:                6,
				Name:              "wrong group",
				Status:            UpstreamStatusActive,
				Weight:            100,
				SupportedModels:   []string{"gpt-*"},
				RemoteGroupIDs:    []string{"standard"},
				CapacityAvailable: true,
			},
			{
				ID:                7,
				Name:              "eligible",
				Status:            UpstreamStatusActive,
				Weight:            100,
				SupportedModels:   []string{"gpt-*"},
				RemoteGroupIDs:    []string{"vip"},
				CapacityAvailable: true,
				HealthScore:       0.9,
				PerformanceScore:  0.8,
				CapacityScore:     0.7,
			},
		},
	})

	if decision.SelectedID != 7 {
		t.Fatalf("selected id = %d, want 7", decision.SelectedID)
	}
	if got, want := len(decision.CandidateScores), 1; got != want {
		t.Fatalf("eligible candidates = %d, want %d", got, want)
	}
	if got, want := len(decision.Filtered), 6; got != want {
		t.Fatalf("filtered candidates = %d, want %d", got, want)
	}

	reasons := map[int64]string{}
	for _, item := range decision.Filtered {
		reasons[item.UpstreamID] = item.FilterReason
	}
	expected := map[int64]string{
		1: "upstream disabled",
		2: "circuit breaker open",
		3: "weight is zero",
		4: "capacity unavailable",
		5: "model not supported",
		6: "remote group unavailable",
	}
	for id, reason := range expected {
		if reasons[id] != reason {
			t.Fatalf("candidate %d filter reason = %q, want %q", id, reasons[id], reason)
		}
	}
}

func TestSelectUpstreamCandidateCostModeRewardsLowerMultiplier(t *testing.T) {
	decision := SelectUpstreamCandidate(UpstreamScheduleRequest{
		Mode:       UpstreamRoutingCost,
		RandomSeed: 11,
		Candidates: []UpstreamCandidate{
			{
				ID:                1,
				Name:              "expensive",
				Status:            UpstreamStatusActive,
				Weight:            100,
				CostMultiplier:    2,
				CapacityAvailable: true,
				HealthScore:       1,
				PerformanceScore:  1,
				CapacityScore:     1,
			},
			{
				ID:                2,
				Name:              "cheap",
				Status:            UpstreamStatusActive,
				Weight:            100,
				CostMultiplier:    0.5,
				CapacityAvailable: true,
				HealthScore:       1,
				PerformanceScore:  1,
				CapacityScore:     1,
			},
		},
	})

	if got, want := decision.Mode, UpstreamRoutingCost; got != want {
		t.Fatalf("mode = %q, want %q", got, want)
	}
	if got, want := len(decision.CandidateScores), 2; got != want {
		t.Fatalf("candidate scores = %d, want %d", got, want)
	}
	if decision.CandidateScores[0].UpstreamID != 2 {
		t.Fatalf("top candidate = %d, want cheap upstream 2", decision.CandidateScores[0].UpstreamID)
	}
	if decision.CandidateScores[0].CostScore <= decision.CandidateScores[1].CostScore {
		t.Fatalf("cheap cost score = %v, expensive cost score = %v; want cheap higher",
			decision.CandidateScores[0].CostScore,
			decision.CandidateScores[1].CostScore,
		)
	}
	if decision.CandidateScores[0].Weights.Cost <= decision.CandidateScores[0].Weights.Health {
		t.Fatalf("cost mode weights = %+v, want cost to dominate health", decision.CandidateScores[0].Weights)
	}
}

func TestSelectUpstreamCandidateManualModeMakesPriorityDominate(t *testing.T) {
	decision := SelectUpstreamCandidate(UpstreamScheduleRequest{
		Mode:       UpstreamRoutingManual,
		RandomSeed: 19,
		Candidates: []UpstreamCandidate{
			{
				ID:                1,
				Name:              "high priority number",
				Status:            UpstreamStatusActive,
				Priority:          100,
				Weight:            100,
				CostMultiplier:    1,
				CapacityAvailable: true,
				HealthScore:       1,
				PerformanceScore:  1,
				CapacityScore:     1,
			},
			{
				ID:                2,
				Name:              "low priority number",
				Status:            UpstreamStatusActive,
				Priority:          1,
				Weight:            100,
				CostMultiplier:    1,
				CapacityAvailable: true,
				HealthScore:       0.75,
				PerformanceScore:  0.75,
				CapacityScore:     0.75,
			},
		},
	})

	if decision.CandidateScores[0].UpstreamID != 2 {
		t.Fatalf("top candidate = %d, want lower priority-number upstream 2", decision.CandidateScores[0].UpstreamID)
	}
	if decision.CandidateScores[0].Weights.Priority != 0.6 {
		t.Fatalf("manual weights = %+v, want priority weight 0.6", decision.CandidateScores[0].Weights)
	}
}

func TestSelectUpstreamCandidateWeightedRandomIsDeterministicAndExplainable(t *testing.T) {
	req := UpstreamScheduleRequest{
		Mode:       UpstreamRoutingBalanced,
		RandomSeed: 42,
		Candidates: []UpstreamCandidate{
			{
				ID:                1,
				Name:              "a",
				Status:            UpstreamStatusActive,
				Priority:          100,
				Weight:            10,
				CostMultiplier:    1,
				CapacityAvailable: true,
				HealthScore:       0.9,
				PerformanceScore:  0.7,
				CapacityScore:     0.8,
			},
			{
				ID:                2,
				Name:              "b",
				Status:            UpstreamStatusActive,
				Priority:          100,
				Weight:            90,
				CostMultiplier:    1,
				CapacityAvailable: true,
				HealthScore:       0.9,
				PerformanceScore:  0.7,
				CapacityScore:     0.8,
			},
		},
	}

	first := SelectUpstreamCandidate(req)
	second := SelectUpstreamCandidate(req)

	if first.SelectedID != second.SelectedID {
		t.Fatalf("same seed selected ids %d and %d, want deterministic result", first.SelectedID, second.SelectedID)
	}
	if first.Reason == "" {
		t.Fatal("reason is empty")
	}
	if got, want := len(first.CandidateScores), 2; got != want {
		t.Fatalf("candidate scores = %d, want %d", got, want)
	}
	for _, score := range first.CandidateScores {
		if score.Score <= 0 {
			t.Fatalf("candidate %d score = %v, want positive", score.UpstreamID, score.Score)
		}
		if score.WeightedTicket <= 0 {
			t.Fatalf("candidate %d weighted ticket = %v, want positive", score.UpstreamID, score.WeightedTicket)
		}
		if score.Weights == (UpstreamScoreWeights{}) {
			t.Fatalf("candidate %d weights are empty", score.UpstreamID)
		}
	}
}

func TestScheduleAPIKeyCandidatesDoesNotRequireRemoteGroup(t *testing.T) {
	upstream := &Upstream{ID: 9, Name: "north"}
	keys := []*UpstreamRemoteAPIKey{
		{
			RemoteAPIKeyID:   "key-1",
			RemoteAPIKeyName: "pro",
			APIKeyConfigured: true,
			Status:           "active",
			LocalGroupIDs:    []int64{36},
		},
	}

	eligible, filtered := scheduleAPIKeyCandidates(upstream, nil, keys, 36, "")

	if got, want := len(eligible), 1; got != want {
		t.Fatalf("eligible keys = %d, want %d; filtered=%+v", got, want, filtered)
	}
	if len(filtered) != 0 {
		t.Fatalf("filtered keys = %+v, want none", filtered)
	}
	if eligible[0].RemoteAPIKeyID != "key-1" {
		t.Fatalf("eligible key id = %q, want key-1", eligible[0].RemoteAPIKeyID)
	}
}
