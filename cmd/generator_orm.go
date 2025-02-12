package cmd

/*
   @NAME    : generator_orm
   @author  : 清风
   @desc    :
   @time    : 2025/2/6 11:31
*/

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/tokmz/zero/config"
	tm "github.com/tokmz/zero/template"
)

// GenerateOrm 生成 ORM 代码
func GenerateOrm(tables []*config.TableInfo, cfg *config.Config) error {
	// 获取包名（从目录路径中获取）
	dirParts := strings.Split(strings.Trim(cfg.Output.OrmDir, "/"), "/")
	var packageName string
	if len(dirParts) > 0 {
		packageName = dirParts[len(dirParts)-1]
	} else {
		packageName = "orm" // 默认包名
	}

	// 准备模板数据
	data := map[string]interface{}{
		"Package":       packageName,
		"Tables":        tables,
		"EnableTracing": cfg.EnableTracing,
	}

	// 加载模板
	tmpl := template.New("orm")

	// 如果指定了自定义模板，则使用自定义模板
	var err error
	if cfg.Template != "" {
		tmpl, err = tmpl.ParseFiles(filepath.Join(filepath.Dir(cfg.Template), "orm.tmpl"))
		if err != nil {
			return fmt.Errorf("解析自定义模板失败: %v", err)
		}
	} else {
		// 使用默认模板
		// 使用嵌入的模板文件
		tmplContent, err := tm.Templates.ReadFile("orm.tmpl")
		if err != nil {
			return fmt.Errorf("读取模板文件失败: %v", err)
		}
		tmpl, err = tmpl.Parse(string(tmplContent))
		if err != nil {
			return fmt.Errorf("解析默认模板失败: %v", err)
		}
	}

	// 生成代码
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "orm", data); err != nil {
		return fmt.Errorf("生成代码失败: %v", err)
	}

	// 格式化代码
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("格式化代码失败: %v", err)
	}

	// 创建输出目录
	outputDir := cfg.Output.OrmDir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 写入文件
	outputFile := filepath.Join(outputDir, "orm.go")
	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("  生成文件: %s\n", outputFile)
	return nil
}
