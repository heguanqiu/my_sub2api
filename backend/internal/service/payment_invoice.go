package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/invoiceprofile"
	"github.com/Wei-Shaw/sub2api/ent/invoicerequest"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	invoice "github.com/fapiaoapi/invoice-sdk-golang"
	"github.com/shopspring/decimal"
)

const (
	InvoiceStatusPending      = "PENDING"
	InvoiceStatusIssuing      = "ISSUING"
	InvoiceStatusIssued       = "ISSUED"
	InvoiceStatusRequiresAuth = "REQUIRES_AUTH"
	InvoiceStatusFailed       = "FAILED"
)

type UserInvoiceSummary struct {
	TotalPaid       float64 `json:"total_paid"`
	InvoicedAmount  float64 `json:"invoiced_amount"`
	ReservedAmount  float64 `json:"reserved_amount"`
	AvailableAmount float64 `json:"available_amount"`
	MinAmount       float64 `json:"min_amount"`
	Currency        string  `json:"currency"`
	Enabled         bool    `json:"enabled"`
}

type UserInvoiceProfile struct {
	ID           int64     `json:"id"`
	TitleType    string    `json:"title_type"`
	Name         string    `json:"name"`
	TaxNo        string    `json:"tax_no"`
	AddressPhone string    `json:"address_phone"`
	BankAccount  string    `json:"bank_account"`
	Email        string    `json:"email"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserInvoiceRequest struct {
	ID              int64      `json:"id"`
	ProfileID       *int64     `json:"profile_id,omitempty"`
	Status          string     `json:"status"`
	Amount          float64    `json:"amount"`
	PaidTotal       float64    `json:"paid_total"`
	InvoicedTotal   float64    `json:"invoiced_total"`
	ReservedTotal   float64    `json:"reserved_total"`
	AvailableAmount float64    `json:"available_amount"`
	Currency        string     `json:"currency"`
	TitleType       string     `json:"title_type"`
	TitleName       string     `json:"title_name"`
	TaxNo           string     `json:"tax_no"`
	AddressPhone    string     `json:"address_phone"`
	BankAccount     string     `json:"bank_account"`
	Email           string     `json:"email"`
	Content         string     `json:"content"`
	Remark          string     `json:"remark"`
	SDKCode         *int       `json:"sdk_code,omitempty"`
	SDKMessage      string     `json:"sdk_message"`
	InvoiceNo       string     `json:"invoice_no"`
	InvoiceDate     string     `json:"invoice_date"`
	PDFURL          string     `json:"pdf_url"`
	OFDURL          string     `json:"ofd_url"`
	XMLURL          string     `json:"xml_url"`
	IssuedAt        *time.Time `json:"issued_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type InvoiceProfileInput struct {
	TitleType    string `json:"title_type"`
	Name         string `json:"name"`
	TaxNo        string `json:"tax_no"`
	AddressPhone string `json:"address_phone"`
	BankAccount  string `json:"bank_account"`
	Email        string `json:"email"`
	IsDefault    bool   `json:"is_default"`
}

type CreateInvoiceApplicationRequest struct {
	ProfileID int64   `json:"profile_id"`
	Amount    float64 `json:"amount"`
	Content   string  `json:"content"`
	Email     string  `json:"email"`
	Remark    string  `json:"remark"`
}

type invoiceTotals struct {
	totalPaid       decimal.Decimal
	invoicedAmount  decimal.Decimal
	reservedAmount  decimal.Decimal
	availableAmount decimal.Decimal
}

type invoiceSDKResult struct {
	status      string
	code        *int
	message     string
	response    map[string]any
	invoiceNo   string
	invoiceDate string
	pdfURL      string
	ofdURL      string
	xmlURL      string
	issuedAt    *time.Time
}

func (s *PaymentService) GetUserInvoiceSummary(ctx context.Context, userID int64) (*UserInvoiceSummary, error) {
	cfg, err := s.configService.GetInvoiceConfig(ctx)
	if err != nil {
		return nil, err
	}
	totals, err := s.computeUserInvoiceTotals(ctx, userID)
	if err != nil {
		return nil, err
	}
	return invoiceSummaryFromTotals(cfg, totals), nil
}

