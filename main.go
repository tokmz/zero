package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cInit "github.com/tokmz/zero/cmd"
	"github.com/tokmz/zero/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

/*
   @NAME    : main
   @author  : 清风
   @desc    :
   @time    : 2025/2/6 11:31
*/

// 命令行参数
type cmdFlags struct {
	DSN      string
	Dir      string
	Tables   string
	Prefix   string
	Template string
	Style    string
}

var (
	// 最终配置
	cfg = &config.Config{}
	// 命令行参数
	flags = &cmdFlags{}
	// 配置文件路径
	configFile string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:     "zero",
	Short:   "Zero 是一个代码生成工具",
	Long:    `Zero 是一个基于数据库表结构生成 Go 代码的工具，支持 GORM、自定义模板等特性。`,
	Version: "v1.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Zero 是一个代码生成工具")
		cmd.Help()
	},
}

// genCmd 生成代码的命令
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "生成代码",
	Long:  `根据数据库表结构生成 Go 代码，支持自定义模板和多种命名风格。`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// 如果指定了配置文件，则从配置文件读取
		if cmd.Flags().Changed("config") {
			viper.SetConfigFile(configFile)
			viper.SetConfigType("yaml")

			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("读取配置文件失败: %v", err)
			}

			// 从配置文件读取配置
			flags.DSN = viper.GetString("dsn")
			flags.Dir = viper.GetString("output.orm_dir")
			flags.Tables = viper.GetString("tables")
			flags.Prefix = viper.GetString("prefix")
			flags.Template = viper.GetString("template")
			flags.Style = viper.GetString("style")
			cfg.ModuleName = viper.GetString("module_name")

			// 读取关联关系配置
			if relations := viper.GetStringMap("relations"); len(relations) > 0 {
				// fmt.Println("\n读取到关联关系配置:")
				cfg.Relations = make(map[string][]config.Relation)
				for tableName, rel := range relations {
					// fmt.Printf("  处理表 %s 的关联关系\n", tableName)
					if relSlice, ok := rel.([]interface{}); ok {
						var tableRelations []config.Relation
						for _, item := range relSlice {
							if itemMap, ok := item.(map[string]interface{}); ok {
								relation := config.Relation{
									Target:         getString(itemMap, "target"),
									Type:           getString(itemMap, "type"),
									ForeignKey:     getString(itemMap, "foreign_key"),
									References:     getString(itemMap, "references"),
									JoinTable:      getString(itemMap, "join_table"),
									JoinForeignKey: getString(itemMap, "join_foreign_key"),
									JoinReferences: getString(itemMap, "join_references"),
									Comment:        getString(itemMap, "comment"),
								}
								// fmt.Printf("    - 目标表: %s, 类型: %s, 外键: %s\n",
								// 	relation.Target, relation.Type, relation.ForeignKey)
								tableRelations = append(tableRelations, relation)
							}
						}
						if len(tableRelations) > 0 {
							cfg.Relations[tableName] = tableRelations
							// fmt.Printf("  成功添加 %d 个关联关系\n", len(tableRelations))
						}
					} else {
						fmt.Printf("  警告: 表 %s 的关联关系格式不正确\n", tableName)
					}
				}
			} // else {
			// fmt.Println("\n未找到关联关系配置")
			// }
		}

		// 命令行参数优先级高于配置文件
		cmd.Flags().Visit(func(f *pflag.Flag) {
			switch f.Name {
			case "dsn":
				flags.DSN = f.Value.String()
			case "dir":
				flags.Dir = f.Value.String()
			case "tables":
				flags.Tables = f.Value.String()
			case "prefix":
				flags.Prefix = f.Value.String()
			case "template":
				flags.Template = f.Value.String()
			case "style":
				flags.Style = f.Value.String()
			}
		})

		// 验证并转换参数
		if flags.DSN == "" {
			return fmt.Errorf("数据库DSN是必填的")
		}

		switch flags.Style {
		case "snake", "camel", "pascal":
		default:
			return fmt.Errorf("不支持的命名风格: %s", flags.Style)
		}

		// 转换为最终配置
		cfg.DSN = flags.DSN
		cfg.Output.OrmDir = flags.Dir
		if cfg.Output.ModelDir == "" {
			// 如果没有指定 model_dir，则使用 orm_dir/model 作为 model 目录
			cfg.Output.ModelDir = filepath.Join(cfg.Output.OrmDir, "model")
		}
		if cfg.Output.QueryDir == "" {
			// 如果没有指定 query_dir，则使用 orm_dir/query 作为 query 目录
			cfg.Output.QueryDir = filepath.Join(cfg.Output.OrmDir, "query")
		}
		if flags.Tables != "" {
			cfg.Tables = strings.Split(flags.Tables, ",")
		} else {
			cfg.Tables = []string{} // 空切片表示生成所有表
		}
		cfg.Prefix = flags.Prefix
		cfg.Template = flags.Template
		cfg.Style = flags.Style

		// 如果没有关联关系配置，初始化一个空的 map
		if cfg.Relations == nil {
			cfg.Relations = make(map[string][]config.Relation)
		}

		// 注释掉调试信息输出
		// fmt.Println("\n关联关系配置:")
		// for table, relations := range cfg.Relations {
		// 	fmt.Printf("  表 %s 的关联关系: %d 个\n", table, len(relations))
		// 	for _, rel := range relations {
		// 		fmt.Printf("    - [%s] %s -> %s\n", rel.Type, table, rel.Target)
		// 		fmt.Printf("      外键: %s, 引用: %s\n", rel.ForeignKey, rel.References)
		// 		if rel.JoinTable != "" {
		// 			fmt.Printf("      连接表: %s (外键: %s, 引用: %s)\n",
		// 				rel.JoinTable, rel.JoinForeignKey, rel.JoinReferences)
		// 		}
		// 		if rel.Comment != "" {
		// 			fmt.Printf("      说明: %s\n", rel.Comment)
		// 		}
		// 	}
		// }

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("执行生成逻辑")
		// 执行生成逻辑
		return cInit.Init(cfg)
	},
}

// getString 安全地获取 map 中的字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// 全局参数
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径")

	// gen 子命令的参数
	genCmd.Flags().StringVarP(&flags.DSN, "dsn", "d", "", "数据库DSN连接串，格式：user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
	genCmd.Flags().StringVarP(&flags.Dir, "dir", "o", ".", "生成代码的输出目录")
	genCmd.Flags().StringVarP(&flags.Tables, "tables", "t", "", "要生成的表名，多个表用逗号分隔")
	genCmd.Flags().StringVarP(&flags.Prefix, "prefix", "p", "", "表名前缀，生成代码时会去除这个前缀")
	genCmd.Flags().StringVar(&flags.Template, "template", "", "自定义模板文件路径")
	genCmd.Flags().StringVarP(&flags.Style, "style", "s", "snake", "生成的文件命名风格: snake(下划线), camel(小驼峰), pascal(大驼峰)")

	// 设置 viper 默认值
	viper.SetDefault("dir", ".")
	viper.SetDefault("style", "snake")

	// 支持环境变量
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ZERO") // 环境变量前缀 ZERO_

	// 添加子命令
	rootCmd.AddCommand(genCmd)
}

func main() {
	Execute()
}
