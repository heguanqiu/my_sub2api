package service

import (
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func ParseOrderListDateRange(startDate, endDate, userTZ string) (*time.Time, *time.Time, error) {
	start, err := parseOrderListStartDate(startDate, userTZ)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseOrderListEndDate(endDate, userTZ)
	if err != nil {
		return nil, nil, err
	}
	if start != nil && end != nil && !end.After(*start) {
		return nil, nil, infraerrors.BadRequest("INVALID_DATE_RANGE", "end_date must be after start_date")
	}
	return start, end, nil
}

func parseOrderListStartDate(raw string, userTZ string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed, nil
	}
	parsed, err := timezone.ParseInUserLocation("2006-01-02", raw, userTZ)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_START_DATE", "invalid start_date format, use YYYY-MM-DD")
	}
	return &parsed, nil
}

func parseOrderListEndDate(raw string, userTZ string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed, nil
	}
	parsed, err := timezone.ParseInUserLocation("2006-01-02", raw, userTZ)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_END_DATE", "invalid end_date format, use YYYY-MM-DD")
	}
	end := parsed.AddDate(0, 0, 1)
	return &end, nil
}