func (s *PaymentService) ListUserInvoiceProfiles(ctx context.Context, userID int64) ([]UserInvoiceProfile, error) {
	profiles, err := s.entClient.InvoiceProfile.Query().
		Where(invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil()).
		Order(dbent.Desc(invoiceprofile.FieldIsDefault), dbent.Desc(invoiceprofile.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query invoice profiles: %w", err)
	}
	out := make([]UserInvoiceProfile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, invoiceProfileFromEnt(p))
	}
	return out, nil
}

func (s *PaymentService) CreateUserInvoiceProfile(ctx context.Context, userID int64, input InvoiceProfileInput) (*UserInvoiceProfile, error) {
	normalized, err := normalizeInvoiceProfileInput(input)
	if err != nil {
		return nil, err
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin invoice profile transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	count, err := tx.InvoiceProfile.Query().
		Where(invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count invoice profiles: %w", err)
	}
	isDefault := normalized.IsDefault || count == 0
	if isDefault {
		if _, err := tx.InvoiceProfile.Update().
			Where(invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil()).
			SetIsDefault(false).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("clear default invoice profiles: %w", err)
		}
	}
	profile, err := tx.InvoiceProfile.Create().
		SetUserID(userID).
		SetTitleType(normalized.TitleType).
		SetName(normalized.Name).
		SetTaxNo(normalized.TaxNo).
		SetAddressPhone(normalized.AddressPhone).
		SetBankAccount(normalized.BankAccount).
		SetEmail(normalized.Email).
		SetIsDefault(isDefault).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create invoice profile: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice profile transaction: %w", err)
	}
	result := invoiceProfileFromEnt(profile)
	return &result, nil
}

func (s *PaymentService) UpdateUserInvoiceProfile(ctx context.Context, userID, profileID int64, input InvoiceProfileInput) (*UserInvoiceProfile, error) {
	normalized, err := normalizeInvoiceProfileInput(input)
	if err != nil {
		return nil, err
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin invoice profile transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	exists, err := tx.InvoiceProfile.Query().
		Where(invoiceprofile.IDEQ(profileID), invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check invoice profile: %w", err)
	}
	if !exists {
		return nil, infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	if normalized.IsDefault {
		if _, err := tx.InvoiceProfile.Update().
			Where(invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil(), invoiceprofile.IDNEQ(profileID)).
			SetIsDefault(false).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("clear default invoice profiles: %w", err)
		}
	}
	profile, err := tx.InvoiceProfile.UpdateOneID(profileID).
		SetTitleType(normalized.TitleType).
		SetName(normalized.Name).
		SetTaxNo(normalized.TaxNo).
		SetAddressPhone(normalized.AddressPhone).
		SetBankAccount(normalized.BankAccount).
		SetEmail(normalized.Email).
		SetIsDefault(normalized.IsDefault).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice profile: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice profile transaction: %w", err)
	}
	result := invoiceProfileFromEnt(profile)
	return &result, nil
}

func (s *PaymentService) DeleteUserInvoiceProfile(ctx context.Context, userID, profileID int64) error {
	now := time.Now()
	count, err := s.entClient.InvoiceProfile.Update().
		Where(invoiceprofile.IDEQ(profileID), invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil()).
		SetDeletedAt(now).
		SetIsDefault(false).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("delete invoice profile: %w", err)
	}
	if count == 0 {
		return infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
	}
	return nil
}

