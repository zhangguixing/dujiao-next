package integrationtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	memberlevelapp "github.com/dujiao-next/internal/modules/memberlevel/application"
	memberlevelcontract "github.com/dujiao-next/internal/modules/memberlevel/contract"
	memberleveldomain "github.com/dujiao-next/internal/modules/memberlevel/domain"
	"github.com/dujiao-next/internal/modules/memberlevel/infrastructure/gormstore"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func newMemberLevelServiceForTest(t *testing.T) (*memberlevelapp.Service, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:member_level_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&userdomain.User{}, &memberleveldomain.MemberLevel{}, &memberleveldomain.MemberLevelPrice{}, &orderdomain.Order{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	levelRepo := gormstore.NewLevelStore(db)
	priceRepo := gormstore.NewPriceStore(db)
	userRepo := gormstore.NewUserStore(db)
	return memberlevelapp.NewService(levelRepo, priceRepo, userRepo, orderstore.New(db, "member-level-test")), db
}

func createPaidOrderFixture(t *testing.T, db *gorm.DB, userID uint, orderNo, amount string) orderdomain.Order {
	t.Helper()
	paidAt := time.Now()
	order := orderdomain.Order{
		OrderNo:     orderNo,
		UserID:      userID,
		Status:      constants.OrderStatusPaid,
		Currency:    "CNY",
		PaidAt:      &paidAt,
		TotalAmount: money.FromDecimal(decimal.RequireFromString(amount)),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create paid order: %v", err)
	}
	return order
}

func createMemberLevelFixture(
	t *testing.T,
	db *gorm.DB,
	slug string,
	sortOrder int,
	spendThreshold string,
	isDefault bool,
) memberleveldomain.MemberLevel {
	t.Helper()

	level := memberleveldomain.MemberLevel{
		NameJSON: jsonmap.JSON{
			"zh-CN": slug,
		},
		Slug:              slug,
		DiscountRate:      money.FromDecimal(decimal.NewFromInt(100)),
		RechargeThreshold: money.FromDecimal(decimal.Zero),
		SpendThreshold:    money.FromDecimal(decimal.RequireFromString(spendThreshold)),
		IsDefault:         isDefault,
		SortOrder:         sortOrder,
		IsActive:          true,
	}
	if err := db.Create(&level).Error; err != nil {
		t.Fatalf("create member level fixture failed: %v", err)
	}
	return level
}

func createUserFixture(t *testing.T, db *gorm.DB, email string, memberLevelID uint) userdomain.User {
	t.Helper()

	user := userdomain.User{
		Email:          email,
		PasswordHash:   "test-hash",
		Status:         "active",
		MemberLevelID:  memberLevelID,
		TotalRecharged: money.FromDecimal(decimal.Zero),
		TotalSpent:     money.FromDecimal(decimal.Zero),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user fixture failed: %v", err)
	}
	return user
}

func TestMemberLevelServiceOnOrderPaidDoesNotMoveToEqualSortOrder(t *testing.T) {
	svc, db := newMemberLevelServiceForTest(t)
	defaultLevel := createMemberLevelFixture(t, db, "default", 0, "0", true)
	_ = createMemberLevelFixture(t, db, "vip", 0, "0.01", false)
	user := createUserFixture(t, db, "equal-sort@example.com", defaultLevel.ID)
	createPaidOrderFixture(t, db, user.ID, "equal-sort", "0.01")

	if err := svc.OnOrderPaid(user.ID, decimal.RequireFromString("0.01")); err != nil {
		t.Fatalf("OnOrderPaid failed: %v", err)
	}

	var updated userdomain.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("fetch updated user failed: %v", err)
	}
	if updated.MemberLevelID != defaultLevel.ID {
		t.Fatalf("expected keep member_level_id=%d, got %d", defaultLevel.ID, updated.MemberLevelID)
	}
	if !updated.TotalSpent.Decimal.Equal(decimal.RequireFromString("0.01")) {
		t.Fatalf("expected total_spent=0.01, got %s", updated.TotalSpent.Decimal.String())
	}
}

func TestMemberLevelServiceOnOrderPaidKeepsHigherLevel(t *testing.T) {
	svc, db := newMemberLevelServiceForTest(t)
	highLevel := createMemberLevelFixture(t, db, "high", 100, "0", true)
	_ = createMemberLevelFixture(t, db, "low", 10, "0.01", false)
	user := createUserFixture(t, db, "no-downgrade@example.com", highLevel.ID)
	createPaidOrderFixture(t, db, user.ID, "no-downgrade", "50.00")

	if err := svc.OnOrderPaid(user.ID, decimal.RequireFromString("50")); err != nil {
		t.Fatalf("OnOrderPaid failed: %v", err)
	}

	var updated userdomain.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("fetch updated user failed: %v", err)
	}
	if updated.MemberLevelID != highLevel.ID {
		t.Fatalf("expected keep member_level_id=%d, got %d", highLevel.ID, updated.MemberLevelID)
	}
}

