package gormdb

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite" // 纯 Go SQLite 驱动（基于 modernc.org/sqlite）
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// DBPoolConfig 数据库连接池配置
type DBPoolConfig struct {
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeSeconds int
	ConnMaxIdleTimeSeconds int
}

// InitDB 初始化数据库连接
func InitDB(driver, dsn string, pool DBPoolConfig, mode string) error {
	normalized := strings.ToLower(strings.TrimSpace(driver))
	var dialector gorm.Dialector
	var mysqlDB *sql.DB
	switch normalized {
	case "", "sqlite":
		// glebarez/sqlite 是基于 modernc.org/sqlite 的纯 Go 驱动
		// 追加 PRAGMA 参数避免并发写入时 busy-spin 导致 CPU 飙升
		dialector = sqlite.Open(appendSQLitePragmas(dsn))
	case "postgres", "postgresql":
		dialector = postgres.Open(dsn)
	case "mysql":
		var err error
		mysqlDB, err = openMySQL(dsn)
		if err != nil {
			return err
		}
		dialector = newMySQLDialector(mysqlDB)
	default:
		return fmt.Errorf("unsupported database driver: %s", driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:  newGORMLogger(mode, nil),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		if mysqlDB != nil {
			_ = mysqlDB.Close()
		}
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	applyDBPool(sqlDB, pool)

	if mysqlDB != nil {
		var collation string
		if err := db.Raw("SELECT @@collation_database").Row().Scan(&collation); err != nil {
			_ = sqlDB.Close()
			return err
		}
		if collation != mysqlCollation {
			_ = sqlDB.Close()
			return fmt.Errorf("MySQL database must use collation %s (got %s)", mysqlCollation, collation)
		}
	}

	// SQLite 额外执行 PRAGMA 确保 WAL 模式生效
	if normalized == "" || normalized == "sqlite" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA busy_timeout=5000")
		db.Exec("PRAGMA synchronous=NORMAL")
	}
	DB = db
	return nil
}

func newMySQLDialector(conn *sql.DB) gorm.Dialector {
	return mysqlDialector{mysql.Dialector{Config: &mysql.Config{Conn: conn}}}
}

const mysqlCollation = "utf8mb4_0900_bin"

type mysqlDialector struct{ mysql.Dialector }

func (d mysqlDialector) Migrator(db *gorm.DB) gorm.Migrator {
	m := d.Dialector.Migrator(db).(mysql.Migrator)
	m.Migrator.Dialector = d
	return mysqlMigrator{m}
}

type mysqlMigrator struct{ mysql.Migrator }

func (m mysqlMigrator) FullDataTypeOf(field *schema.Field) clause.Expr {
	// MySQL requires a parenthesized expression for a TEXT default.
	if field.DataType == "text" && field.HasDefaultValue && field.DefaultValue == "" {
		withExpression := *field
		withExpression.DefaultValue = "('')"
		withExpression.DefaultValueInterface = nil
		return m.Migrator.FullDataTypeOf(&withExpression)
	}
	return m.Migrator.FullDataTypeOf(field)
}

func (m mysqlMigrator) MigrateColumn(model interface{}, field *schema.Field, column gorm.ColumnType) error {
	// MySQL reports scaled decimal literals and canonicalized TEXT expressions.
	// Compare their values so an unchanged schema does not get ALTERed on restart.
	if reported, ok := column.(migrator.ColumnType); ok && reported.DefaultValueValue.Valid && field.HasDefaultValue {
		current := reported.DefaultValueValue.String
		sameDefault := false
		switch column.DatabaseTypeName() {
		case "decimal":
			actual, actualErr := decimal.NewFromString(current)
			expected, expectedErr := decimal.NewFromString(field.DefaultValue)
			sameDefault = actualErr == nil && expectedErr == nil && actual.Equal(expected)
		case "text":
			sameDefault = field.DefaultValue == "" && (current == "_utf8mb4''" || current == `_utf8mb4\'\'`)
		}
		if sameDefault {
			reported.DefaultValueValue.String = field.DefaultValue
			column = reported
		}
	}
	return m.Migrator.MigrateColumn(model, field, column)
}

func (d mysqlDialector) DataTypeOf(field *schema.Field) string {
	// The driver defaults unbounded strings with a uniqueIndex tag to LONGTEXT.
	// Give those keys an indexable width without narrowing columns on other databases.
	if field.DataType == schema.String && field.Size == 0 && field.TagSettings["UNIQUEINDEX"] != "" {
		sized := *field
		sized.Size = 255
		return d.Dialector.DataTypeOf(&sized)
	}
	return d.Dialector.DataTypeOf(field)
}

func openMySQL(dsn string) (*sql.DB, error) {
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	if cfg.DBName == "" {
		return nil, fmt.Errorf("MySQL DSN must specify a database")
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	if cfg.Params == nil {
		cfg.Params = make(map[string]string)
	}
	cfg.Params["time_zone"] = "'+00:00'"
	cfg.Params["default_storage_engine"] = "InnoDB"
	if err := cfg.Apply(mysqldriver.Charset("utf8mb4", "")); err != nil {
		return nil, err
	}
	connector, err := mysqldriver.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

func newGORMLogger(mode string, writer gormlogger.Writer) gormlogger.Interface {
	// SQL parameters can contain card secrets and credentials in every run mode.
	// Keep development query visibility without interpolating their values.
	level := gormlogger.Warn
	if !strings.EqualFold(strings.TrimSpace(mode), "release") {
		level = gormlogger.Info
	}
	if writer == nil {
		writer = log.New(os.Stdout, "\r\n", log.LstdFlags)
	}
	return gormlogger.New(writer, gormlogger.Config{
		SlowThreshold:             2 * time.Second,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		Colorful:                  false,
	})
}

// appendSQLitePragmas 在 SQLite DSN 中追加关键 PRAGMA 参数
func appendSQLitePragmas(dsn string) string {
	// glebarez/sqlite 支持 ?_pragma=key=value 形式
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep +
		"_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)"
}

func applyDBPool(sqlDB *sql.DB, pool DBPoolConfig) {
	if sqlDB == nil {
		return
	}
	if pool.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.MaxIdleConns >= 0 {
		sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetimeSeconds > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(pool.ConnMaxLifetimeSeconds) * time.Second)
	}
	if pool.ConnMaxIdleTimeSeconds > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(pool.ConnMaxIdleTimeSeconds) * time.Second)
	}
}
