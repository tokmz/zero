package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

/*
   @NAME    : generator
   @author  : 清风
   @desc    :
   @time    : 2025/1/31 21:15
*/

// GenerateOptions 代码生成的配置选项
type GenerateOptions struct {
	DSN      string   // 数据库连接字符串
	Dir      string   // 输出目录
	Tables   []string // 要生成的表名列表
	Prefix   string   // 表名前缀
	Template string   // 自定义模板路径
	Style    string   // 文件命名风格: snake(下划线), camel(小驼峰), pascal(大驼峰)
}

// TableInfo 表信息
type TableInfo struct {
	Name    string      // 表名
	Comment string      // 表注释
	Fields  []FieldInfo // 字段列表
	Indexes []IndexInfo // 索引列表
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name       string // 字段名
	Type       string // 字段类型
	Comment    string // 字段注释
	Tag        string // 结构体标签
	IsNullable bool   // 是否可为空
	IsPrimary  bool   // 是否是主键
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name   string   // 索引名
	Fields []string // 索引字段
	IsPK   bool     // 是否是主键
	IsUniq bool     // 是否是唯一索引
}

// Generate 生成代码
func Generate(db *gorm.DB, opts *GenerateOptions) error {
	// 创建输出目录
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 获取数据库表信息
	tables, err := getTables(db, opts.Tables)
	if err != nil {
		return fmt.Errorf("获取表信息失败: %v", err)
	}

	// 加载模板
	tmpl, err := loadTemplates(opts.Template)
	if err != nil {
		return fmt.Errorf("加载模板失败: %v", err)
	}

	// 生成公共文件
	if err := generateCommonFiles(opts, tmpl); err != nil {
		return err
	}

	// 生成每个表的代码
	for _, table := range tables {
		// 去除表名前缀
		modelName := table.Name
		if opts.Prefix != "" {
			modelName = strings.TrimPrefix(modelName, opts.Prefix)
		}

		// 转换为大驼峰命名
		modelName = toCamelCase(modelName)

		// 生成代码
		data := map[string]interface{}{
			"TableName": table.Name,
			"ModelName": modelName,
			"Fields":    table.Fields,
			"Indexes":   table.Indexes,
			"Comment":   table.Comment,
			"Package":   "model", // 默认包名
		}

		// 根据style生成文件名
		var fileName string
		switch opts.Style {
		case "snake":
			fileName = toSnakeCase(modelName)
		case "camel":
			fileName = toCamelCase(modelName)
			fileName = strings.ToLower(fileName[:1]) + fileName[1:]
		case "pascal":
			fileName = toCamelCase(modelName)
		default:
			fileName = toSnakeCase(modelName)
		}

		// 生成model文件
		modelFile := filepath.Join(opts.Dir, fileName+".go")
		if err := generateFile(tmpl, "model", modelFile, data); err != nil {
			return fmt.Errorf("生成model文件失败: %v", err)
		}
	}

	return nil
}

// generateCommonFiles 生成公共文件
func generateCommonFiles(opts *GenerateOptions, tmpl *template.Template) error {
	// 生成orm.go
	ormFile := filepath.Join(opts.Dir, "orm.go")
	if err := generateFile(tmpl, "orm", ormFile, map[string]interface{}{
		"Package": "model",
	}); err != nil {
		return fmt.Errorf("生成orm文件失败: %v", err)
	}

	// 生成vars.go
	varsFile := filepath.Join(opts.Dir, "vars.go")
	if err := generateFile(tmpl, "vars", varsFile, map[string]interface{}{
		"Package": "model",
	}); err != nil {
		return fmt.Errorf("生成vars文件失败: %v", err)
	}

	return nil
}

// getTables 获取数据库表信息
func getTables(db *gorm.DB, tables []string) ([]TableInfo, error) {
	var result []TableInfo

	// 如果没有指定表名，则获取所有表
	if len(tables) == 0 {
		rows, err := db.Raw("SHOW TABLES").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}
	}

	// 获取每个表的详细信息
	for _, table := range tables {
		info, err := getTableInfo(db, table)
		if err != nil {
			return nil, err
		}
		result = append(result, *info)
	}

	return result, nil
}