func (s *PaymentService) ListUserInvoiceRequests(ctx context.Context, userID int64, p OrderListParams) ([]UserInvoiceRequest, int, error) {
	q := s.entClient.InvoiceRequest.Query().Where(invoicerequest.UserIDEQ(userID))
	if p.Status != "" {
		q = q.Where(invoicerequest.StatusEQ(p.Status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count invoice requests: %w", err)
	}
	ps, pg := applyPagination(p.PageSize, p.Page)
	requests, err := q.Order(dbent.Desc(invoicerequest.FieldCreatedAt)).
		Limit(ps).
		Offset((pg - 1) * ps).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query invoice requests: %w", err)
	}
	out := make([]UserInvoiceRequest, 0, len(requests))
	for _, r := range requests {
		out = append(out, invoiceRequestFromEnt(r))
	}
	return out, total, nil
}

func (s *PaymentService) CreateUserInvoiceRequest(ctx context.Context, userID int64, input CreateInvoiceApplicationRequest) (*UserInvoiceRequest, error) {
	cfg, err := s.configService.GetInvoiceConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateInvoiceConfigReady(cfg); err != nil {
		return nil, err
	}
	amount := decimal.NewFromFloat(input.Amount).Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, infraerrors.BadRequest("INVALID_INVOICE_AMOUNT", "invoice amount must be positive")
	}
	minAmount := decimal.NewFromFloat(cfg.MinAmount).Round(2)
	if minAmount.GreaterThan(decimal.Zero) && amount.LessThan(minAmount) {
		return nil, infraerrors.BadRequest("INVOICE_AMOUNT_BELOW_MINIMUM", "invoice amount below minimum").
			WithMetadata(map[string]string{"min": minAmount.StringFixed(2)})
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		content = cfg.DefaultContent
	}
	request, err := s.reserveUserInvoiceRequest(ctx, userID, input, amount, content, cfg)
	if err != nil {
		return nil, err
	}

	result := s.issueInvoiceRequestWithSDK(request, cfg)
	update := s.entClient.InvoiceRequest.UpdateOneID(request.ID).
		SetStatus(result.status).
		SetSdkMessage(result.message).
		SetSdkResponse(result.response).
		SetInvoiceNo(result.invoiceNo).
		SetInvoiceDate(result.invoiceDate).
		SetPdfURL(result.pdfURL).
		SetOfdURL(result.ofdURL).
		SetXMLURL(result.xmlURL)
	if result.code != nil {
		update.SetSdkCode(*result.code)
	}
	if result.issuedAt != nil {
		update.SetIssuedAt(*result.issuedAt)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice request result: %w", err)
	}
	out := invoiceRequestFromEnt(updated)
	return &out, nil
}

func (s *PaymentService) reserveUserInvoiceRequest(ctx context.Context, userID int64, input CreateInvoiceApplicationRequest, amount decimal.Decimal, content string, cfg *InvoiceConfig) (*dbent.InvoiceRequest, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin invoice request transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx); err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, fmt.Errorf("lock invoice user: %w", err)
	}

	profile, err := tx.InvoiceProfile.Query().
		Where(invoiceprofile.IDEQ(input.ProfileID), invoiceprofile.UserIDEQ(userID), invoiceprofile.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("INVOICE_PROFILE_NOT_FOUND", "invoice profile not found")
		}
		return nil, fmt.Errorf("get invoice profile: %w", err)
	}
	email := strings.TrimSpace(input.Email)
	if email == "" {
		email = strings.TrimSpace(profile.Email)
	}
	if !isValidInvoiceEmail(email) {
		return nil, infraerrors.BadRequest("INVALID_INVOICE_EMAIL", "invoice email is invalid")
	}

	totals, err := computeUserInvoiceTotalsWithTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if amount.GreaterThan(totals.availableAmount) {
		return nil, infraerrors.BadRequest("INVOICE_AMOUNT_EXCEEDS_AVAILABLE", "invoice amount exceeds available amount").
			WithMetadata(map[string]string{"available": totals.availableAmount.StringFixed(2)})
	}

	request, err := tx.InvoiceRequest.Create().
		SetUserID(userID).
		SetProfileID(profile.ID).
		SetStatus(InvoiceStatusIssuing).
		SetAmount(amount.InexactFloat64()).
		SetPaidTotal(totals.totalPaid.InexactFloat64()).
		SetInvoicedTotal(totals.invoicedAmount.InexactFloat64()).
		SetReservedTotal(totals.reservedAmount.InexactFloat64()).
		SetAvailableAmount(totals.availableAmount.InexactFloat64()).
		SetCurrency(cfg.Currency).
		SetTitleType(profile.TitleType).
		SetTitleName(profile.Name).
		SetTaxNo(profile.TaxNo).
		SetAddressPhone(profile.AddressPhone).
		SetBankAccount(profile.BankAccount).
		SetEmail(email).
		SetContent(content).
		SetRemark(strings.TrimSpace(input.Remark)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create invoice request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice request transaction: %w", err)
	}
	return request, nil
}

