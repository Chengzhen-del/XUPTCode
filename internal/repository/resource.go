package repository

import (
	"CMS/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// 扩展ResourceRepo接口，新增分页查询+总数统计方法（与原有方法风格一致）
type ResourceRepo interface {
	// CreateResource 新增资源（支持传入事务，保证原子性）
	CreateResource(ctx context.Context, tx *sql.Tx, resource *model.Resource) error
	// GetByUserID 根据用户ID查询资源列表
	GetByUserID(ctx context.Context, userID uint64) ([]*model.Resource, error)
	// GetResourceList 分页查询资源列表（支持关键词模糊搜索）
	// offset: 分页偏移量, limit: 每页条数, keyword: 搜索关键词
	GetResourceList(ctx context.Context, offset, limit int, keyword string) ([]*model.Resource, error)
	// CountResources 统计资源总数（支持关键词过滤）
	CountResources(ctx context.Context, keyword string) (int64, error)
	GetResourceByID(ctx context.Context, id uint64) (*model.Resource, error)
	// 可选扩展：新增计数更新方法（如需实现点赞/浏览/评论量+1）
	IncrViewCount(ctx context.Context, id uint64) error
	IncrLikeCount(ctx context.Context, id uint64) error
	IncrCommentCount(ctx context.Context, id uint64) error
}

// resourceRepoImpl ResourceRepo实现（复用db连接，与accountRepoImpl结构一致）
type resourceRepoImpl struct {
	db *sql.DB // 复用数据库连接，无需新增连接
}

// NewResourceRepo 创建ResourceRepo实例（工厂方法，与NewAccountRepo风格一致）
func NewResourceRepo(db *sql.DB) ResourceRepo {
	return &resourceRepoImpl{db: db}
}

// CreateResource 新增资源（核心：支持外部事务，兼容单独创建场景）- 新增：插入点赞/浏览/评论量字段
func (r *resourceRepoImpl) CreateResource(ctx context.Context, tx *sql.Tx, resource *model.Resource) error {
	// 适配外部事务：有tx用tx执行，无tx用db执行（和CreateAccount逻辑一致）
	execFunc := r.db.ExecContext
	if tx != nil {
		execFunc = tx.ExecContext
	}

	// 插入资源SQL（新增：like_count, view_count, comment_count字段）
	sqlStr := `
	INSERT INTO resources (user_id, title, text_content, code_content, author, publish_time, like_count, view_count, comment_count)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := execFunc(ctx, sqlStr,
		resource.UserID,
		resource.Title,
		resource.TextContent,
		resource.CodeContent,
		resource.Author,
		resource.PublishTime,
		resource.LikeCount,    // 新增：点赞量
		resource.ViewCount,    // 新增：浏览量
		resource.CommentCount, // 新增：评论量
	)
	if err != nil {
		// 处理MySQL特定错误（和AccountRepo错误处理风格一致）
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062: // 唯一索引冲突（如title+user_id唯一索引）
				return fmt.Errorf("资源唯一索引冲突：%s", mysqlErr.Message)
			case 1048: // 非空约束（如user_id/title为空）
				return fmt.Errorf("资源必填字段为空：%s", mysqlErr.Message)
			case 1452: // 外键约束失败（user_id不存在于users表）
				return fmt.Errorf("关联的用户ID不存在：%s", mysqlErr.Message)
			}
		}
		return fmt.Errorf("插入资源失败：%w", err)
	}
	return nil
}

// GetByUserID 根据用户ID查询资源列表（原生SQL查询多行，处理结果集）- 新增：查询/扫描点赞/浏览/评论量
func (r *resourceRepoImpl) GetByUserID(ctx context.Context, userID uint64) ([]*model.Resource, error) {
	// 查询SQL：新增like_count, view_count, comment_count字段
	sqlStr := `
	SELECT id, user_id, title, text_content, code_content, author, publish_time, like_count, view_count, comment_count
	FROM resources
	WHERE user_id = ?
	ORDER BY publish_time DESC
	`
	// 执行查询（使用QueryContext，带Context）
	rows, err := r.db.QueryContext(ctx, sqlStr, userID)
	if err != nil {
		return nil, fmt.Errorf("查询资源列表失败：%w", err)
	}
	defer rows.Close() // 必须关闭rows，避免资源泄漏（核心！）

	// 遍历结果集，封装为model.Resource切片
	var resources []*model.Resource
	for rows.Next() {
		var res model.Resource
		// 修正：text_content/code_content为NullString（和GetResourceList保持一致）
		var textContent, codeContent sql.NullString
		// 扫描行数据到model（新增：like_count, view_count, comment_count）
		err := rows.Scan(
			&res.ID,
			&res.UserID,
			&res.Title,
			&textContent, // 处理NULL值
			&codeContent, // 处理NULL值
			&res.Author,
			&res.PublishTime,
			&res.LikeCount,    // 新增：点赞量
			&res.ViewCount,    // 新增：浏览量
			&res.CommentCount, // 新增：评论量
		)
		if err != nil {
			return nil, fmt.Errorf("扫描资源数据失败：%w", err)
		}
		// 转换NULL值为普通字符串
		res.TextContent = textContent.String
		res.CodeContent = codeContent.String
		resources = append(resources, &res)
	}

	// 检查遍历过程中的错误（如网络中断）
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历资源结果集失败：%w", err)
	}

	return resources, nil
}

// GetResourceList 分页查询资源列表（原生SQL实现，支持关键词模糊搜索）- 新增：查询/扫描点赞/浏览/评论量
func (r *resourceRepoImpl) GetResourceList(ctx context.Context, offset, limit int, keyword string) ([]*model.Resource, error) {
	// 1. 构建基础SQL（新增：like_count, view_count, comment_count字段）
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`
	SELECT id, user_id, title, text_content, code_content, author, publish_time, like_count, view_count, comment_count
	FROM resources
	`)

	// 2. 构建WHERE条件（处理关键词模糊搜索）
	var args []interface{}
	if keyword != "" {
		sqlBuilder.WriteString(`
		WHERE title LIKE ? OR IFNULL(text_content, '') LIKE ? OR IFNULL(code_content, '') LIKE ?
		`)
		// 关键词拼接%%（模糊匹配），用?占位符防SQL注入（核心！）
		likeKeyword := fmt.Sprintf("%%%s%%", keyword)
		args = append(args, likeKeyword, likeKeyword, likeKeyword)
	}

	// 3. 排序+分页（按发布时间倒序，LIMIT offset, limit）
	sqlBuilder.WriteString(`
	ORDER BY publish_time DESC
	LIMIT ?, ?
	`)
	args = append(args, offset, limit)

	// 4. 执行原生SQL查询
	rows, err := r.db.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		// 处理MySQL特定错误（与CreateResource错误处理风格一致）
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return nil, fmt.Errorf("分页查询资源失败：MySQL错误[%d] %s", mysqlErr.Number, mysqlErr.Message)
		}
		return nil, fmt.Errorf("分页查询资源失败：%w", err)
	}
	defer rows.Close() // 必须关闭rows，避免资源泄漏

	// 5. 遍历结果集，封装为model.Resource切片
	var resources []*model.Resource
	for rows.Next() {
		var res model.Resource
		// 临时变量接收NULL值
		var textContent, codeContent sql.NullString
		err := rows.Scan(
			&res.ID,
			&res.UserID,
			&res.Title,
			&textContent, // 先扫到NullString
			&codeContent, // 先扫到NullString
			&res.Author,
			&res.PublishTime,
			&res.LikeCount,    // 新增：点赞量
			&res.ViewCount,    // 新增：浏览量
			&res.CommentCount, // 新增：评论量
		)
		if err != nil {
			return nil, fmt.Errorf("扫描资源数据失败：%w", err)
		}
		// 转换为string（NULL→空字符串）
		res.TextContent = textContent.String
		res.CodeContent = codeContent.String
		resources = append(resources, &res)
	}

	// 6. 检查遍历过程中的错误（如网络中断）
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分页资源结果集失败：%w", err)
	}

	return resources, nil
}

// CountResources 统计资源总数（原生SQL，与GetResourceList过滤条件一致）- 无需修改
func (r *resourceRepoImpl) CountResources(ctx context.Context, keyword string) (int64, error) {
	// 1. 构建统计SQL
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`SELECT COUNT(*) FROM resources`)

	// 2. 处理关键词过滤条件
	var args []interface{}
	if keyword != "" {
		sqlBuilder.WriteString(`
		WHERE title LIKE ? OR IFNULL(text_content, '') LIKE ? OR IFNULL(code_content, '') LIKE ?
		`)
		likeKeyword := fmt.Sprintf("%%%s%%", keyword)
		args = append(args, likeKeyword, likeKeyword, likeKeyword)
	}

	// 3. 执行统计查询
	var total int64
	err := r.db.QueryRowContext(ctx, sqlBuilder.String(), args...).Scan(&total)
	if err != nil {
		// 处理MySQL特定错误
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return 0, fmt.Errorf("统计资源总数失败：MySQL错误[%d] %s", mysqlErr.Number, mysqlErr.Message)
		}
		return 0, fmt.Errorf("统计资源总数失败：%w", err)
	}

	return total, nil
}

// GetResourceByID 根据ID查询单条资源详情（纯原生SQL，适配*sql.DB）- 新增：查询/扫描点赞/浏览/评论量
func (r *resourceRepoImpl) GetResourceByID(ctx context.Context, id uint64) (*model.Resource, error) {
	// 1. 定义原生SQL（新增：like_count, view_count, comment_count字段）
	sqlStr := `
		SELECT id, user_id, title, text_content, code_content, author, publish_time, like_count, view_count, comment_count
		FROM resources 
		WHERE id = ? 
		LIMIT 1
	`

	// 2. 执行单行查询（QueryRowContext适配*sql.DB，带上下文）
	row := r.db.QueryRowContext(ctx, sqlStr, id)

	// 3. 定义变量接收结果（处理NULL值，和GetResourceList一致）
	var res model.Resource
	var textContent, codeContent sql.NullString // 处理可能为NULL的字段

	// 4. 扫描结果到变量（新增：like_count, view_count, comment_count）
	err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.Title,
		&textContent, // 先扫到NullString（处理NULL）
		&codeContent, // 先扫到NullString（处理NULL）
		&res.Author,
		&res.PublishTime,
		&res.LikeCount,    // 新增：点赞量
		&res.ViewCount,    // 新增：浏览量
		&res.CommentCount, // 新增：评论量
	)

	// 5. 错误处理（适配*sql.DB的错误类型，和现有逻辑一致）
	if err != nil {
		// 资源不存在（sql.ErrNoRows）：返回nil（无错误），由上层处理404
		if err == sql.ErrNoRows {
			return nil, nil
		}
		// 其他数据库错误（如连接失败、字段不匹配）：包装错误返回
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return nil, fmt.Errorf("查询资源失败：MySQL错误[%d] %s", mysqlErr.Number, mysqlErr.Message)
		}
		return nil, fmt.Errorf("查询资源失败（id=%d）：%w", id, err)
	}

	// 6. 转换NULL值为普通字符串（NULL→空字符串，和GetResourceList一致）
	res.TextContent = textContent.String
	res.CodeContent = codeContent.String

	// 7. 返回资源指针
	return &res, nil
}

// ========== 可选扩展：计数更新方法（实现点赞/浏览/评论量+1） ==========
// IncrViewCount 浏览量+1（原子更新，避免并发问题）
func (r *resourceRepoImpl) IncrViewCount(ctx context.Context, id uint64) error {
	sqlStr := `
		UPDATE resources 
		SET view_count = view_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, sqlStr, id)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return fmt.Errorf("更新浏览量失败：MySQL错误[%d] %s", mysqlErr.Number, mysqlErr.Message)
		}
		return fmt.Errorf("更新浏览量失败（id=%d）：%w", id, err)
	}
	return nil
}

// IncrLikeCount 点赞量+1
func (r *resourceRepoImpl) IncrLikeCount(ctx context.Context, id uint64) error {
	sqlStr := `
		UPDATE resources 
		SET like_count = like_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, sqlStr, id)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return fmt.Errorf("更新点赞量失败：MySQL错误[%d] %s", mysqlErr.Number, mysqlErr.Message)
		}
		return fmt.Errorf("更新点赞量失败（id=%d）：%w", id, err)
	}
	return nil
}

// IncrCommentCount 评论量+1
func (r *resourceRepoImpl) IncrCommentCount(ctx context.Context, id uint64) error {
	sqlStr := `
		UPDATE resources 
		SET comment_count = comment_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, sqlStr, id)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return fmt.Errorf("更新评论量失败：MySQL错误[%d] %s", mysqlErr.Number, mysqlErr.Message)
		}
		return fmt.Errorf("更新评论量失败（id=%d）：%w", id, err)
	}
	return nil
}
