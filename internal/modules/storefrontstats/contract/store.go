package contract

import (
	"context"

	"github.com/dujiao-next/internal/modules/storefrontstats/domain"
)

type Store interface {
	RecordHomeVisit(ctx context.Context, resellerID uint, visitorKey, day string) error
	HomeVisitorStats(ctx context.Context, day string) (domain.HomeVisitorStats, error)
}