func (s *PaymentService) computeUserInvoiceTotals(ctx context.Context, userID int64) (*invoiceTotals, error) {
	orders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.OrderTypeEQ(payment.OrderTypeBalance),
			paymentorder.StatusIn(
				OrderStatusCompleted,
				OrderStatusPaid,
				OrderStatusRecharging,
				OrderStatusRefundRequested,
				OrderStatusRefunding,
				OrderStatusRefundFailed,
				OrderStatusPartiallyRefunded,
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query invoice-eligible payment orders: %w", err)
	}
	totalPaid := decimal.Zero
	for _, order := range orders {
		if PaymentOrderCurrency(order) != payment.DefaultPaymentCurrency {
			continue
		}
		paid := decimal.NewFromFloat(order.PayAmount).Sub(decimal.NewFromFloat(order.RefundAmount)).Round(2)
		if paid.IsPositive() {
			totalPaid = totalPaid.Add(paid)
		}
	}

	requests, err := s.entClient.InvoiceRequest.Query().
		Where(invoicerequest.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query invoice request totals: %w", err)
	}
	invoicedAmount := decimal.Zero
	reservedAmount := decimal.Zero
	for _, request := range requests {
		amount := decimal.NewFromFloat(request.Amount).Round(2)
		switch request.Status {
		case InvoiceStatusIssued:
			invoicedAmount = invoicedAmount.Add(amount)
		case InvoiceStatusPending, InvoiceStatusIssuing, InvoiceStatusRequiresAuth:
			reservedAmount = reservedAmount.Add(amount)
		}
	}
	available := totalPaid.Sub(invoicedAmount).Sub(reservedAmount).Round(2)
	if available.IsNegative() {
		available = decimal.Zero
	}
	return &invoiceTotals{
		totalPaid:       totalPaid.Round(2),
		invoicedAmount:  invoicedAmount.Round(2),
		reservedAmount:  reservedAmount.Round(2),
		availableAmount: available,
	}, nil
}

func computeUserInvoiceTotalsWithTx(ctx context.Context, tx *dbent.Tx, userID int64) (*invoiceTotals, error) {
	orders, err := tx.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.OrderTypeEQ(payment.OrderTypeBalance),
			paymentorder.StatusIn(
				OrderStatusCompleted,
				OrderStatusPaid,
				OrderStatusRecharging,
				OrderStatusRefundRequested,
				OrderStatusRefunding,
				OrderStatusRefundFailed,
				OrderStatusPartiallyRefunded,
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query invoice-eligible payment orders: %w", err)
	}
	totalPaid := decimal.Zero
	for _, order := range orders {
		if PaymentOrderCurrency(order) != payment.DefaultPaymentCurrency {
			continue
		}
		paid := decimal.NewFromFloat(order.PayAmount).Sub(decimal.NewFromFloat(order.RefundAmount)).Round(2)
		if paid.IsPositive() {
			totalPaid = totalPaid.Add(paid)
		}
	}

	requests, err := tx.InvoiceRequest.Query().
		Where(invoicerequest.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query invoice request totals: %w", err)
	}
	invoicedAmount := decimal.Zero
	reservedAmount := decimal.Zero
	for _, request := range requests {
		amount := decimal.NewFromFloat(request.Amount).Round(2)
		switch request.Status {
		case InvoiceStatusIssued:
			invoicedAmount = invoicedAmount.Add(amount)
		case InvoiceStatusPending, InvoiceStatusIssuing, InvoiceStatusRequiresAuth:
			reservedAmount = reservedAmount.Add(amount)
		}
	}
	available := totalPaid.Sub(invoicedAmount).Sub(reservedAmount).Round(2)
	if available.IsNegative() {
		available = decimal.Zero
	}
	return &invoiceTotals{
		totalPaid:       totalPaid.Round(2),
		invoicedAmount:  invoicedAmount.Round(2),
		reservedAmount:  reservedAmount.Round(2),
		availableAmount: available,
	}, nil
}

