// Package model 定义数据库映射模型和接口响应结构体
// 该包包含与数据库表直接映射的GORM模型，以及适配前后端交互的通用响应结构体
// 所有结构体的json标签与前端字段完全对齐，example标签为Swagger文档提供示例值
package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// User 数据库`users`表映射模型（用户核心信息表）
// @Description 存储用户的基础信息，包含登录标识、个人信息等，适配DATE字段NULL值场景
type User struct {
	ID           uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id" example:"10001"`                                // 用户主键ID（自增）
	UUID         string         `gorm:"column:uuid;uniqueIndex;not null" json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户唯一标识（UUID，用于外部交互）
	Username     string         `gorm:"column:username;uniqueIndex;not null" json:"username" example:"test_user"`                    // 用户名（唯一，登录标识）
	Email        *string        `gorm:"column:email;uniqueIndex" json:"email" example:"test@example.com"`                            // 邮箱（可选，唯一，指针类型支持NULL）
	Phone        *string        `gorm:"column:phone;uniqueIndex" json:"phone" example:"13800138000"`                                 // 手机号（可选，唯一，指针类型支持NULL）
	PasswordHash string         `gorm:"column:password_hash;not null" json:"-"`                                                      // 密码哈希值（bcrypt加密，json:"-"避免返回给前端）
	Role         string         `gorm:"column:role;default:candidate" json:"role" example:"hr"`                                      // 用户角色（默认candidate候选人，可选hr/admin）
	AvatarURL    *string        `gorm:"column:avatar_url" json:"avatar_url" example:"https://example.com/avatar.jpg"`                // 头像URL（可选，指针类型支持NULL）
	RealName     *string        `gorm:"column:real_name" json:"real_name" example:"张三"`                                              // 真实姓名（可选，指针类型支持NULL）
	Gender       *string        `gorm:"column:gender" json:"gender" example:"1"`                                                     // 性别（可选，0-未知/1-男/2-女，指针类型支持NULL）
	Wod          string         `json:"wod" example:""`                                                                              // 注：字段名疑似笔误（建议确认业务含义，如word/wechat_openid等）
	BirthDate    sql.NullString `gorm:"column:birth_date" json:"birth_date" example:"1990-01-01"`                                    // 出生日期（DATE类型，sql.NullString支持数据库NULL值）
}

// TableName 指定User模型对应的数据库表名
func (u *User) TableName() string {
	return "users"
}

// WordResponse 文本类接口响应结构体
// @Description 适配前端文本展示需求的响应格式，包含文本内容、日期、落款等字段
type WordResponse struct {
	Text      string `json:"text" example:"今日学习Go语言Swagger注释规范"` // 核心文本内容（映射word字段）
	Date      string `json:"date" example:"2026-01-07"`          // 日期（格式YYYY-MM-DD，可从数据库/当前时间获取）
	Signature string `json:"signature" example:"技术部"`            // 落款（默认值/数据库配置的组织名称）
}

// ApiResponse 全局统一接口响应格式
// @Description 所有接口的通用响应结构体，与前端交互格式完全匹配
type ApiResponse struct {
	Code    int         `json:"code" example:"200"`        // 业务状态码（200成功/400参数错误/401未登录/500系统错误）
	Message string      `json:"message" example:"success"` // 提示信息（成功/失败原因描述）
	Data    interface{} `json:"data"`                      // 响应数据体（成功时返回具体数据，失败时为nil）
}

// UserAccount 用户账户信息模型（关联users表）
// @Description 存储用户的账户余额、充值/消费记录等财务信息
type UserAccount struct {
	ID            uint64          `json:"id" example:"10001"`                                  // 账户记录自增主键
	UserUUID      string          `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 关联users.uuid（唯一标识用户）
	Balance       decimal.Decimal `json:"balance" example:"950.00"`                            // 账户余额（数据库DECIMAL(10,2)类型，保留2位小数）
	TotalRecharge decimal.Decimal `json:"total_recharge" example:"1000.00"`                    // 累计充值金额（所有充值记录总和）
	TotalConsume  decimal.Decimal `json:"total_consume" example:"50.00"`                       // 累计消费金额（所有消费记录总和）
}

// Resource 资源信息模型（文本/代码资源表）
// @Description 存储用户发布的文本、代码类资源信息，包含点赞、浏览、评论等统计字段
type Resource struct {
	ID           uint64    `json:"id" example:"10001"`                                              // 资源主键ID（自增）
	UserID       uint64    `json:"user_id" example:"10001"`                                         // 关联users表的主键ID（资源发布者）
	Title        string    `json:"title" example:"Go入门教程"`                                          // 资源标题（必填）
	TextContent  string    `json:"text_content" example:"Go基础语法讲解..."`                              // 文本内容（纯文本资源）
	CodeContent  string    `json:"code_content" example:"package main\nimport fmt\nfunc main() {}"` // 代码内容（代码类资源）
	Author       string    `json:"author" example:"test_user"`                                      // 作者（冗余users表的username，避免联表查询）
	PublishTime  time.Time `json:"publish_time" example:"2026-01-07T15:30:00+08:00"`                // 发布时间（RFC3339格式）
	LikeCount    uint64    `json:"like_count" example:"50"`                                         // 点赞量（数据库int unsigned类型）
	ViewCount    uint64    `json:"view_count" example:"200"`                                        // 浏览量（数据库int unsigned类型）
	CommentCount uint64    `json:"comment_count" example:"10"`                                      // 评论量（数据库int unsigned类型）
}