func TestMemberLevelServiceOnOrderPaidUpgradesToHigherSortOrder(t *testing.T) {
	svc, db := newMemberLevelServiceForTest(t)
	defaultLevel := createMemberLevelFixture(t, db, "default2", 0, "0", true)
	goldLevel := createMemberLevelFixture(t, db, "gold", 20, "0.01", false)
	user := createUserFixture(t, db, "higher-sort@example.com", defaultLevel.ID)
	createPaidOrderFixture(t, db, user.ID, "higher-sort", "0.01")

	if err := svc.OnOrderPaid(user.ID, decimal.RequireFromString("0.01")); err != nil {
		t.Fatalf("OnOrderPaid failed: %v", err)
	}

	var updated userdomain.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("fetch updated user failed: %v", err)
	}
	if updated.MemberLevelID != goldLevel.ID {
		t.Fatalf("expected member_level_id=%d, got %d", goldLevel.ID, updated.MemberLevelID)
	}
}

func TestMemberLevelUpgradeUsesNetPaidOrders(t *testing.T) {
	svc, db := newMemberLevelServiceForTest(t)
	defaultLevel := createMemberLevelFixture(t, db, "refund-default", 0, "0", true)
	nextLevel := createMemberLevelFixture(t, db, "refund-next", 10, "75.00", false)
	user := createUserFixture(t, db, "refund@example.com", defaultLevel.ID)
	full := createPaidOrderFixture(t, db, user.ID, "full-refund", "100.00")
	markedRefunded := createPaidOrderFixture(t, db, user.ID, "marked-refunded", "100.00")
	partial := createPaidOrderFixture(t, db, user.ID, "partial-refund", "100.00")
	child := createPaidOrderFixture(t, db, user.ID, "refunded-child", "40.00")
	otherUser := createUserFixture(t, db, "other-customer@example.com", defaultLevel.ID)
	createPaidOrderFixture(t, db, otherUser.ID, "other-customer-order", "100.00")
	pending := createPaidOrderFixture(t, db, user.ID, "unpaid", "100.00")
	deleted := createPaidOrderFixture(t, db, user.ID, "deleted", "100.00")

	for _, fixture := range []struct {
		id     uint
		fields map[string]any
	}{
		{full.ID, map[string]any{"status": constants.OrderStatusRefunded, "refunded_amount": "100.00"}},
		{markedRefunded.ID, map[string]any{"status": constants.OrderStatusRefunded}},
		{partial.ID, map[string]any{"status": constants.OrderStatusPartiallyRefunded, "refunded_amount": "30.20"}},
		{child.ID, map[string]any{"parent_id": partial.ID, "refunded_amount": "30.20"}},
		{pending.ID, map[string]any{"status": constants.OrderStatusPendingPayment, "paid_at": nil}},
		{deleted.ID, map[string]any{"deleted_at": time.Now()}},
	} {
		if err := db.Model(&orderdomain.Order{}).Where("id = ?", fixture.id).Updates(fixture.fields).Error; err != nil {
			t.Fatalf("update order fixture: %v", err)
		}
	}
	if err := db.Model(&userdomain.User{}).Where("id = ?", user.ID).Update("total_spent", "200.00").Error; err != nil {
		t.Fatalf("set historical cumulative spend: %v", err)
	}
	spending := orderstore.New(db, "member-level-test")
	if net, err := spending.NetPaidTotalByUser(user.ID); err != nil || net.String() != "69.80" {
		t.Fatalf("expected net spending 69.80, got %s, %v", net, err)
	}
	if err := svc.CheckAndUpgrade(user.ID); err != nil {
		t.Fatalf("check historical spending: %v", err)
	}
	if err := db.First(&user, user.ID).Error; err != nil || user.MemberLevelID != defaultLevel.ID {
		t.Fatalf("expected refunded spending to leave user at level %d, got %d, %v", defaultLevel.ID, user.MemberLevelID, err)
	}

	createPaidOrderFixture(t, db, user.ID, "reaches-net-threshold", "5.20")
	if err := svc.OnOrderPaid(user.ID, decimal.RequireFromString("5.20")); err != nil {
		t.Fatalf("check spending after payment: %v", err)
	}
	if err := db.First(&user, user.ID).Error; err != nil || user.MemberLevelID != nextLevel.ID {
		t.Fatalf("expected upgrade at exactly 75.00 net to level %d, got %d, %v", nextLevel.ID, user.MemberLevelID, err)
	}
}

type concurrentUserRepository struct {
	base                     memberlevelcontract.UserRepository
	afterFirstSpendIncrement func()
	spendIncrementCalled     bool
}

