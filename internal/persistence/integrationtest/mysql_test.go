//go:build integration

package integrationtest

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dujiao-next/internal/authz"
	"github.com/dujiao-next/internal/bootstrap/database/migrations"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	apicredentialdomain "github.com/dujiao-next/internal/modules/apicredential/domain"
	apicredentialstore "github.com/dujiao-next/internal/modules/apicredential/infrastructure/gormstore"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	dashboardstore "github.com/dujiao-next/internal/modules/dashboard/infrastructure/gormstore"
	adminapp "github.com/dujiao-next/internal/modules/identity/admin/application"
	adminstore "github.com/dujiao-next/internal/modules/identity/admin/infrastructure/gormstore"
	adminauthapp "github.com/dujiao-next/internal/modules/identity/adminauth/application"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	refundapp "github.com/dujiao-next/internal/modules/order/application/refund"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	paymentstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	resellerdomain "github.com/dujiao-next/internal/modules/reseller/domain"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	walletstore "github.com/dujiao-next/internal/modules/wallet/infrastructure/gormstore"
	"github.com/dujiao-next/internal/platform/database/gormdb"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestMySQL requires a dedicated empty schema. It leaves its fixtures in place
// so the test never needs to drop tables or a database.
func TestMySQL(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is empty")
	}

	require.NoError(t, gormdb.InitDB("mysql", dsn, gormdb.DBPoolConfig{
		MaxOpenConns: 8,
		MaxIdleConns: 4,
	}, "release"))
	db := gormdb.DB
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	tables, err := db.Migrator().GetTables()
	require.NoError(t, err)
	require.Empty(t, tables, "TEST_MYSQL_DSN must point to an empty test schema")

	require.NoError(t, migrations.AutoMigrate())
	testMySQLAdmin(t, db)
	var schemaChanges []string
	require.NoError(t, db.Callback().Raw().Before("gorm:raw").Register("test:record_schema_changes", func(tx *gorm.DB) {
		sql := tx.Statement.SQL.String()
		if strings.HasPrefix(sql, "ALTER TABLE ") || strings.HasPrefix(sql, "CREATE ") || strings.HasPrefix(sql, "DROP ") {
			schemaChanges = append(schemaChanges, sql)
		}
	}))
	require.NoError(t, migrations.AutoMigrate())
	testMySQLAdmin(t, db)
	require.Empty(t, schemaChanges, "restarting with an initialized schema must not run DDL")
	require.NoError(t, db.Callback().Raw().Remove("test:record_schema_changes"))

	t.Run("reseller active key", func(t *testing.T) { testMySQLResellerActiveKey(t, db) })
	t.Run("search and reports", func(t *testing.T) { testMySQLQueries(t, db) })
	t.Run("payment and refund concurrency", func(t *testing.T) { testMySQLPayments(t, db) })
}

func testMySQLAdmin(t *testing.T, db *gorm.DB) {
	t.Helper()
	repository := adminstore.New(db)
	_, err := adminapp.InitDefaultAdmin(repository, "mysql-test", "test-only-admin-password")
	require.NoError(t, err)
	auth := adminauthapp.NewService(&config.Config{
		JWT: config.JWTConfig{SecretKey: "mysql-test-only-secret", ExpireHours: 1},
	}, repository)
	login, err := auth.Login("mysql-test", "test-only-admin-password")
	require.NoError(t, err)
	require.NotEmpty(t, login.Token)

	policies, err := authz.NewService(db)
	require.NoError(t, err)
	require.NoError(t, policies.BootstrapBuiltinRoles())
	require.NoError(t, policies.SetAdminRoles(123, []string{"operations"}))
	allowed, err := policies.EnforceAdmin(123, "/api/v1/admin/products", "GET")
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = policies.EnforceAdmin(123, "/api/v1/admin/admins", "DELETE")
	require.NoError(t, err)
	require.False(t, allowed)
}

