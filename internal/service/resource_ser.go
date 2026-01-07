package service

import (
	"CMS/internal/dto"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CMS/internal/model"
	"CMS/internal/repository"
)

type ResourceService interface {
	GetResourceList(ctx context.Context, page, size int, keyword string) ([]dto.ResourceItem, int64, error)
	CreateResource(ctx context.Context, userID uint64, title, text, code string) error
	GetResourceByID(ctx context.Context, id uint64) (*model.Resource, error)
	IncrViewCount(ctx context.Context, id uint64) error
	IncrLikeCount(ctx context.Context, id uint64) error                 // 增加点赞数
	CreateComment(ctx context.Context, id uint64, content string) error // 创建评论（并增加评论数）

}

type ResourceServiceImpl struct {
	resourceRepo repository.ResourceRepo
	userRepo     repository.UserRepo // 用于查询username
	accRepo      repository.AccountRepo
}

func NewResourceService(resourceRepo repository.ResourceRepo, userRepo repository.UserRepo, accRepo repository.AccountRepo) ResourceService {
	return &ResourceServiceImpl{
		resourceRepo: resourceRepo,
		userRepo:     userRepo,
		accRepo:      accRepo,
	}
}

// CreateResource 新增资源（业务逻辑层）- 新增：设置点赞/浏览/评论量默认值0
func (s *ResourceServiceImpl) CreateResource(ctx context.Context, userID uint64, title, text, code string) error {
	// ========== 步骤1：基础参数校验（service层必须做，避免脏数据进入仓库） ==========
	// 1.1 校验用户ID合法性
	if userID == 0 {
		return errors.New("用户ID不能为空（userID=0）")
	}
	// 1.2 校验标题合法性（非空 + 长度限制，根据业务调整）
	title = strings.TrimSpace(title) // 去除首尾空格
	if title == "" {
		return errors.New("资源标题不能为空")
	}
	if len(title) > 100 { // 假设业务规则：标题最长100字符
		return fmt.Errorf("资源标题长度不能超过100字符（当前长度：%d）", len(title))
	}

	// ========== 步骤2：查询用户信息并校验 ==========
	user, err := s.userRepo.GetUserById(ctx, userID)
	if err != nil {
		// 使用%w包装原始错误，上层可通过errors.Is/As判断根因
		return fmt.Errorf("查询用户失败（userID=%d）：%w", userID, err)
	}
	// 2.1 校验用户是否存在（避免空指针）
	if user == nil {
		return fmt.Errorf("用户不存在（userID=%d）", userID)
	}
	// 2.2 校验用户名是否有效（避免冗余存储空值）
	if strings.TrimSpace(user.Username) == "" {
		return fmt.Errorf("用户（userID=%d）的用户名不能为空", userID)
	}

	// ========== 步骤3：构造资源模型 ==========
	resource := &model.Resource{
		UserID:       userID,
		Title:        title,                   // 使用去空格后的标题
		TextContent:  strings.TrimSpace(text), // 可选：文本内容去空格
		CodeContent:  code,                    // 代码内容保留原始格式（不建议去空格）
		Author:       user.Username,           // 冗余存储用户名
		PublishTime:  time.Now(),              // 发布时间取当前时间
		LikeCount:    0,                       // 新增：点赞量默认值0
		ViewCount:    0,                       // 新增：浏览量默认值0
		CommentCount: 0,                       // 新增：评论量默认值0
	}

	// ========== 步骤4：调用仓库层插入数据 ==========
	if err := s.resourceRepo.CreateResource(ctx, nil, resource); err != nil {
		return fmt.Errorf("创建资源失败（title=%s, userID=%d）：%w", title, userID, err)
	}

	return nil
}

// convertModelToDTO model转DTO - 新增：映射点赞/浏览/评论量字段
func convertModelToDTO(res *model.Resource) dto.ResourceItem {
	// 仅需处理时间格式转换（time.Time → 字符串）
	publishTime := res.PublishTime.Format("2006-01-02 15:04:05")

	// 字段一一精准映射（你的model和DTO字段完全对齐）
	return dto.ResourceItem{
		ID:           res.ID,           // 资源主键ID
		Title:        res.Title,        // 资源标题
		TextContent:  res.TextContent,  // 文本内容（string类型直接赋值）
		CodeContent:  res.CodeContent,  // 代码内容（string类型直接赋值）
		Author:       res.Author,       // 作者名
		PublishTime:  publishTime,      // 格式化后的发布时间
		UserID:       res.UserID,       // 关联的用户ID
		LikeCount:    res.LikeCount,    // 新增：映射点赞量
		ViewCount:    res.ViewCount,    // 新增：映射浏览量
		CommentCount: res.CommentCount, // 新增：映射评论量
	}
}

