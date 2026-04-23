package service

import "time"

type InviteLink struct {
	ID              int64
	Code            string
	CreatedByUserID int64
	CreatorRole     string
	OwnerSalesID    *int64
	Status          string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type InviteRewardLedger struct {
	ID             int64
	InviterUserID  int64
	InviteeUserID  int64
	TriggerOrderID int64
	RewardType     string
	RewardAmount   float64
	Status         string
	Reason         string
	CreatedAt      time.Time
	ConfirmedAt    *time.Time
	ReversedAt     *time.Time
}

type InvoiceProfile struct {
	ID          int64
	UserID      int64
	Title       string
	TaxNo       string
	Email       string
	Phone       string
	Address     string
	BankName    string
	BankAccount string
	InvoiceType string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type InvoiceRequest struct {
	ID                int64
	UserID            int64
	OrderID           int64
	ProfileID         int64
	Status            string
	Provider          string
	ProviderRequestID string
	ProviderInvoiceID string
	FailReason        string
	RetryCount        int
	RequestedAt       time.Time
	IssuedAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type InvoiceDocument struct {
	ID                int64
	InvoiceRequestID  int64
	InvoiceNo         string
	InvoiceCode       string
	FileURL           string
	FileType          string
	RawPayloadSummary map[string]any
	CreatedAt         time.Time
}
