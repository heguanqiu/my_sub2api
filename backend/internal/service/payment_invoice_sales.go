package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/invoicedocument"
	"github.com/Wei-Shaw/sub2api/ent/invoiceprofile"
	"github.com/Wei-Shaw/sub2api/ent/invoicerequest"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type InvoiceSettings struct {
	Enabled          bool
	Provider         string
	BaiwangEnabled   bool
	BaiwangBaseURL   string
	BaiwangAppKey    string
	BaiwangAppSecret string
	TaxpayerID       string
	SellerName       string
	DefaultGoodsName string
	AutoRetryEnabled bool
	RetryLimit       int
}

type InvoiceProfileInput struct {
	Title       string `json:"title"`
	TaxNo       string `json:"tax_no"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	BankName    string `json:"bank_name"`
	BankAccount string `json:"bank_account"`
	InvoiceType string `json:"invoice_type"`
	IsDefault   bool   `json:"is_default"`
}

type SalesDashboardSummary struct {
	TotalCustomers      int     `json:"total_customers"`
	TotalOrders         int     `json:"total_orders"`
	CompletedOrders     int     `json:"completed_orders"`
	TotalOrderAmount    float64 `json:"total_order_amount"`
	TotalInvoiceRecords int     `json:"total_invoice_records"`
}

type SalesCustomerSummary struct {
	User                 *User   `json:"user"`
	TotalOrders          int     `json:"total_orders"`
	CompletedOrderAmount float64 `json:"completed_order_amount"`
}

func (s *PaymentService) GetInvoiceSettings(ctx context.Context) (*InvoiceSettings, error) {
	keys := []string{
		SettingKeyInvoiceEnabled,
		SettingKeyInvoiceProvider,
		SettingKeyInvoiceBaiwangEnabled,
		SettingKeyInvoiceBaiwangBaseURL,
		SettingKeyInvoiceBaiwangAppKey,
		SettingKeyInvoiceBaiwangAppSecret,
		SettingKeyInvoiceBaiwangTaxpayerID,
		SettingKeyInvoiceBaiwangSellerName,
		SettingKeyInvoiceBaiwangDefaultGoodsName,
		SettingKeyInvoiceAutoRetryEnabled,
		SettingKeyInvoiceRetryLimit,
	}
	rows, err := s.entClient.Setting.Query().Where(setting.KeyIn(keys...)).All(ctx)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	retryLimit := 3
	if parsed, err := strconv.Atoi(strings.TrimSpace(values[SettingKeyInvoiceRetryLimit])); err == nil && parsed > 0 {
		retryLimit = parsed
	}
	provider := strings.TrimSpace(values[SettingKeyInvoiceProvider])
	if provider == "" {
		provider = InvoiceProviderBaiwang
	}
	return &InvoiceSettings{
		Enabled:          values[SettingKeyInvoiceEnabled] == "true",
		Provider:         provider,
		BaiwangEnabled:   values[SettingKeyInvoiceBaiwangEnabled] == "true",
		BaiwangBaseURL:   strings.TrimSpace(values[SettingKeyInvoiceBaiwangBaseURL]),
		BaiwangAppKey:    strings.TrimSpace(values[SettingKeyInvoiceBaiwangAppKey]),
		BaiwangAppSecret: strings.TrimSpace(values[SettingKeyInvoiceBaiwangAppSecret]),
		TaxpayerID:       strings.TrimSpace(values[SettingKeyInvoiceBaiwangTaxpayerID]),
		SellerName:       strings.TrimSpace(values[SettingKeyInvoiceBaiwangSellerName]),
		DefaultGoodsName: strings.TrimSpace(values[SettingKeyInvoiceBaiwangDefaultGoodsName]),
		AutoRetryEnabled: values[SettingKeyInvoiceAutoRetryEnabled] == "true",
		RetryLimit:       retryLimit,
	}, nil
}

func (s *PaymentService) ListInvoiceProfiles(ctx context.Context, userID int64) ([]*dbent.InvoiceProfile, error) {
	return s.entClient.InvoiceProfile.Query().
		Where(invoiceprofile.UserIDEQ(userID)).
		Order(dbent.Desc(invoiceprofile.FieldIsDefault), dbent.Desc(invoiceprofile.FieldCreatedAt)).
		All(ctx)
}

func (s *PaymentService) CreateInvoiceProfile(ctx context.Context, userID int64, input InvoiceProfileInput) (*dbent.InvoiceProfile, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, infraerrors.BadRequest("INVOICE_PROFILE_TITLE_REQUIRED", "invoice profile title is required")
	}
	if strings.TrimSpace(input.InvoiceType) == "" {
		return nil, infraerrors.BadRequest("INVOICE_PROFILE_TYPE_REQUIRED", "invoice type is required")
	}
	if input.IsDefault {
		if err := s.clearDefaultInvoiceProfiles(ctx, userID); err != nil {
			return nil, err
		}
	}
	return s.entClient.InvoiceProfile.Create().
		SetUserID(userID).
		SetTitle(strings.TrimSpace(input.Title)).
		SetNillableTaxNo(nilIfEmptyTrimmed(input.TaxNo)).
		SetNillableEmail(nilIfEmptyTrimmed(input.Email)).
		SetNillablePhone(nilIfEmptyTrimmed(input.Phone)).
		SetNillableAddress(nilIfEmptyTrimmed(input.Address)).
		SetNillableBankName(nilIfEmptyTrimmed(input.BankName)).
		SetNillableBankAccount(nilIfEmptyTrimmed(input.BankAccount)).
		SetInvoiceType(strings.TrimSpace(input.InvoiceType)).
		SetIsDefault(input.IsDefault).
		Save(ctx)
}

func (s *PaymentService) UpdateInvoiceProfile(ctx context.Context, userID, profileID int64, input InvoiceProfileInput) (*dbent.InvoiceProfile, error) {
	profile, err := s.entClient.InvoiceProfile.Get(ctx, profileID)
	if err != nil {
		return nil, infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	if profile.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice profile")
	}
	if input.IsDefault {
		if err := s.clearDefaultInvoiceProfiles(ctx, userID); err != nil {
			return nil, err
		}
	}
	return s.entClient.InvoiceProfile.UpdateOneID(profileID).
		SetTitle(strings.TrimSpace(input.Title)).
		SetNillableTaxNo(nilIfEmptyTrimmed(input.TaxNo)).
		SetNillableEmail(nilIfEmptyTrimmed(input.Email)).
		SetNillablePhone(nilIfEmptyTrimmed(input.Phone)).
		SetNillableAddress(nilIfEmptyTrimmed(input.Address)).
		SetNillableBankName(nilIfEmptyTrimmed(input.BankName)).
		SetNillableBankAccount(nilIfEmptyTrimmed(input.BankAccount)).
		SetInvoiceType(strings.TrimSpace(input.InvoiceType)).
		SetIsDefault(input.IsDefault).
		Save(ctx)
}

func (s *PaymentService) DeleteInvoiceProfile(ctx context.Context, userID, profileID int64) error {
	profile, err := s.entClient.InvoiceProfile.Get(ctx, profileID)
	if err != nil {
		return infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	if profile.UserID != userID {
		return infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice profile")
	}
	return s.entClient.InvoiceProfile.DeleteOneID(profileID).Exec(ctx)
}

func (s *PaymentService) SetDefaultInvoiceProfile(ctx context.Context, userID, profileID int64) (*dbent.InvoiceProfile, error) {
	profile, err := s.entClient.InvoiceProfile.Get(ctx, profileID)
	if err != nil {
		return nil, infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	if profile.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice profile")
	}
	if err := s.clearDefaultInvoiceProfiles(ctx, userID); err != nil {
		return nil, err
	}
	return s.entClient.InvoiceProfile.UpdateOneID(profileID).SetIsDefault(true).Save(ctx)
}

func (s *PaymentService) clearDefaultInvoiceProfiles(ctx context.Context, userID int64) error {
	_, err := s.entClient.InvoiceProfile.Update().
		Where(invoiceprofile.UserIDEQ(userID), invoiceprofile.IsDefaultEQ(true)).
		SetIsDefault(false).
		Save(ctx)
	return err
}

func (s *PaymentService) CreateInvoiceRequest(ctx context.Context, userID, orderID, profileID int64) (*dbent.InvoiceRequest, error) {
	settings, err := s.GetInvoiceSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, infraerrors.Forbidden("INVOICE_DISABLED", "invoice feature is disabled")
	}
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return nil, infraerrors.NotFound("ORDER_NOT_FOUND", "order not found")
	}
	if order.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this order")
	}
	if order.Status != OrderStatusCompleted {
		return nil, infraerrors.BadRequest("ORDER_NOT_COMPLETED", "only completed orders can request invoices")
	}
	if order.InvoiceStatus == OrderInvoiceStatusRequested || order.InvoiceStatus == OrderInvoiceStatusProcessing || order.InvoiceStatus == OrderInvoiceStatusIssued {
		return nil, infraerrors.Conflict("INVOICE_ALREADY_EXISTS", "invoice request already exists for this order")
	}
	profile, err := s.entClient.InvoiceProfile.Get(ctx, profileID)
	if err != nil {
		return nil, infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	if profile.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice profile")
	}
	req, err := s.entClient.InvoiceRequest.Create().
		SetUserID(userID).
		SetOrderID(orderID).
		SetProfileID(profileID).
		SetStatus(InvoiceRequestStatusRequested).
		SetProvider(settings.Provider).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.entClient.PaymentOrder.UpdateOneID(orderID).
		SetInvoiceStatus(OrderInvoiceStatusRequested).
		SetInvoiceRequestID(req.ID).
		Save(ctx); err != nil {
		return nil, err
	}
	if _, err := s.processInvoiceRequest(ctx, req.ID, false); err != nil {
		return s.entClient.InvoiceRequest.Get(ctx, req.ID)
	}
	return s.entClient.InvoiceRequest.Get(ctx, req.ID)
}

func (s *PaymentService) ListUserInvoiceRequests(ctx context.Context, userID int64, page, pageSize int) ([]*dbent.InvoiceRequest, int, error) {
	q := s.entClient.InvoiceRequest.Query().Where(invoicerequest.UserIDEQ(userID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(invoicerequest.FieldCreatedAt)).Offset(pageOffset(page, pageSize)).Limit(limitValue(pageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *PaymentService) GetUserInvoiceRequest(ctx context.Context, userID, invoiceID int64) (*dbent.InvoiceRequest, error) {
	req, err := s.entClient.InvoiceRequest.Get(ctx, invoiceID)
	if err != nil {
		return nil, infraerrors.NotFound("INVOICE_REQUEST_NOT_FOUND", "invoice request not found")
	}
	if req.UserID != userID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "no permission for this invoice request")
	}
	return req, nil
}

func (s *PaymentService) ListAdminInvoiceRequests(ctx context.Context, page, pageSize int, status, keyword string) ([]*dbent.InvoiceRequest, int, error) {
	q := s.entClient.InvoiceRequest.Query()
	if status != "" {
		q = q.Where(invoicerequest.StatusEQ(strings.TrimSpace(status)))
	}
	if keyword != "" {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			q = q.Where(invoicerequest.Or(
				invoicerequest.ProviderRequestIDContainsFold(keyword),
				invoicerequest.ProviderInvoiceIDContainsFold(keyword),
			))
		}
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(invoicerequest.FieldCreatedAt)).Offset(pageOffset(page, pageSize)).Limit(limitValue(pageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *PaymentService) GetInvoiceRequestDetail(ctx context.Context, invoiceID int64) (*dbent.InvoiceRequest, *dbent.PaymentOrder, *dbent.InvoiceProfile, []*dbent.InvoiceDocument, error) {
	req, err := s.entClient.InvoiceRequest.Get(ctx, invoiceID)
	if err != nil {
		return nil, nil, nil, nil, infraerrors.NotFound("INVOICE_REQUEST_NOT_FOUND", "invoice request not found")
	}
	order, _ := s.entClient.PaymentOrder.Get(ctx, req.OrderID)
	profile, _ := s.entClient.InvoiceProfile.Get(ctx, req.ProfileID)
	documents, _ := s.entClient.InvoiceDocument.Query().
		Where(invoicedocument.InvoiceRequestIDEQ(req.ID)).
		Order(dbent.Desc(invoicedocument.FieldCreatedAt)).
		All(ctx)
	return req, order, profile, documents, nil
}

func (s *PaymentService) RetryInvoiceRequest(ctx context.Context, invoiceID int64) (*dbent.InvoiceRequest, error) {
	return s.processInvoiceRequest(ctx, invoiceID, true)
}

func (s *PaymentService) processInvoiceRequest(ctx context.Context, invoiceID int64, isRetry bool) (*dbent.InvoiceRequest, error) {
	req, err := s.entClient.InvoiceRequest.Get(ctx, invoiceID)
	if err != nil {
		return nil, infraerrors.NotFound("INVOICE_REQUEST_NOT_FOUND", "invoice request not found")
	}
	settings, err := s.GetInvoiceSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || !settings.BaiwangEnabled {
		return s.failInvoiceRequest(ctx, req, "invoice provider is disabled")
	}
	order, err := s.entClient.PaymentOrder.Get(ctx, req.OrderID)
	if err != nil {
		return nil, infraerrors.NotFound("ORDER_NOT_FOUND", "order not found")
	}
	profile, err := s.entClient.InvoiceProfile.Get(ctx, req.ProfileID)
	if err != nil {
		return nil, infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	update := s.entClient.InvoiceRequest.UpdateOneID(req.ID).
		SetStatus(InvoiceRequestStatusProcessing).
		SetRetryCount(req.RetryCount)
	if isRetry {
		update = update.AddRetryCount(1)
	}
	req, err = update.Save(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.entClient.PaymentOrder.UpdateOneID(order.ID).SetInvoiceStatus(OrderInvoiceStatusProcessing).SetInvoiceRequestID(req.ID).Save(ctx); err != nil {
		return nil, err
	}

	if strings.EqualFold(settings.BaiwangBaseURL, "mock") || strings.HasPrefix(strings.ToLower(settings.BaiwangBaseURL), "mock://") {
		now := time.Now()
		req, err = s.entClient.InvoiceRequest.UpdateOneID(req.ID).
			SetStatus(InvoiceRequestStatusIssued).
			SetProviderRequestID(fmt.Sprintf("mock-req-%d", req.ID)).
			SetProviderInvoiceID(fmt.Sprintf("mock-invoice-%d", req.ID)).
			SetIssuedAt(now).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := s.entClient.InvoiceDocument.Create().
			SetInvoiceRequestID(req.ID).
			SetInvoiceNo(fmt.Sprintf("INV-%06d", req.ID)).
			SetInvoiceCode("MOCK").
			SetFileURL(fmt.Sprintf("mock://invoice/%d.pdf", req.ID)).
			SetFileType(InvoiceDocumentTypePDF).
			SetRawPayloadSummary(map[string]any{
				"mode":        "mock",
				"order_id":    order.ID,
				"profile_id":  profile.ID,
				"invoice_req": req.ID,
			}).
			Save(ctx); err != nil {
			return nil, err
		}
		if _, err := s.entClient.PaymentOrder.UpdateOneID(order.ID).SetInvoiceStatus(OrderInvoiceStatusIssued).SetInvoiceRequestID(req.ID).Save(ctx); err != nil {
			return nil, err
		}
		return req, nil
	}
	return s.failInvoiceRequest(ctx, req, "baiwang integration is not configured for non-mock mode")
}

func (s *PaymentService) failInvoiceRequest(ctx context.Context, req *dbent.InvoiceRequest, reason string) (*dbent.InvoiceRequest, error) {
	updated, err := s.entClient.InvoiceRequest.UpdateOneID(req.ID).
		SetStatus(InvoiceRequestStatusFailed).
		SetFailReason(reason).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.entClient.PaymentOrder.UpdateOneID(req.OrderID).SetInvoiceStatus(OrderInvoiceStatusFailed).SetInvoiceRequestID(req.ID).Save(ctx); err != nil {
		return nil, err
	}
	return updated, nil
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
	customerRows, err := s.entClient.User.Query().
		Where(dbuser.OwnerSalesIDEQ(salesUserID), dbuser.RoleEQ(RoleUser)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	customerIDs := make([]int64, 0, len(customerRows))
	for _, customer := range customerRows {
		customerIDs = append(customerIDs, customer.ID)
	}
	totalInvoices := 0
	if len(customerIDs) > 0 {
		totalInvoices, err = s.entClient.InvoiceRequest.Query().Where(invoicerequest.UserIDIn(customerIDs...)).Count(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &SalesDashboardSummary{
		TotalCustomers:      customerCount,
		TotalOrders:         totalOrders,
		CompletedOrders:     len(completedOrders),
		TotalOrderAmount:    completedAmount,
		TotalInvoiceRecords: totalInvoices,
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

func (s *PaymentService) GetSalesCustomerInvoices(ctx context.Context, salesUserID, customerID int64, page, pageSize int) ([]*dbent.InvoiceRequest, int, error) {
	if err := s.ensureSalesCustomerAccess(ctx, salesUserID, customerID); err != nil {
		return nil, 0, err
	}
	q := s.entClient.InvoiceRequest.Query().Where(invoicerequest.UserIDEQ(customerID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(invoicerequest.FieldCreatedAt)).Offset(pageOffset(page, pageSize)).Limit(limitValue(pageSize)).All(ctx)
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