func testMySQLResellerActiveKey(t *testing.T, db *gorm.DB) {
	user := userdomain.User{Email: "reseller@example.test", PasswordHash: "test"}
	require.NoError(t, db.Create(&user).Error)
	profile := resellerdomain.Profile{UserID: user.ID}
	require.NoError(t, db.Create(&profile).Error)
	create := func() *resellerdomain.Domain {
		return &resellerdomain.Domain{ResellerID: profile.ID, Domain: "reseller.example.test", Type: "custom"}
	}
	first := create()
	require.NoError(t, db.Create(first).Error)
	require.ErrorContains(t, db.Create(create()).Error, "1062")
	require.NoError(t, db.Model(first).Update("deleted_at", time.Now().UTC()).Error)
	second := create()
	require.NoError(t, db.Create(second).Error)
	require.NoError(t, db.Model(second).Update("deleted_at", time.Now().UTC()).Error)
	require.NoError(t, db.Create(create()).Error)
	require.ErrorContains(t, db.Create(create()).Error, "1062")

	credential := apicredentialdomain.ApiCredential{UserID: user.ID, ApiKey: "ExactKey"}
	require.NoError(t, db.Create(&credential).Error)
	store := apicredentialstore.New(db)
	found, err := store.GetByApiKey("ExactKey")
	require.NoError(t, err)
	require.NotNil(t, found)
	found, err = store.GetByApiKey("exactkey")
	require.NoError(t, err)
	require.Nil(t, found)
}

func testMySQLQueries(t *testing.T, db *gorm.DB) {
	t.Helper()
	category := categorydomain.Category{Slug: "mysql-category", NameJSON: jsonmap.JSON{"en-US": "Test category"}}
	require.NoError(t, db.Create(&category).Error)
	product := productdomain.Product{
		CategoryID:      category.ID,
		Slug:            "mysql-product",
		TitleJSON:       jsonmap.JSON{"zh-CN": "商品", "en-US": "Voucher"},
		IsActive:        true,
		WholesalePrices: productdomain.WholesalePriceTiers{{MinQuantity: 2, UnitPrice: money.FromDecimal(decimal.NewFromInt(9))}},
	}
	require.NoError(t, db.Create(&product).Error)
	hasWholesale := true
	products, _, err := productstore.NewProductStore(db).List(productcontract.ListFilter{
		Search:             "voucher",
		HasWholesalePrices: &hasWholesale,
	})
	require.NoError(t, err)
	require.Len(t, products, 1)
	require.Equal(t, product.ID, products[0].ID)

	channel := paymentdomain.PaymentChannel{Name: "MySQL test"}
	require.NoError(t, db.Create(&channel).Error)
	loc := time.FixedZone("UTC+08:00", 8*3600)
	startAt := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	for i, hours := range []int{-1, 1, 25, 48} {
		order := orderdomain.Order{
			OrderNo:   []string{"BEFORE", "FIRST", "SECOND", "AFTER"}[i],
			Status:    constants.OrderStatusPendingPayment,
			CreatedAt: startAt.Add(time.Duration(hours) * time.Hour).UTC(),
		}
		require.NoError(t, db.Create(&order).Error)
	}
	rows, err := dashboardstore.New(db).GetOrderTrends(startAt, startAt.AddDate(0, 0, 2))
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "2026-01-01", rows[0].Day)
	require.Equal(t, "2026-01-02", rows[1].Day)

	payment := paymentdomain.Payment{ChannelID: channel.ID, ProviderPayload: jsonmap.JSON{"display_channel_type": "test"}}
	require.NoError(t, db.Create(&payment).Error)
	payments, _, err := paymentstore.New(db, "test-only-guest-credential-secret").ListAdmin(paymentcontract.ListFilter{Lightweight: true})
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Equal(t, "test", payments[0].DisplayChannelType)

	dst, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	dstStart := time.Date(2026, 11, 1, 0, 0, 0, 0, dst)
	order := orderdomain.Order{
		OrderNo: "MYSQL-DST", Status: constants.OrderStatusPendingPayment,
		CreatedAt: time.Date(2026, 11, 2, 4, 30, 0, 0, time.UTC),
	}
	require.NoError(t, db.Create(&order).Error)
	rows, err = dashboardstore.New(db).GetOrderTrends(dstStart, dstStart.AddDate(0, 0, 2))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "2026-11-01", rows[0].Day)
}

