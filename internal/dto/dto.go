package dto

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

// RegisterRequest 注册请求参数
// @Description 用户注册接口的请求参数，用户名/密码为必填，邮箱/手机号二选一（可选），其余字段为可选
type RegisterRequest struct {
	Username  string         `json:"username" binding:"required,min=3,max=50" example:"test_user"`   // 用户名（必填，3-50位）
	Password  string         `json:"password" binding:"required,min=6,max=20" example:"123456Ab"`    // 密码（必填，6-20位）
	Email     string         `json:"email" binding:"omitempty,email" example:"test@example.com"`     // 邮箱（可选，需符合邮箱格式）
	Phone     string         `json:"phone" binding:"omitempty,phone" example:"13800138000"`          // 手机号（可选，需符合11位国内手机号格式）
	Role      string         `json:"role" binding:"omitempty,oneof=candidate hr admin" example:"hr"` // 角色（可选，仅支持candidate/hr/admin）
	BirthDate sql.NullString `gorm:"column:birth_date" json:"birth_date" example:"1990-01-01"`       // 出生日期（可选，格式YYYY-MM-DD）
	RealName  string         `json:"real_name" binding:"omitempty,max=10" example:"张三"`              // 真实姓名（可选，最多10位）
	Gender    string         `json:"gender" binding:"omitempty,oneof=0 1 2" example:"1"`             // 性别（可选，0-未知/1-男/2-女）
	AvatarURL string         `json:"avatar_url" example:"https://example.com/avatar.jpg"`            // 头像URL（可选）
}

// RegisterResponse 注册响应参数
// @Description 用户注册成功后返回的信息
type RegisterResponse struct {
	UserID   uint64 `json:"user_id" example:"10001"`      // 用户主键ID
	Username string `json:"username" example:"test_user"` // 用户名
	Role     string `json:"role" example:"hr"`            // 用户角色
}

// LoginRequest 登录请求参数
// @Description 用户登录接口的请求参数，用户名/手机号/邮箱三选一 + 密码必填
type LoginRequest struct {
	Username string `json:"username" binding:"omitempty" example:"test_user"`            // 用户名（可选，与手机号/邮箱三选一）
	Phone    string `json:"phone" binding:"omitempty,phone" example:"13800138000"`       // 手机号（可选，与用户名/邮箱三选一）
	Email    string `json:"email" binding:"omitempty,email" example:"test@example.com"`  // 邮箱（可选，与用户名/手机号三选一）
	Password string `json:"password" binding:"required,min=6,max=20" example:"123456Ab"` // 密码（必填，6-20位）
}

// LoginResponse 登录响应参数
// @Description 用户登录成功后返回的信息，包含登录凭证Token
type LoginResponse struct {
	Token    string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // 登录凭证Token
	UserID   uint64 `json:"user_id" example:"10001"`                                 // 用户主键ID
	Username string `json:"username" example:"test_user"`                            // 用户名
	Role     string `json:"role" example:"hr"`                                       // 用户角色
}

// JobsRequest 职位管理请求参数
// @Description 职位新增/编辑接口的请求参数
type JobsRequest struct {
	Id      uint64 `json:"id" example:"1001"`         // 职位ID（新增时不传，编辑时必填）
	Name    string `json:"name" example:"Go开发工程师"`    // 职位名称
	Require string `json:"require" example:"熟悉Gin框架"` // 职位要求
	Need    int    `json:"need" example:"5"`          // 招聘人数
}