func (r *concurrentUserRepository) GetByID(id uint) (*userdomain.User, error) {
	return r.base.GetByID(id)
}

func (r *concurrentUserRepository) Update(user *userdomain.User) error {
	return r.base.Update(user)
}

func (r *concurrentUserRepository) IncrementTotalRecharged(userID uint, amount decimal.Decimal) error {
	return r.base.IncrementTotalRecharged(userID, amount)
}

func (r *concurrentUserRepository) IncrementTotalSpent(userID uint, amount decimal.Decimal) error {
	if err := r.base.IncrementTotalSpent(userID, amount); err != nil {
		return err
	}
	if !r.spendIncrementCalled && r.afterFirstSpendIncrement != nil {
		r.spendIncrementCalled = true
		r.afterFirstSpendIncrement()
	}
	return nil
}

func (r *concurrentUserRepository) UpdateMemberLevelIfCurrent(userID, currentLevelID, nextLevelID uint) (int64, error) {
	return r.base.UpdateMemberLevelIfCurrent(userID, currentLevelID, nextLevelID)
}

func (r *concurrentUserRepository) AssignDefaultMemberLevel(defaultLevelID uint) (int64, error) {
	return r.base.AssignDefaultMemberLevel(defaultLevelID)
}

func TestMemberLevelServiceOnOrderPaidDoesNotOverwriteConcurrentHigherLevel(t *testing.T) {
	_, db := newMemberLevelServiceForTest(t)
	defaultLevel := createMemberLevelFixture(t, db, "race-default", 0, "0", true)
	goldLevel := createMemberLevelFixture(t, db, "race-gold", 20, "1.00", false)
	highLevel := createMemberLevelFixture(t, db, "race-high", 100, "1000.00", false)
	user := createUserFixture(t, db, "race-no-downgrade@example.com", defaultLevel.ID)
	createPaidOrderFixture(t, db, user.ID, "race-no-downgrade", "1.00")

	baseUserRepo := gormstore.NewUserStore(db)
	raceRepo := &concurrentUserRepository{
		base: baseUserRepo,
		afterFirstSpendIncrement: func() {
			if err := db.Model(&userdomain.User{}).Where("id = ?", user.ID).Update("member_level_id", highLevel.ID).Error; err != nil {
				t.Fatalf("simulate concurrent higher level update failed: %v", err)
			}
		},
	}
	svc := memberlevelapp.NewService(
		gormstore.NewLevelStore(db),
		gormstore.NewPriceStore(db),
		raceRepo,
		orderstore.New(db, "member-level-test"),
	)

	if err := svc.OnOrderPaid(user.ID, decimal.RequireFromString("1.00")); err != nil {
		t.Fatalf("OnOrderPaid failed: %v", err)
	}

	var updated userdomain.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("fetch updated user failed: %v", err)
	}
	if updated.MemberLevelID == goldLevel.ID {
		t.Fatalf("expected concurrent high level to be preserved, got auto-upgraded lower level %d", updated.MemberLevelID)
	}
	if updated.MemberLevelID != highLevel.ID {
		t.Fatalf("expected member_level_id=%d, got %d", highLevel.ID, updated.MemberLevelID)
	}
	if !updated.TotalSpent.Decimal.Equal(decimal.RequireFromString("1.00")) {
		t.Fatalf("expected total_spent=1.00, got %s", updated.TotalSpent.Decimal.String())
	}
}

func TestMemberLevelServiceCreateLevelRejectsActiveSortOrderConflict(t *testing.T) {
	svc, db := newMemberLevelServiceForTest(t)
	_ = createMemberLevelFixture(t, db, "sort-existing", 10, "0", true)

	err := svc.CreateLevel(&memberleveldomain.MemberLevel{
		NameJSON:          jsonmap.JSON{"zh-CN": "sort-conflict"},
		Slug:              "sort-conflict",
		DiscountRate:      money.FromDecimal(decimal.NewFromInt(100)),
		RechargeThreshold: money.FromDecimal(decimal.Zero),
		SpendThreshold:    money.FromDecimal(decimal.NewFromInt(1)),
		IsDefault:         false,
		SortOrder:         10,
		IsActive:          true,
	})
	if err == nil {
		t.Fatalf("expected active sort_order conflict to be rejected")
	}
}

func TestMemberLevelServiceUpdateLevelRejectsActiveSortOrderConflict(t *testing.T) {
	svc, db := newMemberLevelServiceForTest(t)
	_ = createMemberLevelFixture(t, db, "sort-update-existing", 10, "0", true)
	target := createMemberLevelFixture(t, db, "sort-update-target", 20, "1.00", false)

	target.SortOrder = 10
	err := svc.UpdateLevel(&target)
	if err == nil {
		t.Fatalf("expected active sort_order conflict to be rejected")
	}
}
