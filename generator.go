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

// Generate 生成代码
func Generate(db *gorm.DB, opts *GenerateOptions) error {
	// 创建输出目录
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 获取数据库表信息
	tables, err := getTables(db, opts.Tables, opts.Relations)
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
			"TableName":  table.Name,
			"ModelName":  modelName,
			"Fields":     table.Fields,
			"Indexes":    table.Indexes,
			"Comment":    table.Comment,
			"Relations":  table.Relations,
			"Package":    "model",                         // 默认包名
			"ImportTime": containsTimeField(table.Fields), // 是否需要导入time包
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
func getTables(db *gorm.DB, tables []string, relations map[string][]Relation) ([]TableInfo, error) {
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
		info, err := getTableInfo(db, table, relations)
		if err != nil {
			return nil, err
		}
		result = append(result, *info)
	}

	return result, nil
}

// getTableInfo 获取单个表的详细信息
func getTableInfo(db *gorm.DB, tableName string, relations map[string][]Relation) (*TableInfo, error) {
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

		// 转换字段名为大驼峰
		field.Name = toCamelCase(field.Name)

		field.IsNullable = isNullable == "YES"
		field.IsPrimary = columnKey == "PRI"
		field.Type = convertDataType(dataType)
		field.Tag = generateTag(toSnakeCase(field.Name), field.IsNullable, field.IsPrimary, extra == "auto_increment", dataType, field.Comment)

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

	// 处理关联关系
	if relations != nil {
		if tableRelations, ok := relations[tableName]; ok {
			tableInfo.Relations = parseTableRelations(tableRelations)
		}
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

	// 处理生成的内容
	content := buf.String()
	// 删除多余的空行
	content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	// 删除文件末尾的空行
	content = strings.TrimRight(content, "\n")
	// 确保文件以一个换行符结束
	content = content + "\n"

	return os.WriteFile(file, []byte(content), 0644)
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

	// 添加列名
	tags = append(tags, fmt.Sprintf("column:%s", name))

	// 添加主键
	if isPrimary {
		tags = append(tags, "primaryKey")
		if isAutoIncrement {
			tags = append(tags, "autoIncrement")
		}
	}

	// 添加非空约束
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
		tags = append(tags, "type:bigint unsigned")
	case "decimal", "float", "double":
		tags = append(tags, "type:decimal(10,2)")
	case "bool", "boolean":
		tags = append(tags, "type:tinyint(1)")
	}

	// 添加注释
	if comment != "" {
		tags = append(tags, fmt.Sprintf("comment:%s", comment))
	}

	// 使用反引号避免转义问题
	return "`" + fmt.Sprintf(`gorm:"%s"`, strings.Join(tags, ";")) + "`"
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

// parseTableRelations 解析关联关系配置
func parseTableRelations(relations []Relation) *Relations {
	result := &Relations{}

	for _, rel := range relations {
		// 转换目标表名为大驼峰
		targetModel := toCamelCase(rel.Target)
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

// containsTimeField 检查字段中是否包含时间类型
func containsTimeField(fields []FieldInfo) bool {
	for _, field := range fields {
		if field.Type == "time.Time" {
			return true
		}
	}
	return false
}
