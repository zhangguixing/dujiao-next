package gormstore

import (
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/persistence/gormutil"

	"gorm.io/gorm"
)

func localizedJSONCoalesceExpr(db *gorm.DB, column string) string {
	parts := make([]string, 0, len(constants.SupportedLocales)+1)
	for _, key := range constants.SupportedLocales {
		parts = append(parts, gormutil.JSONTextExpr(db, column, key))
	}
	parts = append(parts, "''")
	return fmt.Sprintf("COALESCE(%s)", strings.Join(parts, ", "))
}

func dateGroupExpr(db *gorm.DB, column string, startAt, endAt time.Time) string {
	loc := startAt.Location()
	switch gormutil.DialectName(db) {
	case "postgres", "postgresql":
		zoneName := loc.String()
		if zoneName == "" || zoneName == "Local" {
			zoneName = "UTC"
		}
		return fmt.Sprintf("TO_CHAR(%s AT TIME ZONE '%s', 'YYYY-MM-DD')", column, zoneName)
	case "mysql":
		return mysqlDateGroupExpr(column, startAt, endAt)
	default:
		_, offset := startAt.Zone()
		sign := "+"
		if offset < 0 {
			sign = "-"
			offset = -offset
		}
		hours := offset / 3600
		minutes := (offset % 3600) / 60
		if minutes != 0 {
			return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, '%s%d hours', '%s%d minutes')", column, sign, hours, sign, minutes)
		}
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, '%s%d hours')", column, sign, hours)
	}
}

// Compute offsets in Go so named time zones work even without MySQL time zone tables.
func mysqlDateGroupExpr(column string, startAt, endAt time.Time) string {
	var branches []string
	for cursor := startAt; ; {
		_, offset := cursor.Zone()
		shifted := fmt.Sprintf("TIMESTAMPADD(SECOND, %d, %s)", offset, column)
		_, next := cursor.ZoneBounds()
		if next.IsZero() || !next.Before(endAt) {
			if len(branches) > 0 {
				shifted = "CASE " + strings.Join(branches, " ") + " ELSE " + shifted + " END"
			}
			return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-%%d')", shifted)
		}
		branches = append(branches, fmt.Sprintf("WHEN %s < '%s' THEN %s", column, next.UTC().Format(time.DateTime), shifted))
		cursor = next
	}
}

// quotedStatusList only accepts internal status constants, never user input.
func quotedStatusList(statuses []string) string {
	parts := make([]string, len(statuses))
	for index, status := range statuses {
		parts[index] = "'" + status + "'"
	}
	return strings.Join(parts, ",")
}

// timeRangeQuery 返回时间范围查询的 WHERE 条件和参数
// 在 SQLite 下使用 datetime() 函数确保格式一致，避免字符串比较错误
func timeRangeQuery(db *gorm.DB, column string, startAt, endAt time.Time) (string, []interface{}) {
	switch gormutil.DialectName(db) {
	case "postgres", "postgresql":
		return fmt.Sprintf("%s >= ? AND %s < ?", column, column), []interface{}{startAt, endAt}
	case "mysql":
		return fmt.Sprintf("%s >= ? AND %s < ?", column, column), []interface{}{startAt.UTC(), endAt.UTC()}
	default:
		// SQLite: 使用 datetime() 函数确保时间比较正确
		return fmt.Sprintf("datetime(%s) >= datetime(?) AND datetime(%s) < datetime(?)", column, column), []interface{}{startAt.Format(time.RFC3339Nano), endAt.Format(time.RFC3339Nano)}
	}
}
