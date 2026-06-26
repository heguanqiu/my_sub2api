package service

import "testing"

func TestBuildUpstreamSyncDiffCapturesGroupsKeysAndCostChanges(t *testing.T) {
	diff := BuildUpstreamSyncDiff(
		[]*UpstreamRemoteGroup{
			{RemoteGroupID: "default", RemoteGroupName: "Default", RateMultiplier: 1, Status: UpstreamStatusActive},
			{RemoteGroupID: "legacy", RemoteGroupName: "Legacy", RateMultiplier: 2, Status: UpstreamStatusActive},
		},
		[]*UpstreamRemoteAPIKey{
			{RemoteAPIKeyID: "key-1", RemoteAPIKeyName: "Key 1", SyncedRemoteGroupID: "default", RemoteGroupID: "default", LocalGroupIDs: []int64{10}},
			{RemoteAPIKeyID: "key-old", RemoteAPIKeyName: "Old", SyncedRemoteGroupID: "legacy", RemoteGroupID: "legacy", LocalGroupIDs: []int64{11}},
		},
		[]*UpstreamRemoteGroup{
			{RemoteGroupID: "default", RemoteGroupName: "Default", RateMultiplier: 1.5, Status: UpstreamStatusActive},
			{RemoteGroupID: "vip", RemoteGroupName: "VIP", RateMultiplier: 3, Status: UpstreamStatusActive},
		},
		[]*UpstreamRemoteAPIKey{
			{RemoteAPIKeyID: "key-1", RemoteAPIKeyName: "Key 1", SyncedRemoteGroupID: "vip", RemoteGroupID: "vip", LocalGroupIDs: []int64{10}},
			{RemoteAPIKeyID: "key-new", RemoteAPIKeyName: "New", SyncedRemoteGroupID: "vip", RemoteGroupID: "vip", LocalGroupIDs: []int64{12}},
		},
	)

	if got := len(diff.AddedGroups); got != 1 {
		t.Fatalf("AddedGroups len = %d, want 1", got)
	}
	if got := len(diff.RemovedGroups); got != 1 {
		t.Fatalf("RemovedGroups len = %d, want 1", got)
	}
	if got := len(diff.ChangedGroups); got != 1 {
		t.Fatalf("ChangedGroups len = %d, want 1", got)
	}
	if diff.CostMultiplierChangeCount != 1 {
		t.Fatalf("CostMultiplierChangeCount = %d, want 1", diff.CostMultiplierChangeCount)
	}
	if got := len(diff.AddedAPIKeys); got != 1 {
		t.Fatalf("AddedAPIKeys len = %d, want 1", got)
	}
	if got := len(diff.RemovedAPIKeys); got != 1 {
		t.Fatalf("RemovedAPIKeys len = %d, want 1", got)
	}
	if got := len(diff.ChangedAPIKeys); got != 1 {
		t.Fatalf("ChangedAPIKeys len = %d, want 1", got)
	}
	if len(diff.UnschedulableAPIKeyIDs) != 1 || diff.UnschedulableAPIKeyIDs[0] != "key-old" {
		t.Fatalf("UnschedulableAPIKeyIDs = %#v, want [key-old]", diff.UnschedulableAPIKeyIDs)
	}
}

func TestBuildUpstreamSyncDiffSkipsAPIKeysWhenNextListUnknown(t *testing.T) {
	diff := BuildUpstreamSyncDiff(
		[]*UpstreamRemoteGroup{{RemoteGroupID: "default", RemoteGroupName: "Default", RateMultiplier: 1, Status: UpstreamStatusActive}},
		[]*UpstreamRemoteAPIKey{{RemoteAPIKeyID: "key-old", RemoteAPIKeyName: "Old", SyncedRemoteGroupID: "default", RemoteGroupID: "default", LocalGroupIDs: []int64{11}}},
		[]*UpstreamRemoteGroup{{RemoteGroupID: "default", RemoteGroupName: "Default", RateMultiplier: 1, Status: UpstreamStatusActive}},
		nil,
	)

	if got := len(diff.RemovedAPIKeys); got != 0 {
		t.Fatalf("RemovedAPIKeys len = %d, want 0", got)
	}
	if got := len(diff.UnschedulableAPIKeyIDs); got != 0 {
		t.Fatalf("UnschedulableAPIKeyIDs len = %d, want 0", got)
	}
	if got := len(diff.AffectedLocalGroupIDs); got != 0 {
		t.Fatalf("AffectedLocalGroupIDs len = %d, want 0", got)
	}
}

func TestNormalizeGovernancePolicyDefaultsAndDeduplicatesCodes(t *testing.T) {
	policy := normalizeGovernancePolicy(UpstreamGovernancePolicy{
		IgnoredStatusCodes:          []int{429, 0, 429, 529},
		ImmediateCircuitStatusCodes: []int{401, 401, 500},
	})

	if policy.ConsecutiveFailuresToCircuitOpen != 5 {
		t.Fatalf("ConsecutiveFailuresToCircuitOpen = %d, want 5", policy.ConsecutiveFailuresToCircuitOpen)
	}
	if policy.FirstTokenDegradeThresholdMS != 8000 {
		t.Fatalf("FirstTokenDegradeThresholdMS = %d, want 8000", policy.FirstTokenDegradeThresholdMS)
	}
	if len(policy.IgnoredStatusCodes) != 2 || policy.IgnoredStatusCodes[0] != 429 || policy.IgnoredStatusCodes[1] != 529 {
		t.Fatalf("IgnoredStatusCodes = %#v, want [429 529]", policy.IgnoredStatusCodes)
	}
	if len(policy.ImmediateCircuitStatusCodes) != 2 || policy.ImmediateCircuitStatusCodes[0] != 401 || policy.ImmediateCircuitStatusCodes[1] != 500 {
		t.Fatalf("ImmediateCircuitStatusCodes = %#v, want [401 500]", policy.ImmediateCircuitStatusCodes)
	}
}
