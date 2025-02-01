package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// 配置文件
	configFile = flag.String("config", "", "配置文件路径")

	// 数据库连接信息
	dsn = flag.String("dsn", "", "数据库DSN连接串，格式：user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local")

	// 代码生成配置
	dir      = flag.String("dir", ".", "生成代码的输出目录")
	tables   = flag.String("tables", "", "要生成的表名，多个表用逗号分隔")
	prefix   = flag.String("prefix", "", "表名前缀，生成代码时会去除这个前缀")
	tmplPath = flag.String("template", "", "自定义模板文件路径")
	style    = flag.String("style", "snake", "生成的文件命名风格: snake(下划线), camel(小驼峰), pascal(大驼峰)")
)

func loadConfig(configFile string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("dir", ".")
	v.SetDefault("style", "snake")

	if configFile != "" {
		// 设置配置文件
		v.SetConfigFile(configFile)

		// 读取配置文件
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %v", err)
		}
	}

	// 解析配置到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	return &config, nil
}

func main() {
	flag.Parse()

	var config *Config
	var err error

	// 优先使用配置文件
	if *configFile != "" {
		config, err = loadConfig(*configFile)
		if err != nil {
			fmt.Printf("加载配置文件失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 使用命令行参数
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

		// 使用命令行参数构造配置
		config = &Config{
			DSN:      *dsn,
			Dir:      *dir,
			Tables:   tableList,
			Prefix:   *prefix,
			Style:    *style,
			Template: *tmplPath,
		}
	}

	// 验证配置完整性
	if config.DSN == "" {
		fmt.Println("Error: 数据库DSN是必填的")
		os.Exit(1)
	}

	if config.Style != "snake" && config.Style != "camel" && config.Style != "pascal" {
		fmt.Println("Error: style参数只能是snake、camel或pascal")
		os.Exit(1)
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(config.DSN), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 生成代码
	if err := Generate(db, &GenerateOptions{
		DSN:       config.DSN,
		Dir:       config.Dir,
		Tables:    config.Tables,
		Prefix:    config.Prefix,
		Template:  config.Template,
		Style:     config.Style,
		Relations: config.Relations,
	}); err != nil {
		fmt.Printf("生成代码失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("代码生成成功!")
}
