package gormstore

import (
	"errors"
	"fmt"

	resellercontract "github.com/dujiao-next/internal/modules/reseller/contract"
	resellerdomain "github.com/dujiao-next/internal/modules/reseller/domain"
	"gorm.io/gorm"
)

// Store is the single GORM implementation of all reseller persistence ports.
type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Migrate creates the reseller tables and enforces uniqueness among active rows.
func Migrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("reseller migration database is nil")
	}
	if err := db.AutoMigrate(
		&resellerdomain.Profile{},
		&resellerdomain.Domain{},
		&resellerdomain.SiteConfig{},
		&resellerdomain.ProductSetting{},
		&resellerdomain.OrderSnapshot{},
		&resellerdomain.LedgerEntry{},
		&resellerdomain.WithdrawRequest{},
		&resellerdomain.BalanceAccount{},
		&resellerdomain.RelatedAccount{},
	); err != nil {
		return err
	}
	indexes := []struct {
		table   string
		name    string
		columns string
	}{
		{(resellerdomain.Domain{}).TableName(), "idx_reseller_domains_active_domain", "domain"},
		{(resellerdomain.SiteConfig{}).TableName(), "idx_reseller_site_configs_active_reseller", "reseller_id"},
		{(resellerdomain.ProductSetting{}).TableName(), "idx_reseller_product_settings_active_scope", "reseller_id, product_id, sku_id"},
		{(resellerdomain.BalanceAccount{}).TableName(), "idx_reseller_balance_accounts_active_currency", "reseller_id, currency"},
		{(resellerdomain.RelatedAccount{}).TableName(), "idx_reseller_related_accounts_active_user", "reseller_id, user_id"},
	}
	for _, index := range indexes {
		if err := createActiveUniqueIndex(db, index.table, index.name, index.columns); err != nil {
			return err
		}
	}
	return nil
}

// Table, index and column names are fixed schema identifiers from Migrate.
func createActiveUniqueIndex(db *gorm.DB, table, name, columns string) error {
	if db.Dialector.Name() != "mysql" {
		return db.Exec(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s(%s) WHERE deleted_at IS NULL", name, table, columns)).Error
	}

	migrator := db.Migrator()
	if !migrator.HasColumn(table, "active_marker") {
		// MySQL unique indexes allow multiple NULLs, so deleted rows can share a business key.
		sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN active_marker TINYINT GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN 1 END) VIRTUAL", table)
		if err := db.Exec(sql).Error; err != nil && !migrator.HasColumn(table, "active_marker") {
			return err
		}
	}
	if migrator.HasIndex(table, name) {
		return nil
	}
	if err := db.Exec(fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s(%s, active_marker)", name, table, columns)).Error; err != nil && !migrator.HasIndex(table, name) {
		return err
	}
	return nil
}

// BindTx returns the same reseller store surface bound to a caller-owned transaction.
func (s *Store) BindTx(tx *gorm.DB) *Store {
	if tx == nil {
		return s
	}
	return New(tx)
}

func (s *Store) transaction(run func(*Store) error) error {
	if s == nil || s.db == nil || run == nil {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return run(New(tx))
	})
}

func (s *Store) WithinManagementTransaction(run func(resellercontract.ManagementStore) error) error {
	return s.transaction(func(tx *Store) error { return run(tx) })
}

func (s *Store) WithinProductSettingTransaction(run func(resellercontract.ProductSettingStore) error) error {
	return s.transaction(func(tx *Store) error { return run(tx) })
}

func (s *Store) WithinWithdrawTransaction(run func(resellercontract.AccountingWithdrawStore) error) error {
	return s.transaction(func(tx *Store) error { return run(tx) })
}

func (s *Store) WithinLedgerTransaction(run func(resellercontract.AccountingLedgerStore) error) error {
	return s.transaction(func(tx *Store) error { return run(tx) })
}

func applyPagination(query *gorm.DB, page, pageSize int) *gorm.DB {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return query.Offset((page - 1) * pageSize).Limit(pageSize)
}
