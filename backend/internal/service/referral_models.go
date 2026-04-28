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