// UpdateUserReq 更新用户信息请求参数
// @Description 用户更新自身信息的请求参数，UUID由中间件自动填充（前端无需传）
type UpdateUserReq struct {
	UUID      string `json:"uuid" form:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户唯一标识（由中间件覆盖，前端无需传）
	Username  string `json:"username" form:"username" example:"test_user_new"`                // 用户名（可选）
	Email     string `json:"email" form:"email" example:"test_new@example.com"`               // 邮箱（可选，需符合邮箱格式）
	Phone     string `json:"phone" form:"phone" example:"13900139000"`                        // 手机号（可选，需符合11位国内手机号格式）
	RealName  string `json:"real_name" form:"real_name" example:"李四"`                         // 真实姓名（可选，最多10位）
	Gender    string `json:"gender" form:"gender" example:"2"`                                // 性别（可选，0-未知/1-男/2-女）
	BirthDate string `json:"birth_date" form:"birth_date" example:"1995-05-05"`               // 出生日期（可选，格式YYYY-MM-DD）
}

// CommonResponse 通用响应体
// @Description 接口通用返回格式，所有接口统一使用该结构体返回（除特殊定制接口）
type CommonResponse struct {
	Code int         `json:"code" example:"200"`    // 响应码：200成功/400参数错误/401未登录/404不存在/500系统错误
	Msg  string      `json:"msg" example:"success"` // 提示信息（成功/失败原因）
	Data interface{} `json:"data"`                  // 响应数据（成功时返回具体内容，失败时为nil）
}

// UpdateAvatarReq 更新头像请求参数
// @Description 用户更新头像接口的请求参数，UUID由中间件自动填充
type UpdateAvatarReq struct {
	UUID           string `json:"uuid" form:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户唯一标识（由中间件覆盖，前端无需传）
	AvatarFileName string `form:"avatar_file_name" example:"my_avatar.jpg"`                        // 头像文件名（可选，不传则使用文件原始名）
}

// RechargeRequest 账户充值请求参数
// @Description 用户账户充值接口的请求参数，UUID由中间件自动填充，金额必填且大于0
type RechargeRequest struct {
	UserUUID string          `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户UUID（由中间件覆盖，前端无需传）
	Amount   decimal.Decimal `json:"amount" binding:"required" example:"100.00"`          // 充值金额（必填，必须大于0，保留2位小数）
}

// DeductRequest 账户余额扣减请求参数
// @Description 用户账户余额扣减接口的请求参数，UUID由中间件自动填充，金额必填且大于0
type DeductRequest struct {
	UserUUID string          `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户UUID（由中间件覆盖，前端无需传）
	Amount   decimal.Decimal `json:"amount" binding:"required" example:"50.00"`           // 扣减金额（必填，必须大于0，保留2位小数）
}

