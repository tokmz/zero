package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// generateModelFile 生成单个model文件
func generateModelFile(table TableInfo, opts *GenerateOptions, tmpl *template.Template) error {
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
		"Package":    "model",                         // 默认包名
		"ImportTime": ContainsTimeField(table.Fields), // 是否需要导入time包
	}

	// 根据style生成文件名
	var fileName string
	switch opts.Style {
	case "snake":
		fileName = ToSnakeCase(modelName)
	case "camel":
		fileName = ToCamelCase(modelName)
		fileName = strings.ToLower(fileName[:1]) + fileName[1:]
	case "pascal":
		fileName = ToCamelCase(modelName)
	default:
		fileName = ToSnakeCase(modelName)
	}

	// 生成model文件
	modelFile := filepath.Join(opts.Dir, fileName+".go")
	if err := generateFile(tmpl, "model", modelFile, data); err != nil {
		return fmt.Errorf("生成model文件失败: %v", err)
	}

	return nil
}

// generateModelFiles 生成所有model文件
func generateModelFiles(tables []TableInfo, opts *GenerateOptions, tmpl *template.Template) error {
	for _, table := range tables {
		if err := generateModelFile(table, opts, tmpl); err != nil {
			return err
		}
	}
	return nil
}
