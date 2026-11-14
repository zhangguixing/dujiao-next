package gormstore

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dujiao-next/internal/modules/storefrontstats/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRecordHomeVisitDeduplicatesAndIsolatesStorefronts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "visitors.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&domain.HomeVisitor{}); err != nil {
		t.Fatal(err)
	}

	store := New(db)
	ctx := context.Background()
	const firstDay = "2026-11-14"
	const nextDay = "2026-11-15"

	var requests sync.WaitGroup
	for range 8 {
		requests.Go(func() {
			if err := store.RecordHomeVisit(ctx, 0, "browser-a", firstDay); err != nil {
				t.Errorf("concurrent repeated visit: %v", err)
			}
		})
	}
	requests.Wait()

	for _, visit := range []struct {
		siteID uint
		key    string
		day    string
		want   domain.HomeVisitorStats
	}{
		{0, "browser-a", firstDay, domain.HomeVisitorStats{Today: 1, Total: 1}},
		{7, "browser-a", firstDay, domain.HomeVisitorStats{Today: 2, Total: 2}},
		{0, "browser-b", firstDay, domain.HomeVisitorStats{Today: 3, Total: 3}},
		{0, "browser-a", nextDay, domain.HomeVisitorStats{Today: 1, Total: 3}},
		{0, "browser-a", firstDay, domain.HomeVisitorStats{Today: 2, Total: 3}},
		{0, "browser-b", nextDay, domain.HomeVisitorStats{Today: 2, Total: 3}},
	} {
		if err := store.RecordHomeVisit(ctx, visit.siteID, visit.key, visit.day); err != nil {
			t.Fatal(err)
		}
		stats, err := store.HomeVisitorStats(ctx, visit.day)
		if err != nil || stats != visit.want {
			t.Fatalf("visit %+v = %+v, %v", visit, stats, err)
		}
	}
}