// AccountResponse 账户信息响应参数
// @Description 用户账户信息查询接口返回的参数
type AccountResponse struct {
	UserUUID      string          `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户UUID
	Balance       decimal.Decimal `json:"balance" example:"950.00"`                            // 当前账户余额（保留2位小数）
	TotalRecharge decimal.Decimal `json:"total_recharge" example:"1000.00"`                    // 累计充值金额（保留2位小数）
	TotalConsume  decimal.Decimal `json:"total_consume" example:"50.00"`                       // 累计消费金额（保留2位小数）
}

// CreateResourceReq 创建资源请求参数
// @Description 用户创建文本/代码资源接口的请求参数，标题必填，其余字段可选
type CreateResourceReq struct {
	Title       string `json:"title" binding:"required,max=255" example:"Go入门教程"`               // 资源标题（必填，最多255位）
	TextContent string `json:"text_content" example:"Go基础语法讲解..."`                              // 文本内容（可选）
	CodeContent string `json:"code_content" example:"package main\nimport fmt\nfunc main() {}"` // 代码内容（可选）
	// UserID由中间件从Token解析，不接收前端传参，避免伪造
}

// ResourceResp 单个资源响应参数
// @Description 单个资源详情接口返回的参数
type ResourceResp struct {
	ID          uint64 `json:"id" example:"1001"`                               // 资源主键ID
	UserID      uint64 `json:"user_id" example:"10001"`                         // 发布者用户ID
	Title       string `json:"title" example:"Go入门教程"`                          // 资源标题
	TextContent string `json:"text_content" example:"Go基础语法..."`                // 文本内容
	CodeContent string `json:"code_content" example:"package main\nimport fmt"` // 代码内容
	Author      string `json:"author" example:"test_user"`                      // 发布者用户名
	PublishTime string `json:"publish_time" example:"2026-01-07 15:30:00"`      // 发布时间（格式YYYY-MM-DD HH:MM:SS）
}

// ResourceIDReq 资源ID请求参数
// @Description 仅包含资源ID的请求参数（用于点赞、增加浏览量等接口）
type ResourceIDReq struct {
	ID uint64 `json:"id" binding:"required,min=1" example:"1"` // 资源ID（必填，最小为1）
}

// CommentReq 评论请求参数
// @Description 提交资源评论的请求参数
type CommentReq struct {
	ID      uint64 `json:"id" binding:"required,min=1" example:"1"`             // 资源ID（必填，最小为1）
	Content string `json:"content" binding:"required,min=1" example:"这篇教程很实用！"` // 评论内容（必填，非空）
}

// ResourceListResp 资源列表响应参数
// @Description 资源列表分页查询接口返回的参数
type ResourceListResp struct {
	List  []ResourceItem `json:"list"`                // 资源列表（包含点赞/浏览/评论量）
	Total int64          `json:"total" example:"100"` // 资源总条数
	Page  int            `json:"page" example:"1"`    // 当前页码
	Size  int            `json:"size" example:"10"`   // 每页条数
}

// Response 通用响应结构体
// @Description 接口通用返回格式（替代CommonResponse的另一种格式，字段命名更贴合前端习惯）
type Response struct {
	Code    int         `json:"code" example:"200"`        // 响应码：200成功/400参数错误/401未登录/404不存在/500系统错误
	Message string      `json:"message" example:"success"` // 提示信息（成功/失败原因）
	Data    interface{} `json:"data"`                      // 响应数据（成功时返回具体内容，失败时为nil）
}

// ResourceItem 资源列表项参数
// @Description 资源列表中单个资源的展示参数（包含点赞/浏览/评论量）
type ResourceItem struct {
	ID           uint64 `json:"id" example:"1001"`                               // 资源主键ID
	Title        string `json:"title" example:"Go入门教程"`                          // 资源标题
	TextContent  string `json:"text_content" example:"Go基础语法..."`                // 文本内容
	CodeContent  string `json:"code_content" example:"package main\nimport fmt"` // 代码内容
	Author       string `json:"author" example:"test_user"`                      // 发布者用户名
	PublishTime  string `json:"publish_time" example:"2026-01-07 15:30:00"`      // 发布时间（格式YYYY-MM-DD HH:MM:SS）
	UserID       uint64 `json:"user_id" example:"10001"`                         // 发布者用户ID
	LikeCount    uint64 `json:"like_count" example:"50"`                         // 点赞量
	ViewCount    uint64 `json:"view_count" example:"200"`                        // 浏览量
	CommentCount uint64 `json:"comment_count" example:"10"`                      // 评论量
}

// ResourceListReq 资源列表查询请求参数
// @Description 资源列表分页查询接口的请求参数
type ResourceListReq struct {
	Page    int    `json:"page" binding:"gte=1" example:"1"`         // 页码（必填，最小为1）
	Size    int    `json:"size" binding:"gte=1,lte=50" example:"10"` // 每页条数（必填，1~50之间）
	Keyword string `json:"keyword" example:"Go教程"`                   // 搜索关键词（可选，模糊匹配标题/内容）
}

// UploadMdFileReq MD文件上传请求参数
// @Description Markdown文件上传接口的请求参数，UUID由中间件自动填充
type UploadMdFileReq struct {
	UUID             string `json:"uuid" form:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"` // 用户唯一标识（由中间件覆盖，前端无需传）
	MarkdownFileName string `form:"markdown_file_name" example:"demo.md"`                            // Markdown文件名（可选，不传则使用文件原始名）
}

// EmailLoginReq 邮箱登录-发送验证码请求参数
// @Description 仅传入邮箱即可发送登录验证码
type EmailLoginReq struct {
	Email string `json:"email" binding:"required,email" example:"test@example.com"` // 接收验证码的邮箱（必填，需符合邮箱格式）
}

// EmailVerifyReq 邮箱验证码验证请求参数
// @Description 传入邮箱和验证码完成验证登录
type EmailVerifyReq struct {
	Email string `json:"email" binding:"required,email" example:"test@example.com"` // 接收验证码的邮箱
	Code  string `json:"code" binding:"required,len=6" example:"123456"`            // 6位数字验证码
}
