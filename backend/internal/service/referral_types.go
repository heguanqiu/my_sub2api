package service

type ReferralTreeNode struct {
	User     *User               `json:"user"`
	Children []*ReferralTreeNode `json:"children"`
}

type ReferralMutationResult struct {
	RootUserID         int64   `json:"root_user_id"`
	AffectedUserCount  int     `json:"affected_user_count"`
	AffectedUserIDs    []int64 `json:"affected_user_ids,omitempty"`
	TargetSalesUserID  *int64  `json:"target_sales_user_id,omitempty"`
	NewInvitedByUserID *int64  `json:"new_invited_by_user_id,omitempty"`
}
