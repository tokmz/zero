# Go-Zero Gorm Generator

## 项目简介

这是一个用于 Go-Zero 框架的代码生成器，主要功能是根据数据库表结构自动生成与 Gorm 兼容的 Model 代码和 ORM 封装。

## 功能特性

- 自动生成 Gorm 模型结构体
- 生成基础的 CRUD 操作封装
- 支持自定义模板
- 支持多种命名风格
- 支持表名前缀处理
- 与 Go-Zero 框架无缝集成

## 安装

### 方式一：直接安装

```bash
go install github.com/tokmz/zero@latest
```

### 方式二：从源码安装

```bash
# 克隆项目
git clone https://github.com/tokmz/zero.git

# 进入项目目录
cd zero

# 编译安装
go build -o zero
```

安装完成后，确保 `zero` 可执行文件在系统的 PATH 路径中。

## 模板说明

默认模板位于 `template` 目录下：

```
template/
  ├── model.tmpl  # 模型文件模板
  ├── orm.tmpl    # 数据库配置模板
  └── vars.tmpl   # 全局变量模板
```

### 模板文件说明

- `model.tmpl`: 用于生成每个表对应的模型文件
- `orm.tmpl`: 用于生成数据库配置和连接代码
- `vars.tmpl`: 用于生成全局变量和工具函数

## 使用方法

### 基本用法

```bash
# 生成所有表的模型
./zero_orm -dsn "root:123456@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local"

# 指定输出目录
./zero_orm -dsn "..." -dir ./model

# 生成指定表的模型
./zero_orm -dsn "..." -tables user,order,product

# 使用配置文件
./zero_orm -config config.yaml
```

### 配置文件示例

```yaml
# config.yaml
dsn: root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
dir: ./model
tables: # 要生成的表，为空则生成所有表
prefix: # 表名前缀
style: snake # 文件命名风格：snake/camel/pascal
template: # 自定义模板目录

# 关联关系配置
relations:
  user: # 用户表关联关系
    - target: user_login_log # 目标表
      type: has_many # 关系类型：has_many/has_one/belongs_to/many2many
      foreign_key: user_id # 外键
      references: id # 引用键
```

### 命名风格

支持三种文件命名风格：

```bash
# 下划线命名（默认）
./zero_orm -dsn "..." -style snake    # 生成 user_info.go

# 小驼峰命名
./zero_orm -dsn "..." -style camel    # 生成 userInfo.go

# 大驼峰命名
./zero_orm -dsn "..." -style pascal   # 生成 UserInfo.go
```

### 表名前缀处理

如果你的表名有统一的前缀（如：t_user, t_order），可以使用 -prefix 参数去除前缀：

```bash
./zero_orm -dsn "..." -prefix t_    # t_user 表生成为 User 结构体
```

### 自定义模板

你可以自定义模板目录来覆盖默认模板：

```bash
./zero_orm -dsn "..." -template ./custom/template

# 自定义模板目录结构
custom/template/
  ├── model.tmpl  # 自定义模型模板
  ├── orm.tmpl    # 自定义ORM模板
  └── vars.tmpl   # 自定义变量模板
```

## 生成的文件结构

```
model/
  ├── orm.go      # 数据库配置和连接
  ├── vars.go     # 全局变量和工具函数
  ├── user.go     # 用户表模型
  ├── order.go    # 订单表模型
  └── ...         # 其他表模型
```

## 参数说明

| 参数     | 说明                         | 默认值           | 示例                                   |
| -------- | ---------------------------- | ---------------- | -------------------------------------- |
| dsn      | 数据库连接字符串             | -                | "root:123456@tcp(localhost:3306)/mydb" |
| dir      | 输出目录                     | .                | ./model                                |
| tables   | 要生成的表名，多个用逗号分隔 | 空（生成所有表） | user,order                             |
| prefix   | 表名前缀，生成时会去除此前缀 | 空               | t\_                                    |
| style    | 文件命名风格                 | snake            | snake/camel/pascal                     |
| template | 自定义模板目录路径           | template         | ./custom/template                      |

## 生成的代码示例

```go
// Code generated by github.com/tokmz/zero. DO NOT EDIT.
// Generated at: 2024-03-21 15:04:05

package model

type User struct {
    ID        int64     `gorm:"column:id;primaryKey;autoIncrement;type:bigint;comment:用户ID"`
    Username  string    `gorm:"column:username;not null;type:varchar(255);comment:用户名"`
    CreatedAt time.Time `gorm:"column:created_at;not null;type:datetime;comment:创建时间"`
}

func (m *User) TableName() string {
    return "user"
}

// ... 其他方法
```