func (s *PaymentService) issueInvoiceRequestWithSDK(request *dbent.InvoiceRequest, cfg *InvoiceConfig) invoiceSDKResult {
	result := invoiceSDKResult{
		status:   InvoiceStatusFailed,
		message:  "invoice sdk request failed",
		response: map[string]any{},
	}
	client := invoice.NewClient(cfg.SDKAppKey, cfg.SDKAppSecret, cfg.Debug)
	if strings.TrimSpace(cfg.SDKBaseURL) != "" {
		client.SetBaseURL(strings.TrimSpace(cfg.SDKBaseURL))
	}

	authResp, err := client.GetAuthorization(cfg.TaxpayerID, cfg.AccountType, cfg.Username, cfg.Password)
	if err != nil {
		result.message = err.Error()
		result.response["authorization_error"] = err.Error()
		return result
	}
	result.response["authorization"] = map[string]any{"code": authResp.Code, "msg": authResp.Msg}
	if authResp.Code != 200 {
		result.code = &authResp.Code
		result.message = authResp.Msg
		return result
	}

	faceResp, err := client.QueryFaceAuthState(cfg.TaxpayerID, map[string]string{"username": cfg.Username})
	if err != nil {
		result.message = err.Error()
		result.response["face_auth_error"] = err.Error()
		return result
	}
	result.response["face_auth"] = compactSDKResponse(faceResp)
	if faceResp.Code == 420 || faceResp.Code == 430 {
		result.code = &faceResp.Code
		result.status = InvoiceStatusRequiresAuth
		result.message = faceResp.Msg
		return result
	}
	if faceResp.Code != 200 {
		result.code = &faceResp.Code
		result.message = faceResp.Msg
		return result
	}

	params, items := buildSDKBlueTicketPayload(request, cfg)
	invoiceResp, err := client.BlueTicket(params, items)
	if err != nil {
		result.message = err.Error()
		result.response["blue_ticket_error"] = err.Error()
		return result
	}
	result.response["blue_ticket"] = map[string]any{
		"code": invoiceResp.Code,
		"msg":  invoiceResp.Msg,
		"fphm": invoiceResp.Fphm,
		"kprq": invoiceResp.Kprq,
	}
	result.code = &invoiceResp.Code
	result.message = invoiceResp.Msg
	if invoiceResp.Code == 420 || invoiceResp.Code == 430 {
		result.status = InvoiceStatusRequiresAuth
		return result
	}
	if invoiceResp.Code != 200 {
		return result
	}

	now := time.Now()
	result.status = InvoiceStatusIssued
	result.invoiceNo = invoiceResp.Fphm
	result.invoiceDate = invoiceResp.Kprq
	result.issuedAt = &now
	if invoiceResp.Fphm == "" {
		result.message = "invoice issued but invoice number is empty"
	}

	fileResp, err := client.GetVersionFile(cfg.TaxpayerID, invoiceResp.Fphm, "4", map[string]string{
		"kprq":     invoiceResp.Kprq,
		"username": cfg.Username,
	})
	if err != nil {
		result.response["version_file_error"] = err.Error()
		return result
	}
	result.response["version_file"] = compactSDKResponse(fileResp)
	if fileResp.Code == 200 {
		var files invoice.VersionFileResponse
		if err := invoice.ParseResponseData(fileResp, &files); err == nil {
			result.pdfURL = files.PdfUrl
			result.ofdURL = files.OfdUrl
			result.xmlURL = files.XmlUrl
		} else {
			result.response["version_file_parse_error"] = err.Error()
		}
	}
	return result
}