// GetResourceList 分页查询资源列表（最终无类型错误版）
func (s *ResourceServiceImpl) GetResourceList(ctx context.Context, page, size int, keyword string) ([]dto.ResourceItem, int64, error) {
	// 1. 分页参数校验
	if page < 1 {
		return nil, 0, fmt.Errorf("页码必须≥1，当前值：%d", page)
	}
	if size < 1 || size > 50 {
		return nil, 0, fmt.Errorf("每页条数必须在1~50之间，当前值：%d", size)
	}

	// 2. 计算分页偏移量（原生SQL的LIMIT offset, limit）
	offset := (page - 1) * size

	// 3. 调用Repo层统计总条数
	total, err := s.resourceRepo.CountResources(ctx, keyword)
	if err != nil {
		return nil, 0, fmt.Errorf("统计资源总数失败：%w", err)
	}

	// 4. 总数为0时返回空DTO切片
	if total == 0 {
		return []dto.ResourceItem{}, 0, nil
	}

	// 5. 调用Repo层查询原生model切片（[]*model.Resource）
	resourceModels, err := s.resourceRepo.GetResourceList(ctx, offset, size, keyword)
	if err != nil {
		return nil, 0, fmt.Errorf("查询资源列表失败：%w", err)
	}

	// 6. 核心：遍历转换model → DTO（解决类型不匹配的关键）
	var resourceDTOs []dto.ResourceItem
	for _, resModel := range resourceModels {
		dtoItem := convertModelToDTO(resModel) // 逐个转换（已包含新字段）
		resourceDTOs = append(resourceDTOs, dtoItem)
	}

	// 7. 返回DTO切片（完全匹配Handler层预期类型）+ 总条数
	return resourceDTOs, total, nil
}

// GetResourceByID 根据ID查询单条资源详情 - 无需修改（repo层已返回新字段，直接透传）
func (s *ResourceServiceImpl) GetResourceByID(ctx context.Context, id uint64) (*model.Resource, error) {
	// 1. 参数校验（service层必须做，避免非法参数进入repo）
	if id <= 0 {
		return nil, fmt.Errorf("资源ID必须为正整数（当前值：%d）", id)
	}

	// 2. 调用仓库层查询资源
	resource, err := s.resourceRepo.GetResourceByID(ctx, id)
	if err != nil {
		// 包装错误，保留根因（上层可通过errors.Is判断）
		return nil, fmt.Errorf("查询资源失败（id=%d）：%w", id, err)
	}

	// 3. 资源不存在时返回nil（无错误），由handler层处理404
	return resource, nil
}
func (s *ResourceServiceImpl) IncrViewCount(ctx context.Context, id uint64) error {
	// 调用Repo层的原子更新方法
	err := s.resourceRepo.IncrViewCount(ctx, id)
	if err != nil {
		return fmt.Errorf("更新浏览量失败：%w", err)
	}
	return nil
}

// IncrLikeCount 增加点赞数（原子更新，和浏览量逻辑一致）
func (s *ResourceServiceImpl) IncrLikeCount(ctx context.Context, id uint64) error {
	if id <= 0 {
		return fmt.Errorf("资源ID无效（id=%d）", id)
	}
	// 调用Repo层的原子更新点赞数方法（需确保Repo已实现）
	err := s.resourceRepo.IncrLikeCount(ctx, id)
	if err != nil {
		return fmt.Errorf("更新点赞数失败：%w", err)
	}
	return nil
}

// CreateComment 创建评论（业务逻辑：新增评论记录 + 原子更新评论数）
func (s *ResourceServiceImpl) CreateComment(ctx context.Context, id uint64, content string) error {
	// 参数校验
	if id <= 0 {
		return fmt.Errorf("资源ID无效（id=%d）", id)
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("评论内容不能为空")
	}

	// 1. 新增评论记录（需确保有Comment模型和Repo）
	// comment := &model.Comment{
	// 	ResourceID: id,
	// 	Content:    content,
	// 	CreateTime: time.Now(),
	// 	// 可补充：UserID（评论用户ID）、Username（评论用户名）等字段
	// }
	// if err := s.commentRepo.CreateComment(ctx, comment); err != nil {
	// 	return fmt.Errorf("创建评论记录失败：%w", err)
	// }

	// 2. 原子更新资源的评论数
	if err := s.resourceRepo.IncrCommentCount(ctx, id); err != nil {
		return fmt.Errorf("更新评论数失败：%w", err)
	}
	return nil
}
