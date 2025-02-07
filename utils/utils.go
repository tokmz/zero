package utils

import (
	"fmt"
	"strings"
)

/*
   @NAME    : utils
   @author  : 清风
   @desc    :
   @time    : 2025/2/6 11:33
*/

// ToSnake 转换为蛇形命名
func ToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result = append(result, '_')
		}
		result = append(result, rune(strings.ToLower(string(r))[0]))
	}
	return string(result)
}

// ToCamel 转换为驼峰命名
func ToCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	var result []rune
	upperNext := true
	for _, r := range s {
		if r == '_' {
			upperNext = true
		} else {
			if upperNext {
				result = append(result, rune(strings.ToUpper(string(r))[0]))
				upperNext = false
			} else {
				result = append(result, rune(strings.ToLower(string(r))[0]))
			}
		}
	}
	return string(result)
}

// GetGoType 将数据库类型转换为 Go 类型
func GetGoType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "tinyint", "smallint", "mediumint", "int", "integer":
		return "int"
	case "bigint":
		return "int64"
	case "float", "double", "decimal", "numeric":
		return "float64"
	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext":
		return "string"
	case "date", "datetime", "timestamp", "time":
		return "time.Time"
	case "tinyint(1)", "bool", "boolean":
		return "bool"
	case "json":
		return "json.RawMessage"
	default:
		return "string"
	}
}

// BuildFieldTags 构建字段标签
func BuildFieldTags(name, columnType string, isNullable bool) string {
	// 移除多余的空格
	columnType = strings.TrimSpace(columnType)

	// 构建 gorm tag
	gormTag := fmt.Sprintf("column:%s;type:%s", name, columnType)
	if !isNullable {
		gormTag += ";not null"
	}

	return fmt.Sprintf(`gorm:"%s" json:"%s"`, gormTag, name)
}
