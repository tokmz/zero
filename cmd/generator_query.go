package cmd

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
	"github.com/tokmz/zero/utils"
)

/*
   @NAME    : generator_query
   @author  : 清风
   @desc    :
   @time    : 2025/2/6 11:32
*/

// GenerateQuery 生成 Query 代码
func GenerateQuery(table *config.TableInfo, cfg *config.Config) error {
	// 获取包名（固定使用 query）
	packageName := "query"

	// 准备模板数据
	data := map[string]interface{}{
		"Package":    packageName,
		"TableName":  table.Name,
		"Comment":    table.Comment,
		"Fields":     table.Fields,
		"Relations":  table.Relations,
		"ModelPath":  strings.TrimPrefix(cfg.Output.ModelDir, "./"),
		"ModuleName": cfg.ModuleName,
	}

	// 加载模板
	tmpl := template.New("query")

	// 添加自定义函数
	tmpl = tmpl.Funcs(template.FuncMap{
		"ToSnake":        utils.ToSnake,
		"ToCamel":        utils.ToCamel,
		"ToLower":        strings.ToLower,
		"ToUpper":        strings.ToUpper,
		"Contains":       strings.Contains,
		"not":            func(b bool) bool { return !b },
		"BuildFieldTags": utils.BuildFieldTags,
	})

	// 如果指定了自定义模板，则使用自定义模板
	var err error
	if cfg.Template != "" {
		tmpl, err = tmpl.ParseFiles(filepath.Join(filepath.Dir(cfg.Template), "query.tmpl"))
		if err != nil {
			return fmt.Errorf("解析自定义模板失败: %v", err)
		}
	} else {
		// 使用默认模板
		// 使用嵌入的模板文件
		tmplContent, err := tm.Templates.ReadFile("query.tmpl")
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
	if err := tmpl.ExecuteTemplate(&buf, "query", data); err != nil {
		return fmt.Errorf("生成代码失败: %v", err)
	}

	// 格式化代码
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("格式化代码失败: %v", err)
	}

	// 创建输出目录
	outputDir := filepath.Join(cfg.Output.OrmDir, "query")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 生成文件名
	var filename string
	switch cfg.Style {
	case "snake":
		filename = fmt.Sprintf("%s.go", utils.ToSnake(table.Name))
	case "camel":
		filename = fmt.Sprintf("%s.go", utils.ToCamel(table.Name))
	case "pascal":
		filename = fmt.Sprintf("%s.go", utils.ToCamel(table.Name))
	default:
		filename = fmt.Sprintf("%s.go", table.Name)
	}

	// 写入文件
	outputFile := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("  生成文件: %s\n", outputFile)
	return nil
}