func testMySQLPayments(t *testing.T, db *gorm.DB) {
	t.Helper()
	const guestSecret = "test-only-guest-credential-secret"
	user := userdomain.User{Email: "payments@example.test", PasswordHash: "test"}
	require.NoError(t, db.Create(&user).Error)
	channel := paymentdomain.PaymentChannel{Name: "MySQL concurrency test"}
	require.NoError(t, db.Create(&channel).Error)
	amount := money.FromDecimal(decimal.NewFromInt(100))

	orders := orderstore.New(db, guestSecret)
	payments := paymentstore.New(db, guestSecret)
	walletRepo := walletstore.New(db)
	wallets := walletapp.NewService(walletapp.Options{Repository: walletRepo, Transactions: walletRepo})
	service := paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{
		OrderStore:    orders,
		PaymentStore:  payments,
		WalletService: wallets,
	})
	notify := func(orderNo string, paid money.Amount) uint {
		t.Helper()
		order := orderdomain.Order{
			OrderNo:     orderNo,
			UserID:      user.ID,
			Currency:    "CNY",
			Status:      constants.OrderStatusPendingPayment,
			TotalAmount: amount,
		}
		require.NoError(t, db.Create(&order).Error)
		payment := paymentdomain.Payment{
			OrderID: order.ID, ChannelID: channel.ID, Currency: "CNY",
			Status: constants.PaymentStatusPending, Amount: paid,
		}
		require.NoError(t, db.Create(&payment).Error)
		errs := runMySQLConcurrently(func() error {
			_, err := service.HandleCallback(paymentapp.PaymentCallbackInput{
				PaymentID: payment.ID, OrderNo: order.OrderNo, ChannelID: channel.ID,
				Status: constants.PaymentStatusSuccess, Amount: paid, Currency: payment.Currency,
			})
			return err
		})
		for _, err := range errs {
			require.NoError(t, err)
		}
		return order.ID
	}

	partial := money.FromDecimal(decimal.NewFromInt(50))
	underpaidID := notify("MYSQL-UNDERPAID", partial)
	underpaid, err := orders.GetByID(underpaidID)
	require.NoError(t, err)
	require.Equal(t, constants.OrderStatusPendingPayment, underpaid.Status)
	account, err := wallets.GetAccount(user.ID)
	require.NoError(t, err)
	require.True(t, account.Balance.Decimal.Equal(partial.Decimal))

	paidID := notify("MYSQL-PAID", amount)
	refunds := refundapp.New(orders, nil, nil, nil, wallets, payments)
	successes := 0
	for _, err := range runMySQLConcurrently(func() error {
		_, _, _, err := refunds.AdminRefundToWallet(refundapp.AdminRefundToWalletInput{OrderID: paidID, Amount: amount})
		return err
	}) {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, walletcontract.ErrRefundExceeded)
		}
	}
	require.Equal(t, 1, successes)
	account, err = wallets.GetAccount(user.ID)
	require.NoError(t, err)
	require.True(t, account.Balance.Decimal.Equal(partial.Decimal.Add(amount.Decimal)))
	var ledgerCount int64
	require.NoError(t, db.Model(&walletdomain.Transaction{}).Where("user_id = ?", user.ID).Count(&ledgerCount).Error)
	require.EqualValues(t, 2, ledgerCount)
}

func runMySQLConcurrently(run func() error) [2]error {
	var errs [2]error
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range errs {
		wg.Go(func() {
			<-start
			errs[i] = run()
		})
	}
	close(start)
	wg.Wait()
	return errs
}