func buildSDKBlueTicketPayload(request *dbent.InvoiceRequest, cfg *InvoiceConfig) (map[string]string, []invoice.InvoiceItem) {
	amount := decimal.NewFromFloat(request.Amount).Round(2)
	taxRate := decimal.NewFromFloat(cfg.TaxRate)
	if taxRate.IsNegative() {
		taxRate = decimal.Zero
	}
	taxAmount := decimal.Zero
	if taxRate.IsPositive() {
		taxAmount = amount.Mul(taxRate).Div(decimal.NewFromInt(1).Add(taxRate)).Round(2)
	}
	netAmount := amount.Sub(taxAmount).Round(2)

	params := map[string]string{
		"fpqqlsh": fmt.Sprintf("sub2api-%d-%d", request.ID, time.Now().Unix()),
		"fplxdm":  cfg.TypeCode,
		"kplx":    "0",
		"xhdwsbh": cfg.TaxpayerID,
		"xhdwmc":  cfg.SellerName,
		"ghdwmc":  request.TitleName,
		"zsfs":    "0",
		"hjje":    netAmount.StringFixed(2),
		"hjse":    taxAmount.StringFixed(2),
		"jshj":    amount.StringFixed(2),
		"gmfyx":   request.Email,
	}
	optional := map[string]string{
		"xhdwdzdh": cfg.SellerAddressPhone,
		"xhdwyhzh": cfg.SellerBankAccount,
		"ghdwsbh":  request.TaxNo,
		"ghdwdzdh": request.AddressPhone,
		"ghdwyhzh": request.BankAccount,
		"bz":       request.Remark,
	}
	for key, value := range optional {
		if strings.TrimSpace(value) != "" {
			params[key] = strings.TrimSpace(value)
		}
	}

	items := []invoice.InvoiceItem{
		{
			Fphxz: "0",
			Spmc:  request.Content,
			Je:    amount.StringFixed(2),
			Sl:    taxRate.String(),
			Se:    taxAmount.StringFixed(2),
			Hsbz:  "1",
			Spbm:  cfg.TaxCode,
		},
	}
	return params, items
}

func compactSDKResponse(resp any) map[string]any {
	raw, err := json.Marshal(resp)
	if err != nil {
		return map[string]any{"marshal_error": err.Error()}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw": string(raw)}
	}
	delete(out, "token")
	delete(out, "Token")
	return out
}

func validateInvoiceConfigReady(cfg *InvoiceConfig) error {
	if cfg == nil || !cfg.Enabled {
		return infraerrors.Forbidden("INVOICE_DISABLED", "invoice self-service is disabled")
	}
	missing := make([]string, 0)
	required := map[string]string{
		"sdk_app_key":          cfg.SDKAppKey,
		"sdk_app_secret":       cfg.SDKAppSecret,
		"taxpayer_id":          cfg.TaxpayerID,
		"seller_name":          cfg.SellerName,
		"username":             cfg.Username,
		"password":             cfg.Password,
		"default_content":      cfg.DefaultContent,
		"tax_code":             cfg.TaxCode,
		"type_code":            cfg.TypeCode,
		"seller_address_phone": cfg.SellerAddressPhone,
		"seller_bank_account":  cfg.SellerBankAccount,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return infraerrors.BadRequest("INVOICE_CONFIG_INCOMPLETE", "invoice SDK configuration is incomplete").
			WithMetadata(map[string]string{"missing": strings.Join(missing, ",")})
	}
	return nil
}

