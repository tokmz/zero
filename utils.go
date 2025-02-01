package main

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ConvertDataType 转换数据类型
func ConvertDataType(dbType string) string {
	switch dbType {
	case "int", "tinyint", "smallint", "mediumint":
		return "int32"
	case "bigint":
		return "int64"
	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext":
		return "string"
	case "date", "datetime", "timestamp":
		return "time.Time"
	case "decimal", "float", "double":
		return "float64"
	case "bool", "boolean":
		return "bool"
	default:
		return "string"
	}
}

// GenerateTag 生成结构体标签
func GenerateTag(name string, nullable bool, isPrimary bool, isAutoIncrement bool, dataType string, comment string) string {
	var tags []string

	// 添加列名
	tags = append(tags, fmt.Sprintf("column:%s", name))

	// 添加主键
	if isPrimary {
		tags = append(tags, "primaryKey")
		if isAutoIncrement {
			tags = append(tags, "autoIncrement")
		}
	}

	// 添加类型
	switch dataType {
	case "varchar", "char":
		tags = append(tags, "type:varchar(255)")
	case "text":
		tags = append(tags, "type:text")
	case "datetime", "timestamp":
		tags = append(tags, "type:datetime")
	case "int", "tinyint", "smallint", "mediumint":
		tags = append(tags, "type:int")
	case "bigint":
		tags = append(tags, "type:bigint unsigned")
	case "decimal", "float", "double":
		tags = append(tags, "type:decimal(10,2)")
	case "bool", "boolean":
		tags = append(tags, "type:tinyint(1)")
	}

	// 添加非空约束
	if !nullable {
		tags = append(tags, "not null")
	}

	// 添加注释
	if comment != "" {
		tags = append(tags, fmt.Sprintf("comment:%s", comment))
	}

	// 使用反引号避免转义问题
	return "`" + fmt.Sprintf(`gorm:"%s"`, strings.Join(tags, ";")) + "`"
}

// ToCamelCase 转换为大驼峰命名
func ToCamelCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = cases.Title(language.English).String(s)
	return strings.ReplaceAll(s, " ", "")
}

// ToSnakeCase 转换为下划线命名
func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// ContainsTimeField 检查字段中是否包含时间类型
func ContainsTimeField(fields []FieldInfo) bool {
	for _, field := range fields {
		if field.Type == "time.Time" {
			return true
		}
	}
	return false
}

// ParseTableRelations 解析关联关系配置
func ParseTableRelations(relations []Relation) *Relations {
	result := &Relations{}

	for _, rel := range relations {
		// 转换目标表名为大驼峰
		targetModel := ToCamelCase(rel.Target)
		fmt.Printf("解析关联关系: target=%s, type=%s, foreignKey=%s, references=%s\n",
			targetModel, rel.Type, rel.ForeignKey, rel.References)

		switch rel.Type {
		case "has_many":
			result.HasMany = append(result.HasMany, HasManyRelation{
				Table:      targetModel,
				ForeignKey: rel.ForeignKey,
				References: rel.References,
			})
		case "has_one":
			result.HasOne = append(result.HasOne, HasOneRelation{
				Table:      targetModel,
				ForeignKey: rel.ForeignKey,
				References: rel.References,
			})
		case "belongs_to":
			result.BelongsTo = append(result.BelongsTo, BelongsToRelation{
				Table:      targetModel,
				ForeignKey: rel.ForeignKey,
				References: rel.References,
			})
		case "many2many":
			result.ManyToMany = append(result.ManyToMany, ManyToManyRelation{
				Table:          targetModel,
				JoinTable:      rel.JoinTable,
				JoinForeignKey: rel.ForeignKey,
				References:     rel.References,
				JoinReferences: rel.JoinReferences,
			})
		}
	}

	return result
}
