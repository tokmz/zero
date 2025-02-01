package main

import (
	"fmt"
	"path/filepath"
	"text/template"
)

// generateOrmFile 生成 orm.go 文件
func generateOrmFile(opts *GenerateOptions, tmpl *template.Template) error {
	// 生成文件
	ormFile := filepath.Join(opts.Dir, "orm.go")
	if err := generateFile(tmpl, "orm", ormFile, map[string]interface{}{
		"Package": "model",
	}); err != nil {
		return fmt.Errorf("生成orm文件失败: %v", err)
	}

	return nil
}