func normalizeInvoiceProfileInput(input InvoiceProfileInput) (InvoiceProfileInput, error) {
	input.TitleType = strings.ToLower(strings.TrimSpace(input.TitleType))
	if input.TitleType == "" {
		input.TitleType = "personal"
	}
	if input.TitleType != "personal" && input.TitleType != "company" {
		return input, infraerrors.BadRequest("INVALID_INVOICE_TITLE_TYPE", "invoice title type must be personal or company")
	}
	input.Name = strings.TrimSpace(input.Name)
	input.TaxNo = strings.TrimSpace(input.TaxNo)
	input.AddressPhone = strings.TrimSpace(input.AddressPhone)
	input.BankAccount = strings.TrimSpace(input.BankAccount)
	input.Email = strings.TrimSpace(input.Email)
	if input.Name == "" {
		return input, infraerrors.BadRequest("INVALID_INVOICE_TITLE", "invoice title name is required")
	}
	if input.TitleType == "company" && input.TaxNo == "" {
		return input, infraerrors.BadRequest("INVALID_INVOICE_TAX_NO", "company invoice tax number is required")
	}
	if input.Email != "" && !isValidInvoiceEmail(input.Email) {
		return input, infraerrors.BadRequest("INVALID_INVOICE_EMAIL", "invoice email is invalid")
	}
	return input, nil
}

func isValidInvoiceEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}

func invoiceSummaryFromTotals(cfg *InvoiceConfig, totals *invoiceTotals) *UserInvoiceSummary {
	if totals == nil {
		totals = &invoiceTotals{}
	}
	minAmount := decimal.NewFromFloat(cfg.MinAmount).Round(2)
	if minAmount.IsNegative() {
		minAmount = decimal.Zero
	}
	return &UserInvoiceSummary{
		TotalPaid:       roundInvoiceFloat(totals.totalPaid.InexactFloat64()),
		InvoicedAmount:  roundInvoiceFloat(totals.invoicedAmount.InexactFloat64()),
		ReservedAmount:  roundInvoiceFloat(totals.reservedAmount.InexactFloat64()),
		AvailableAmount: roundInvoiceFloat(totals.availableAmount.InexactFloat64()),
		MinAmount:       roundInvoiceFloat(minAmount.InexactFloat64()),
		Currency:        cfg.Currency,
		Enabled:         cfg.Enabled,
	}
}

func invoiceProfileFromEnt(p *dbent.InvoiceProfile) UserInvoiceProfile {
	return UserInvoiceProfile{
		ID:           p.ID,
		TitleType:    p.TitleType,
		Name:         p.Name,
		TaxNo:        p.TaxNo,
		AddressPhone: p.AddressPhone,
		BankAccount:  p.BankAccount,
		Email:        p.Email,
		IsDefault:    p.IsDefault,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func invoiceRequestFromEnt(r *dbent.InvoiceRequest) UserInvoiceRequest {
	return UserInvoiceRequest{
		ID:              r.ID,
		ProfileID:       r.ProfileID,
		Status:          r.Status,
		Amount:          roundInvoiceFloat(r.Amount),
		PaidTotal:       roundInvoiceFloat(r.PaidTotal),
		InvoicedTotal:   roundInvoiceFloat(r.InvoicedTotal),
		ReservedTotal:   roundInvoiceFloat(r.ReservedTotal),
		AvailableAmount: roundInvoiceFloat(r.AvailableAmount),
		Currency:        r.Currency,
		TitleType:       r.TitleType,
		TitleName:       r.TitleName,
		TaxNo:           r.TaxNo,
		AddressPhone:    r.AddressPhone,
		BankAccount:     r.BankAccount,
		Email:           r.Email,
		Content:         r.Content,
		Remark:          r.Remark,
		SDKCode:         r.SdkCode,
		SDKMessage:      r.SdkMessage,
		InvoiceNo:       r.InvoiceNo,
		InvoiceDate:     r.InvoiceDate,
		PDFURL:          r.PdfURL,
		OFDURL:          r.OfdURL,
		XMLURL:          r.XMLURL,
		IssuedAt:        r.IssuedAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func roundInvoiceFloat(v float64) float64 {
	return math.Round(v*100) / 100
}
