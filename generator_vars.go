package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// generateVarsFile 生成vars.go文件
func generateVarsFile(tables []TableInfo, opts *GenerateOptions, tmpl *template.Template) error {
	// 处理表名为模型名
	type TableData struct {
		Name      string
		ModelName string
	}
	var processedTables []TableData
	for _, table := range tables {
		modelName := table.Name
		if opts.Prefix != "" {
			modelName = strings.TrimPrefix(modelName, opts.Prefix)
		}
		modelName = ToCamelCase(modelName)
		processedTables = append(processedTables, TableData{
			Name:      table.Name,
			ModelName: modelName,
		})
	}

	// 生成vars.go
	varsFile := filepath.Join(opts.Dir, "vars.go")
	if err := generateFile(tmpl, "vars", varsFile, map[string]interface{}{
		"Package": "model",
		"Tables":  processedTables,
	}); err != nil {
		return fmt.Errorf("生成vars文件失败: %v", err)
	}

	return nil
}
