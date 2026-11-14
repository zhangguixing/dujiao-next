package gormstore

import (
	"context"

	"github.com/dujiao-next/internal/modules/storefrontstats/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) RecordHomeVisit(ctx context.Context, resellerID uint, visitorKey, day string) error {
	visit := domain.HomeVisitor{ResellerID: resellerID, VisitorKey: visitorKey, VisitDay: day}
	// Older requests that finish after midnight must not overwrite a newer visit.
	currentDay := clause.Column{Table: clause.CurrentTable, Name: "visit_day"}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "reseller_id"}, {Name: "visitor_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"visit_day": gorm.Expr("CASE WHEN ? < ? THEN ? ELSE ? END", currentDay, day, day, currentDay),
		}),
	}).Create(&visit).Error
}

func (s *Store) HomeVisitorStats(ctx context.Context, day string) (domain.HomeVisitorStats, error) {
	var stats domain.HomeVisitorStats
	err := s.db.WithContext(ctx).Model(&domain.HomeVisitor{}).
		Select("COUNT(CASE WHEN visit_day = ? THEN 1 END) AS today, COUNT(*) AS total", day).
		Scan(&stats).Error
	return stats, err
}
