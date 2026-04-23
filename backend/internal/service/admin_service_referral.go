package service

import (
	"context"
	"sort"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *adminServiceImpl) GetReferralTree(ctx context.Context, userID int64) (*ReferralTreeNode, error) {
	rows, root, childrenByParent, err := s.loadReferralSubtree(ctx, userID)
	if err != nil {
		return nil, err
	}
	serviceUsers := make(map[int64]*User, len(rows))
	for _, row := range rows {
		serviceUsers[row.ID] = adminUserEntityToService(row)
	}
	var build func(id int64) *ReferralTreeNode
	build = func(id int64) *ReferralTreeNode {
		node := &ReferralTreeNode{
			User:     serviceUsers[id],
			Children: []*ReferralTreeNode{},
		}
		for _, child := range childrenByParent[id] {
			node.Children = append(node.Children, build(child.ID))
		}
		return node
	}
	return build(root.ID), nil
}

func (s *adminServiceImpl) ChangeInviter(ctx context.Context, userID int64, newInvitedByUserID *int64) (*ReferralMutationResult, error) {
	root, err := s.entClient.User.Get(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if newInvitedByUserID != nil {
		if *newInvitedByUserID == userID {
			return nil, infraerrors.BadRequest("REFERRAL_CYCLE", "cannot set user as their own inviter")
		}
		descendants, err := s.collectReferralSubtreeIDs(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, id := range descendants {
			if id == *newInvitedByUserID {
				return nil, infraerrors.BadRequest("REFERRAL_CYCLE", "cannot move user under a descendant")
			}
		}
		if _, err := s.entClient.User.Get(ctx, *newInvitedByUserID); err != nil {
			return nil, ErrUserNotFound
		}
	}
	update := s.entClient.User.UpdateOneID(userID)
	if newInvitedByUserID == nil {
		update = update.ClearInvitedByUserID()
	} else {
		update = update.SetInvitedByUserID(*newInvitedByUserID)
	}
	if _, err := update.Save(ctx); err != nil {
		return nil, err
	}
	result, err := s.recomputeReferralOwnerSales(ctx, root.ID)
	if err != nil {
		return nil, err
	}
	result.NewInvitedByUserID = newInvitedByUserID
	return result, nil
}

func (s *adminServiceImpl) RecomputeSalesOwner(ctx context.Context, userID int64) (*ReferralMutationResult, error) {
	return s.recomputeReferralOwnerSales(ctx, userID)
}

func (s *adminServiceImpl) PreviewSalesOwnerMigration(ctx context.Context, userID, targetSalesUserID int64) (*ReferralMutationResult, error) {
	if err := s.ensureSalesUser(ctx, targetSalesUserID); err != nil {
		return nil, err
	}
	ids, err := s.collectReferralSubtreeIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	target := targetSalesUserID
	return &ReferralMutationResult{
		RootUserID:        userID,
		AffectedUserCount: len(ids),
		AffectedUserIDs:   ids,
		TargetSalesUserID: &target,
	}, nil
}

func (s *adminServiceImpl) MigrateSalesOwner(ctx context.Context, userID, targetSalesUserID int64) (*ReferralMutationResult, error) {
	if err := s.ensureSalesUser(ctx, targetSalesUserID); err != nil {
		return nil, err
	}
	ids, err := s.collectReferralSubtreeIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.entClient.User.Query().Where(dbuser.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	roleByID := make(map[int64]string, len(rows))
	for _, row := range rows {
		roleByID[row.ID] = row.Role
	}
	for _, id := range ids {
		update := s.entClient.User.UpdateOneID(id)
		if roleByID[id] == RoleSales {
			update = update.ClearOwnerSalesID()
		} else {
			update = update.SetOwnerSalesID(targetSalesUserID)
		}
		if _, err := update.Save(ctx); err != nil {
			return nil, err
		}
	}
	target := targetSalesUserID
	return &ReferralMutationResult{
		RootUserID:        userID,
		AffectedUserCount: len(ids),
		AffectedUserIDs:   ids,
		TargetSalesUserID: &target,
	}, nil
}

func (s *adminServiceImpl) ensureSalesUser(ctx context.Context, userID int64) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if !user.IsSales() {
		return infraerrors.BadRequest("INVALID_SALES_USER", "target sales user is not a sales account")
	}
	return nil
}

func (s *adminServiceImpl) recomputeReferralOwnerSales(ctx context.Context, userID int64) (*ReferralMutationResult, error) {
	rows, root, childrenByParent, err := s.loadReferralSubtree(ctx, userID)
	if err != nil {
		return nil, err
	}
	rowByID := make(map[int64]*dbent.User, len(rows))
	for _, row := range rows {
		rowByID[row.ID] = row
	}
	var rootOwner *int64
	if root.InvitedByUserID != nil {
		inviter, err := s.userRepo.GetByID(ctx, *root.InvitedByUserID)
		if err == nil {
			rootOwner = deriveOwnerSalesIDForReferral(inviter)
		}
	}
	var affected []int64
	var walk func(nodeID int64, inheritedOwner *int64) error
	walk = func(nodeID int64, inheritedOwner *int64) error {
		node := rowByID[nodeID]
		update := s.entClient.User.UpdateOneID(nodeID)
		if node.Role == RoleSales {
			update = update.ClearOwnerSalesID()
			inheritedOwner = cloneInt64Ptr(&node.ID)
		} else if inheritedOwner == nil {
			update = update.ClearOwnerSalesID()
		} else {
			update = update.SetOwnerSalesID(*inheritedOwner)
		}
		if _, err := update.Save(ctx); err != nil {
			return err
		}
		affected = append(affected, nodeID)
		nextOwner := inheritedOwner
		if node.Role == RoleSales {
			nextOwner = cloneInt64Ptr(&node.ID)
		}
		for _, child := range childrenByParent[nodeID] {
			if err := walk(child.ID, nextOwner); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root.ID, rootOwner); err != nil {
		return nil, err
	}
	sort.Slice(affected, func(i, j int) bool { return affected[i] < affected[j] })
	return &ReferralMutationResult{
		RootUserID:        userID,
		AffectedUserCount: len(affected),
		AffectedUserIDs:   affected,
	}, nil
}

func (s *adminServiceImpl) collectReferralSubtreeIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, _, _, err := s.loadReferralSubtree(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (s *adminServiceImpl) loadReferralSubtree(ctx context.Context, userID int64) ([]*dbent.User, *dbent.User, map[int64][]*dbent.User, error) {
	root, err := s.entClient.User.Get(ctx, userID)
	if err != nil {
		return nil, nil, nil, ErrUserNotFound
	}
	allRows, err := s.entClient.User.Query().Select(
		dbuser.FieldID,
		dbuser.FieldEmail,
		dbuser.FieldUsername,
		dbuser.FieldRole,
		dbuser.FieldStatus,
		dbuser.FieldInvitedByUserID,
		dbuser.FieldOwnerSalesID,
		dbuser.FieldCreatedAt,
		dbuser.FieldUpdatedAt,
	).All(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	childrenByParent := make(map[int64][]*dbent.User)
	for _, row := range allRows {
		if row.InvitedByUserID != nil {
			childrenByParent[*row.InvitedByUserID] = append(childrenByParent[*row.InvitedByUserID], row)
		}
	}
	var out []*dbent.User
	queue := []*dbent.User{root}
	seen := map[int64]struct{}{root.ID: {}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		out = append(out, current)
		for _, child := range childrenByParent[current.ID] {
			if _, ok := seen[child.ID]; ok {
				continue
			}
			seen[child.ID] = struct{}{}
			queue = append(queue, child)
		}
	}
	for parentID := range childrenByParent {
		sort.Slice(childrenByParent[parentID], func(i, j int) bool {
			return childrenByParent[parentID][i].CreatedAt.Before(childrenByParent[parentID][j].CreatedAt)
		})
	}
	return out, root, childrenByParent, nil
}

func adminUserEntityToService(u *dbent.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:              u.ID,
		Email:           u.Email,
		Username:        u.Username,
		Role:            u.Role,
		Status:          u.Status,
		InvitedByUserID: u.InvitedByUserID,
		OwnerSalesID:    u.OwnerSalesID,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}
