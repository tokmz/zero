package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// 数据库连接信息
	dsn = flag.String("dsn", "", "数据库DSN连接串，格式：user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local")

	// 代码生成配置
	dir      = flag.String("dir", ".", "生成代码的输出目录")
	tables   = flag.String("tables", "", "要生成的表名，多个表用逗号分隔")
	prefix   = flag.String("prefix", "", "表名前缀，生成代码时会去除这个前缀")
	tmplPath = flag.String("template", "", "自定义模板文件路径")
	style    = flag.String("style", "snake", "生成的文件命名风格: snake(下划线), camel(小驼峰), pascal(大驼峰)")
)

func main() {
	flag.Parse()

	// 验证必填参数
	if *dsn == "" {
		fmt.Println("Error: 数据库DSN是必填的")
		flag.Usage()
		os.Exit(1)
	}

	// 验证style参数
	if *style != "snake" && *style != "camel" && *style != "pascal" {
		fmt.Println("Error: style参数只能是snake、camel或pascal")
		flag.Usage()
		os.Exit(1)
	}

	// 解析表名
	var tableList []string
	if *tables != "" {
		tableList = strings.Split(*tables, ",")
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 生成代码
	if err := Generate(db, &GenerateOptions{
		DSN:      *dsn,
		Dir:      *dir,
		Tables:   tableList,
		Prefix:   *prefix,
		Template: *tmplPath,
		Style:    *style,
	}); err != nil {
		fmt.Printf("生成代码失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("代码生成成功!")
}
