package gormutil

import (
	"fmt"
	"strings"

	"github.com/dujiao-next/internal/constants"

	"gorm.io/gorm"
)

var localizedJSONSearchKeys = append([]string(nil), constants.SupportedLocales...)

func DialectName(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return "sqlite"
	}
	name := strings.ToLower(strings.TrimSpace(db.Dialector.Name()))
	if name == "" {
		return "sqlite"
	}
	return name
}

func JSONTextExpr(db *gorm.DB, column, key string) string {
	return JSONTextExprByDialect(DialectName(db), column, key)
}

func JSONTextExprByDialect(dialect, column, key string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		return fmt.Sprintf("(%s::jsonb ->> '%s')", column, key)
	case "mysql":
		extracted := fmt.Sprintf("JSON_EXTRACT(%s, '$.\"%s\"')", column, key)
		return fmt.Sprintf("(CASE WHEN JSON_TYPE(%s) <> 'NULL' THEN JSON_UNQUOTE(%s) END)", extracted, extracted)
	default:
		return fmt.Sprintf("json_extract(%s, '$.\"%s\"')", column, key)
	}
}

func JSONArrayLengthExpr(db *gorm.DB, column string) string {
	return JSONArrayLengthExprByDialect(DialectName(db), column)
}

func JSONArrayLengthExprByDialect(dialect, column string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		return fmt.Sprintf("jsonb_array_length(COALESCE(%s::jsonb, '[]'::jsonb))", column)
	case "mysql":
		return fmt.Sprintf("(CASE WHEN JSON_TYPE(%s) = 'ARRAY' THEN JSON_LENGTH(%s) ELSE 0 END)", column, column)
	default:
		return fmt.Sprintf("json_array_length(COALESCE(%s, '[]'))", column)
	}
}

// BuildLocalizedLikeCondition 构建普通列和多语言 JSON 列的模糊查询条件。
func BuildLocalizedLikeCondition(db *gorm.DB, plainColumns, jsonColumns []string) (string, int) {
	return BuildLocalizedLikeConditionByDialect(DialectName(db), plainColumns, jsonColumns)
}

func BuildLocalizedLikeConditionByDialect(dialect string, plainColumns, jsonColumns []string) (string, int) {
	parts := make([]string, 0, len(plainColumns)+len(jsonColumns)*len(localizedJSONSearchKeys))
	argCount := 0
	operator := "LIKE"
	normalized := strings.ToLower(strings.TrimSpace(dialect))
	if normalized == "postgres" || normalized == "postgresql" {
		operator = "ILIKE"
	}
	likeCondition := func(expr string) string {
		if normalized == "mysql" {
			return expr + " COLLATE utf8mb4_0900_as_ci LIKE ?"
		}
		return fmt.Sprintf("%s %s ?", expr, operator)
	}

	for _, column := range plainColumns {
		trimmed := strings.TrimSpace(column)
		if trimmed == "" {
			continue
		}
		parts = append(parts, likeCondition(trimmed))
		argCount++
	}

	for _, column := range jsonColumns {
		trimmed := strings.TrimSpace(column)
		if trimmed == "" {
			continue
		}
		for _, key := range localizedJSONSearchKeys {
			parts = append(parts, likeCondition(JSONTextExprByDialect(dialect, trimmed, key)))
			argCount++
		}
	}

	return strings.Join(parts, " OR "), argCount
}

func RepeatLikeArgs(like string, count int) []interface{} {
	args := make([]interface{}, 0, count)
	for i := 0; i < count; i++ {
		args = append(args, like)
	}
	return args
}

func ApplyPagination(query *gorm.DB, page, pageSize int) *gorm.DB {
	if query == nil || pageSize <= 0 {
		return query
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return query.Limit(pageSize).Offset(offset)
}
