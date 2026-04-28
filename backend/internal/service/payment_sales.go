package service

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type SalesDashboardSummary struct {
	TotalCustomers   int     `json:"total_customers"`
	TotalOrders      int     `json:"total_orders"`
	CompletedOrders  int     `json:"completed_orders"`
	TotalOrderAmount float64 `json:"total_order_amount"`
}

type SalesCustomerSummary struct {
	User                 *User   `json:"user"`
	TotalOrders          int     `json:"total_orders"`
	CompletedOrderAmount float64 `json:"completed_order_amount"`
}

func (s *PaymentService) GetSalesDashboard(ctx context.Context, salesUserID int64) (*SalesDashboardSummary, error) {
	if err := s.ensureSalesAccess(ctx, salesUserID); err != nil {
		return nil, err
	}
	customerCount, err := s.entClient.User.Query().
		Where(dbuser.OwnerSalesIDEQ(salesUserID), dbuser.RoleEQ(RoleUser)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	totalOrders, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.HasUserWith(dbuser.OwnerSalesIDEQ(salesUserID), dbuser.RoleEQ(RoleUser))).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	completedOrders, err := s.entClient.PaymentOrder.Query().Where(
		paymentorder.HasUserWith(dbuser.OwnerSalesIDEQ(salesUserID), dbuser.RoleEQ(RoleUser)),
		paymentorder.StatusEQ(OrderStatusCompleted),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	var completedAmount float64
	for _, order := range completedOrders {
		completedAmount += order.Amount
	}
	return &SalesDashboardSummary{
		TotalCustomers:   customerCount,
		TotalOrders:      totalOrders,
		CompletedOrders:  len(completedOrders),
		TotalOrderAmount: completedAmount,
	}, nil
}

func (s *PaymentService) ListSalesCustomers(ctx context.Context, salesUserID int64, page, pageSize int, search, status string) ([]SalesCustomerSummary, int, error) {
	if err := s.ensureSalesAccess(ctx, salesUserID); err != nil {
		return nil, 0, err
	}
	q := s.entClient.User.Query().Where(dbuser.OwnerSalesIDEQ(salesUserID), dbuser.RoleEQ(RoleUser))
	search = strings.TrimSpace(search)
	if search != "" {
		q = q.Where(dbuser.Or(dbuser.EmailContainsFold(search), dbuser.UsernameContainsFold(search)))
	}
	status = strings.TrimSpace(status)
	if status != "" {
		q = q.Where(dbuser.StatusEQ(status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(dbuser.FieldCreatedAt)).Offset(pageOffset(page, pageSize)).Limit(limitValue(pageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]SalesCustomerSummary, 0, len(rows))
	for _, row := range rows {
		orderRows, err := s.entClient.PaymentOrder.Query().
			Where(paymentorder.UserIDEQ(row.ID)).
			All(ctx)
		if err != nil {
			return nil, 0, err
		}
		var completedAmount float64
		for _, order := range orderRows {
			if order.Status == OrderStatusCompleted {
				completedAmount += order.Amount
			}
		}
		out = append(out, SalesCustomerSummary{
			User:                 authUserEntityToService(row),
			TotalOrders:          len(orderRows),
			CompletedOrderAmount: completedAmount,
		})
	}
	return out, total, nil
}

func (s *PaymentService) GetSalesCustomer(ctx context.Context, salesUserID, customerID int64) (*User, error) {
	if err := s.ensureSalesCustomerAccess(ctx, salesUserID, customerID); err != nil {
		return nil, err
	}
	row, err := s.entClient.User.Get(ctx, customerID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return authUserEntityToService(row), nil
}

func (s *PaymentService) ListSalesOrders(ctx context.Context, salesUserID int64, p OrderListParams) ([]*dbent.PaymentOrder, int, error) {
	if err := s.ensureSalesAccess(ctx, salesUserID); err != nil {
		return nil, 0, err
	}
	q := s.entClient.PaymentOrder.Query().
		Where(paymentorder.HasUserWith(dbuser.OwnerSalesIDEQ(salesUserID), dbuser.RoleEQ(RoleUser)))
	if p.Status != "" {
		q = q.Where(paymentorder.StatusEQ(p.Status))
	}
	if p.PaymentType != "" {
		q = q.Where(paymentorder.PaymentTypeEQ(p.PaymentType))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(paymentorder.FieldCreatedAt)).Offset(pageOffset(p.Page, p.PageSize)).Limit(limitValue(p.PageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *PaymentService) GetSalesCustomerOrders(ctx context.Context, salesUserID, customerID int64, p OrderListParams) ([]*dbent.PaymentOrder, int, error) {
	if err := s.ensureSalesCustomerAccess(ctx, salesUserID, customerID); err != nil {
		return nil, 0, err
	}
	q := s.entClient.PaymentOrder.Query().Where(paymentorder.UserIDEQ(customerID))
	if p.Status != "" {
		q = q.Where(paymentorder.StatusEQ(p.Status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(paymentorder.FieldCreatedAt)).Offset(pageOffset(p.Page, p.PageSize)).Limit(limitValue(p.PageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *PaymentService) ensureSalesAccess(ctx context.Context, salesUserID int64) error {
	user, err := s.userRepo.GetByID(ctx, salesUserID)
	if err != nil {
		return ErrUserNotFound
	}
	if !user.IsSales() {
		return infraerrors.Forbidden("FORBIDDEN", "sales access required")
	}
	return nil
}

func (s *PaymentService) ensureSalesCustomerAccess(ctx context.Context, salesUserID, customerID int64) error {
	if err := s.ensureSalesAccess(ctx, salesUserID); err != nil {
		return err
	}
	row, err := s.entClient.User.Get(ctx, customerID)
	if err != nil {
		return ErrUserNotFound
	}
	if row.OwnerSalesID == nil || *row.OwnerSalesID != salesUserID {
		return infraerrors.Forbidden("FORBIDDEN", "customer does not belong to current sales user")
	}
	return nil
}

func pageOffset(page, pageSize int) int {
	if page <= 0 {
		page = 1
	}
	return (page - 1) * limitValue(pageSize)
}

func limitValue(pageSize int) int {
	if pageSize <= 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}
