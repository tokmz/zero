package config

// Config 配置结构体
type Config struct {
	DSN           string                `yaml:"dsn"`
	Output        OutputConfig          `yaml:"output"`
	Tables        []string              `yaml:"tables"`
	Prefix        string                `yaml:"prefix"`
	Style         string                `yaml:"style"`
	Template      string                `yaml:"template"`
	Relations     map[string][]Relation `yaml:"relations"`
	ModuleName    string                `yaml:"module_name" mapstructure:"module_name"`
	EnableTracing bool                  `yaml:"enable_tracing" mapstructure:"enable_tracing"` // 是否启用链路追踪
}

// OutputConfig 输出目录配置
type OutputConfig struct {
	OrmDir   string `yaml:"orm_dir"`   // orm代码根目录
	ModelDir string `yaml:"model_dir"` // model代码生成目录
	QueryDir string `yaml:"query_dir"` // query代码生成目录
}

// Relation 表关联关系配置
type Relation struct {
	Target         string `yaml:"target"`                                           // 目标表
	Type           string `yaml:"type"`                                             // 关联类型: has_one, has_many, belongs_to, many2many
	ForeignKey     string `yaml:"foreign_key" mapstructure:"foreign_key"`           // 外键
	References     string `yaml:"references" mapstructure:"references"`             // 引用键
	JoinTable      string `yaml:"join_table" mapstructure:"join_table"`             // 连接表（多对多关系）
	JoinForeignKey string `yaml:"join_foreign_key" mapstructure:"join_foreign_key"` // 连接表外键（多对多关系）
	JoinReferences string `yaml:"join_references" mapstructure:"join_references"`   // 连接表引用键（多对多关系）
	Comment        string `yaml:"comment" mapstructure:"comment"`                   // 关联关系注释
}

// TableInfo 表信息
type TableInfo struct {
	Name      string         // 表名
	Comment   string         // 表注释
	Fields    []FieldInfo    // 字段列表
	Indexes   []IndexInfo    // 索引列表
	Relations []RelationInfo // 关联关系
	Package   string         // 包名
}

// RelationInfo 关联关系信息
type RelationInfo struct {
	Name           string // 关联名称
	Type           string // 关联类型: HasOne, HasMany, BelongsTo, ManyToMany
	Model          string // 关联模型名称
	ForeignKey     string // 外键
	References     string // 引用键
	JoinTable      string // 连接表（多对多关系）
	JoinForeignKey string // 连接表外键（多对多关系）
	JoinReferences string // 连接表引用键（多对多关系）
	Comment        string // 关联关系注释
}

// Relations 关联关系集合
type Relations struct {
	HasMany    []HasManyRelation    // 一对多关系
	HasOne     []HasOneRelation     // 一对一关系
	BelongsTo  []BelongsToRelation  // 从属关系
	ManyToMany []ManyToManyRelation // 多对多关系
}

// HasManyRelation 一对多关系
type HasManyRelation struct {
	Table      string // 关联表名
	ForeignKey string // 外键
	References string // 引用键
}

// HasOneRelation 一对一关系
type HasOneRelation struct {
	Table      string // 关联表名
	ForeignKey string // 外键
	References string // 引用键
}

// BelongsToRelation 从属关系
type BelongsToRelation struct {
	Table      string // 关联表名
	ForeignKey string // 外键
	References string // 引用键
}

// ManyToManyRelation 多对多关系
type ManyToManyRelation struct {
	Table          string // 关联表名
	JoinTable      string // 连接表
	JoinForeignKey string // 连接表外键
	References     string // 引用键
	JoinReferences string // 连接表引用键
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name       string // 字段名
	Type       string // 字段类型
	Comment    string // 字段注释
	Tag        string // 结构体标签
	IsNullable bool   // 是否可为空
	IsPrimary  bool   // 是否是主键
	ColumnType string // 数据库列类型
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name   string   // 索引名
	Fields []string // 索引字段
	IsPK   bool     // 是否是主键
	IsUniq bool     // 是否是唯一索引
}

// GenerateOptions 代码生成的配置选项
type GenerateOptions struct {
	DSN       string                // 数据库连接字符串
	Dir       string                // 输出目录
	Tables    []string              // 要生成的表名列表
	Prefix    string                // 表名前缀
	Template  string                // 自定义模板路径
	Style     string                // 文件命名风格: snake(下划线), camel(小驼峰), pascal(大驼峰)
	Relations map[string][]Relation // 关联关系配置
}
