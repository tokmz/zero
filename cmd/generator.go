package cmd

import (
	"fmt"

	"github.com/tokmz/zero/config"
	"github.com/tokmz/zero/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

/*
   @NAME    : generator.go
   @author  : 清风
   @desc    :
   @time    : 2025/2/6 11:31
*/

// Init 初始化并执行代码生成
func Init(cfg *config.Config) error {
	// 打印配置信息
	fmt.Println("配置信息:")
	fmt.Printf("  DSN: %s\n", cfg.DSN)
	fmt.Printf("  输出目录:\n")
	fmt.Printf("    - Model: %s\n", cfg.Output.ModelDir)
	fmt.Printf("    - Query: %s\n", cfg.Output.QueryDir)
	fmt.Printf("  表名: %v\n", cfg.Tables)
	fmt.Printf("  命名风格: %s\n", cfg.Style)
	// if len(cfg.Relations) > 0 {
	// 	fmt.Println("  关联关系配置:")
	// 	for table, relations := range cfg.Relations {
	// 		fmt.Printf("    %s 表: %d 个关联\n", table, len(relations))
	// 		for _, rel := range relations {
	// 			fmt.Printf("      - %s -> %s (%s)\n", rel.Type, rel.Target, rel.Comment)
	// 		}
	// 	}
	// }

	// 获取数据库表结构信息
	tableInfos, err := connectDB(cfg)
	if err != nil {
		return fmt.Errorf("获取表结构失败: %v", err)
	}

	// 打印调试信息
	for _, table := range tableInfos {
		fmt.Printf("\n处理表: %s (%s)\n", table.Name, table.Comment)
		fmt.Printf("  字段数量: %d\n", len(table.Fields))
		fmt.Printf("  索引数量: %d\n", len(table.Indexes))
		if len(table.Relations) > 0 {
			fmt.Printf("  关联关系: %d 个\n", len(table.Relations))
			for _, rel := range table.Relations {
				fmt.Printf("    - [%s] %s -> %s\n", rel.Type, table.Name, rel.Model)
				fmt.Printf("      外键: %s, 引用: %s\n", rel.ForeignKey, rel.References)
				if rel.JoinTable != "" {
					fmt.Printf("      连接表: %s (外键: %s, 引用: %s)\n",
						rel.JoinTable, rel.JoinForeignKey, rel.JoinReferences)
				}
				if rel.Comment != "" {
					fmt.Printf("      说明: %s\n", rel.Comment)
				}
			}
		}
	}

	// 生成 ORM 代码
	if err := GenerateOrm(tableInfos, cfg); err != nil {
		return fmt.Errorf("生成 ORM 代码失败: %v", err)
	}

	// TODO: 根据模板生成代码
	// 1. 加载模板
	// 2. 生成代码
	// 3. 格式化代码
	// 4. 写入文件
	for _, table := range tableInfos {
		// 生成 model 代码
		if err := GenerateModel(table, cfg); err != nil {
			return fmt.Errorf("生成表 %s 的模型代码失败: %v", table.Name, err)
		}

		// 生成 query 代码
		if err := GenerateQuery(table, cfg); err != nil {
			return fmt.Errorf("生成表 %s 的查询代码失败: %v", table.Name, err)
		}
	}

	return nil
}

// 数据库连接获取表结构信息
func connectDB(cfg *config.Config) ([]*config.TableInfo, error) {
	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	fmt.Println("连接数据库成功")

	var tableNames []string
	if len(cfg.Tables) == 0 {
		// 如果未指定表名，则获取所有表
		var tables []string
		if err := db.Raw("SHOW TABLES").Scan(&tables).Error; err != nil {
			return nil, fmt.Errorf("获取所有表名失败: %v", err)
		}
		tableNames = tables
		fmt.Printf("未指定表名，将生成所有表(%d个)的代码\n", len(tableNames))
	} else {
		tableNames = cfg.Tables
		fmt.Printf("将生成指定的%d个表的代码\n", len(tableNames))
	}

	var tableInfos []*config.TableInfo

	// 遍历处理每个表
	for _, tableName := range tableNames {
		if tableName == "" {
			continue
		}

		// 获取表信息
		tableInfo := &config.TableInfo{
			Name: tableName,
		}

		// 获取表注释
		var tableComment string
		row := db.Raw("SELECT table_comment FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Row()
		if err := row.Scan(&tableComment); err != nil {
			return nil, fmt.Errorf("获取表 %s 的注释失败: %v", tableName, err)
		}
		tableInfo.Comment = tableComment

		// 获取列信息
		type columnInfo struct {
			ColumnName    string `gorm:"column:COLUMN_NAME"`
			DataType      string `gorm:"column:DATA_TYPE"`
			ColumnType    string `gorm:"column:COLUMN_TYPE"`
			IsNullable    string `gorm:"column:IS_NULLABLE"`
			ColumnKey     string `gorm:"column:COLUMN_KEY"`
			ColumnDefault string `gorm:"column:COLUMN_DEFAULT"`
			Extra         string `gorm:"column:EXTRA"`
			ColumnComment string `gorm:"column:COLUMN_COMMENT"`
		}

		var columns []columnInfo
		if err := db.Raw(`SELECT 
			COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, 
			COLUMN_DEFAULT, EXTRA, COLUMN_COMMENT
		FROM information_schema.columns 
		WHERE table_schema = DATABASE() 
		AND table_name = ? 
		ORDER BY ORDINAL_POSITION`, tableName).Scan(&columns).Error; err != nil {
			return nil, fmt.Errorf("获取表 %s 的字段信息失败: %v", tableName, err)
		}

		// 处理列信息
		for _, col := range columns {
			// 处理字段类型
			fieldType := utils.GetGoType(col.DataType)
			if col.IsNullable == "YES" {
				fieldType = "*" + fieldType
			}

			field := config.FieldInfo{
				Name:       col.ColumnName,
				Type:       fieldType,
				Comment:    col.ColumnComment,
				IsNullable: col.IsNullable == "YES",
				IsPrimary:  col.ColumnKey == "PRI",
				Tag:        utils.BuildFieldTags(col.ColumnName, col.ColumnType, col.IsNullable == "YES"),
				ColumnType: col.ColumnType,
			}
			tableInfo.Fields = append(tableInfo.Fields, field)

			// 如果是主键，添加到索引信息中
			if field.IsPrimary {
				tableInfo.Indexes = append(tableInfo.Indexes, config.IndexInfo{
					Name:   "PRIMARY",
					Fields: []string{field.Name},
					IsPK:   true,
					IsUniq: true,
				})
			}
		}

		// 获取索引信息（非主键）
		var indexes []struct {
			IndexName string `gorm:"column:INDEX_NAME"`
			NonUnique int    `gorm:"column:NON_UNIQUE"`
			ColName   string `gorm:"column:COLUMN_NAME"`
		}
		if err := db.Raw(`SELECT 
			INDEX_NAME, NON_UNIQUE, COLUMN_NAME
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = ?
		AND INDEX_NAME != 'PRIMARY'
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`, tableName).Scan(&indexes).Error; err != nil {
			return nil, fmt.Errorf("获取表 %s 的索引信息失败: %v", tableName, err)
		}

		// 处理索引信息
		indexMap := make(map[string]*config.IndexInfo)
		for _, idx := range indexes {
			if index, ok := indexMap[idx.IndexName]; ok {
				index.Fields = append(index.Fields, idx.ColName)
			} else {
				indexMap[idx.IndexName] = &config.IndexInfo{
					Name:   idx.IndexName,
					Fields: []string{idx.ColName},
					IsUniq: idx.NonUnique == 0,
				}
			}
		}
		for _, index := range indexMap {
			tableInfo.Indexes = append(tableInfo.Indexes, *index)
		}

		// 尝试从配置中获取关联关系
		if relations, ok := cfg.Relations[tableName]; ok {
			// fmt.Printf("从配置中读取到表 %s 的关联关系: %d 个\n", tableName, len(relations))
			for _, rel := range relations {
				relationInfo := config.RelationInfo{
					Name:           rel.Target,
					Type:           rel.Type,
					Model:          rel.Target,
					ForeignKey:     rel.ForeignKey,
					References:     rel.References,
					JoinTable:      rel.JoinTable,
					JoinForeignKey: rel.JoinForeignKey,
					JoinReferences: rel.JoinReferences,
					Comment:        rel.Comment,
				}
				tableInfo.Relations = append(tableInfo.Relations, relationInfo)
			}
		} else {
			// 尝试从数据库中推断关联关系
			// TODO: 根据外键约束推断关联关系
			// 1. 查询外键约束
			// 2. 根据外键约束推断关联类型
			// 3. 生成关联关系信息
		}

		tableInfos = append(tableInfos, tableInfo)
	}

	return tableInfos, nil
}