// getTableInfo 获取单个表的详细信息
func getTableInfo(db *gorm.DB, tableName string) (*TableInfo, error) {
	var tableInfo TableInfo
	tableInfo.Name = tableName

	// 获取表注释
	row := db.Raw("SELECT table_comment FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Row()
	if err := row.Scan(&tableInfo.Comment); err != nil {
		return nil, err
	}

	// 获取字段信息
	rows, err := db.Raw(`
		SELECT 
			column_name, 
			data_type,
			column_comment,
			is_nullable,
			column_key,
			extra
		FROM information_schema.columns 
		WHERE table_schema = DATABASE() 
		AND table_name = ?
		ORDER BY ordinal_position
	`, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var field FieldInfo
		var dataType, isNullable, columnKey, extra string
		if err := rows.Scan(&field.Name, &dataType, &field.Comment, &isNullable, &columnKey, &extra); err != nil {
			return nil, err
		}

		field.IsNullable = isNullable == "YES"
		field.IsPrimary = columnKey == "PRI"
		field.Type = convertDataType(dataType)
		field.Tag = generateTag(field.Name, field.IsNullable, field.IsPrimary, extra == "auto_increment", dataType, field.Comment)

		tableInfo.Fields = append(tableInfo.Fields, field)
	}

	// 获取索引信息
	rows, err = db.Raw(`
		SELECT DISTINCT
			s1.index_name,
			GROUP_CONCAT(s1.column_name ORDER BY s1.seq_in_index) as columns,
			s1.non_unique
		FROM information_schema.statistics s1
		JOIN (
			SELECT index_name, MIN(seq_in_index) as min_seq
			FROM information_schema.statistics
			WHERE table_schema = DATABASE()
			AND table_name = ?
			GROUP BY index_name
		) s2 ON s1.index_name = s2.index_name
		WHERE s1.table_schema = DATABASE()
		AND s1.table_name = ?
		GROUP BY s1.index_name, s1.non_unique
	`, tableName, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var index IndexInfo
		var columns string
		var nonUnique int
		if err := rows.Scan(&index.Name, &columns, &nonUnique); err != nil {
			return nil, err
		}

		index.Fields = strings.Split(columns, ",")
		index.IsUniq = nonUnique == 0
		index.IsPK = index.Name == "PRIMARY"

		tableInfo.Indexes = append(tableInfo.Indexes, index)
	}

	return &tableInfo, nil
}

// loadTemplates 加载模板文件
func loadTemplates(templateDir string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"generateTag": generateTag,
		"toCamelCase": toCamelCase,
		"toSnakeCase": toSnakeCase,
		"now": func(layout string) string {
			return time.Now().Format(layout)
		},
	}

	// 创建带有函数的模板
	tmpl := template.New("").Funcs(funcMap)

	// 如果没有指定模板目录，使用默认的template目录
	if templateDir == "" {
		templateDir = "template"
	}

	// 检查模板目录是否存在
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("模板目录不存在: %s", templateDir)
	}

	// 加载所有模板文件
	templateFiles := []string{
		filepath.Join(templateDir, "model.tmpl"),
		filepath.Join(templateDir, "orm.tmpl"),
		filepath.Join(templateDir, "vars.tmpl"),
	}

	// 检查所有必需的模板文件是否存在
	for _, file := range templateFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return nil, fmt.Errorf("模板文件不存在: %s", file)
		}
	}

	// 解析所有模板文件
	return tmpl.ParseFiles(templateFiles...)
}

// generateFile 生成文件
func generateFile(tmpl *template.Template, name, file string, data interface{}) error {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}

	return os.WriteFile(file, buf.Bytes(), 0644)
}

// convertDataType 转换数据类型
func convertDataType(dbType string) string {
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

// generateTag 生成结构体标签
func generateTag(name string, nullable bool, isPrimary bool, isAutoIncrement bool, dataType string, comment string) string {
	var tags []string
	tags = append(tags, fmt.Sprintf("column:%s", name))

	if isPrimary {
		tags = append(tags, "primaryKey")
	}

	if isAutoIncrement {
		tags = append(tags, "autoIncrement")
	}

	if !nullable {
		tags = append(tags, "not null")
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
		tags = append(tags, "type:bigint")
	case "decimal", "float", "double":
		tags = append(tags, "type:decimal(10,2)")
	case "bool", "boolean":
		tags = append(tags, "type:tinyint(1)")
	}

	// 添加注释
	if comment != "" {
		tags = append(tags, fmt.Sprintf("comment:%s", comment))
	}

	return fmt.Sprintf("gorm:\"%s\"", strings.Join(tags, ";"))
}

// toCamelCase 转换为大驼峰命名
func toCamelCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = cases.Title(language.English).String(s)
	return strings.ReplaceAll(s, " ", "")
}

// toSnakeCase 转换为下划线命名
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
