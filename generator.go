package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	// "path/filepath"
	"strings"
	"text/template"
	"time"

	"gorm.io/gorm"
)

//go:embed template/*
var templates embed.FS

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
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("加载模板失败: %v", err)
	}

	// 生成orm.go
	if err := generateOrmFile(opts, tmpl); err != nil {
		return err
	}

	// 生成model文件
	if err := generateModelFiles(tables, opts, tmpl); err != nil {
		return err
	}

	// 生成query文件
	if err := generateQueryFiles(tables, opts, tmpl); err != nil {
		return err
	}

	if err := generateVarsFile(tables, opts, tmpl); err != nil {
		return err
	}

	// 生成field文件
	if err := generateFieldFile(opts, tmpl); err != nil {
		return err
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

	// 获取表的所有列信息
	rows, err := db.Raw(`
		SELECT 
			column_name,
			column_type,
			is_nullable,
			column_key,
			extra,
			column_comment
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
		var columnType, isNullable, columnKey, extra string
		if err := rows.Scan(
			&field.Name,
			&columnType,
			&isNullable,
			&columnKey,
			&extra,
			&field.Comment,
		); err != nil {
			return nil, err
		}

		// 处理字段类型
		field.Type = ConvertDataType(strings.Split(columnType, "(")[0])
		field.Tag = GenerateTag(
			field.Name,
			isNullable == "NO",
			columnKey == "PRI",
			strings.Contains(extra, "auto_increment"),
			strings.Split(columnType, "(")[0],
			field.Comment,
		)

		tableInfo.Fields = append(tableInfo.Fields, field)
	}

	// 处理关联关系
	if rels, ok := relations[tableName]; ok {
		tableInfo.Relations = ParseTableRelations(rels)
	}

	return &tableInfo, nil
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

// loadTemplates 加载模板文件
func loadTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"generateTag": GenerateTag,
		"toCamelCase": ToCamelCase,
		"toSnakeCase": ToSnakeCase,
		"toLowerCamel": func(s string) string {
			if s == "" {
				return s
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
		"now": func(layout string) string {
			return time.Now().Format(layout)
		},
	}

	// 创建带有函数的模板
	tmpl := template.New("").Funcs(funcMap)

	// 读取嵌入的模板文件
	entries, err := templates.ReadDir("template")
	if err != nil {
		return nil, fmt.Errorf("读取模板目录失败: %v", err)
	}

	// 遍历并解析每个模板文件
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
			// 读取模板内容
			content, err := templates.ReadFile("template/" + entry.Name())
			if err != nil {
				return nil, fmt.Errorf("读取模板文件失败 %s: %v", entry.Name(), err)
			}

			// 获取模板名称（不包含扩展名）
			templateName := strings.TrimSuffix(entry.Name(), ".tmpl")

			// 解析模板内容
			tmpl, err = tmpl.New(templateName).Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("解析模板文件失败 %s: %v", entry.Name(), err)
			}
		}
	}

	return tmpl, nil
}

func generateFieldFile(opts *GenerateOptions, tmpl *template.Template) error {
	data := struct {
		Package string
	}{
		Package: filepath.Base(opts.Dir),
	}
	return generateFile(tmpl, "field", filepath.Join(opts.Dir, "field.go"), data)
}
