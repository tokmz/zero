package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// generateQueryFile 生成单个查询文件
func generateQueryFile(table TableInfo, opts *GenerateOptions, tmpl *template.Template) error {
	// 去除表名前缀
	modelName := table.Name
	if opts.Prefix != "" {
		modelName = strings.TrimPrefix(modelName, opts.Prefix)
	}

	// 转换为大驼峰命名
	modelName = ToCamelCase(modelName)

	// 准备模板数据
	data := map[string]interface{}{
		"TableName":  table.Name,
		"ModelName":  modelName,
		"Fields":     table.Fields,
		"Indexes":    table.Indexes,
		"Comment":    table.Comment,
		"Relations":  table.Relations,
		"Package":    "model",
		"ImportTime": ContainsTimeField(table.Fields),
	}

	// 根据style生成文件名
	var fileName string
	switch opts.Style {
	case "snake":
		fileName = ToSnakeCase(modelName) + "_query"
	case "camel":
		fileName = ToCamelCase(modelName)
		fileName = strings.ToLower(fileName[:1]) + fileName[1:] + "Query"
	case "pascal":
		fileName = ToCamelCase(modelName) + "Query"
	default:
		fileName = ToSnakeCase(modelName) + "_query"
	}

	// 生成query文件，直接放在opts.Dir目录下
	queryFile := filepath.Join(opts.Dir, fileName+".go")
	if err := generateFile(tmpl, "query", queryFile, data); err != nil {
		return fmt.Errorf("生成query文件失败: %v", err)
	}

	return nil
}

// generateQueryFiles 生成所有查询文件
func generateQueryFiles(tables []TableInfo, opts *GenerateOptions, tmpl *template.Template) error {
	// 生成每个表的查询文件
	for _, table := range tables {
		if err := generateQueryFile(table, opts, tmpl); err != nil {
			return err
		}
	}

	return nil
}
