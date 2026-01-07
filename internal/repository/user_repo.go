package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"CMS/internal/model"

	"github.com/go-sql-driver/mysql"
)

type UserRepo interface {
	GetUserByCredential(ctx context.Context, username, phone, email string) (*model.User, error)
	CreateUser(ctx context.Context, tx *sql.Tx, user *model.User) error
	CheckUsernameExist(ctx context.Context, username string) (bool, error)
	UpdateUser(ctx context.Context, user *model.User) error
	Getwod(ctx context.Context, uuid string) (*model.User, error)
	CheckPhoneExist(ctx context.Context, phone string) (bool, error)
	CheckEmailExist(ctx context.Context, email string) (bool, error)
	UpdateAvatar(ctx context.Context, user *model.User) error
	GetUserByUuid(ctx context.Context, uuid string) (*model.User, error)
	GetDB() *sql.DB
	GetUserById(ctx context.Context, id uint64) (*model.User, error)
	FindByemail(user *model.User) (*model.User, error)
	GetByemail(ctx context.Context, Email string) (*model.User, error)
}

type userRepoImpl struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) UserRepo {
	return &userRepoImpl{db: db}
}

// 100%匹配数据库字段+修正SQL语法
func (r *userRepoImpl) GetUserByCredential(ctx context.Context, username, phone, email string) (*model.User, error) {
	var sqlStr string
	var args []interface{}

	switch {
	// 匹配数据库字段：uuid（不是wuid）、password_hash（不是password hash）
	case username != "":
		sqlStr = `
			SELECT id, uuid, username, email, phone, password_hash, role, avatar_url, real_name, gender, birth_date
			FROM users 
			WHERE username = ? 
			LIMIT 1
		`
		args = []interface{}{username}
	case phone != "":
		sqlStr = `
			SELECT id, uuid, username, email, phone, password_hash, role, avatar_url, real_name, gender, birth_date
			FROM users 
			WHERE phone = ? 
			LIMIT 1
		`
		args = []interface{}{phone}
	case email != "":
		sqlStr = `
			SELECT id, uuid, username, email, phone, password_hash, role, avatar_url, real_name, gender, birth_date
			FROM users 
			WHERE email = ? 
			LIMIT 1
		`
		args = []interface{}{email}
	default:
		return nil, errors.New("未指定查询凭证")
	}

	// 执行查询
	row := r.db.QueryRowContext(ctx, sqlStr, args...)
	if row.Err() != nil {
		return nil, errors.New("查询用户失败：" + row.Err().Error())
	}

	// 扫描字段与数据库/Model完全对齐
	var user model.User
	err := row.Scan(
		&user.ID,
		&user.UUID, // 对应数据库uuid字段
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.PasswordHash, // 对应数据库password_hash字段
		&user.Role,
		&user.AvatarURL, // 对应数据库avatar_url字段
		&user.RealName,
		&user.Gender,
		&user.BirthDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.New("扫描用户数据失败：" + err.Error())
	}

	return &user, nil
}
func (r *userRepoImpl) GetDB() *sql.DB {
	return r.db
}

// 其余函数保持正确（已匹配数据库）
func (r *userRepoImpl) CreateUser(ctx context.Context, tx *sql.Tx, user *model.User) error {
	// 1. 处理Role默认值（兜底，兼容代码层传空的场景）
	role := user.Role
	if strings.TrimSpace(role) == "" {
		role = "candidate"
	}
	execFunc := r.db.ExecContext
	if tx != nil {
		execFunc = tx.ExecContext
	}
	// 3. 执行插入SQL（字段与model/User完全对齐）
	sqlStr := `
	INSERT INTO users (
		uuid, username, email, phone, password_hash, 
		role, avatar_url, real_name, gender, birth_date
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := execFunc(ctx, sqlStr,
		user.UUID,         // string (not null)
		user.Username,     // string (not null)
		user.Email,        // *string (NULLable)
		user.Phone,        // *string (NULLable)
		user.PasswordHash, // string (not null)
		role,              // string (默认candidate)
		user.AvatarURL,    // *string (NULLable)
		user.RealName,     // *string (NULLable)
		user.Gender,       // *string (NULLable)
		user.BirthDate,    // sql.NullString (适配BirthDate字段类型)
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062: // 唯一键冲突（uuid/username/email/phone重复）
				return fmt.Errorf("唯一约束冲突：%s", mysqlErr.Message)
			case 1048: // 非空约束（uuid/username/password_hash为空）
				return fmt.Errorf("必填字段为空：%s", mysqlErr.Message)
			case 1406: // 数据太长（字段值超过数据库定义长度）
				return fmt.Errorf("字段长度超过限制：%s", mysqlErr.Message)
			}
		}
		return fmt.Errorf("执行插入失败：%w", err)
	}

	// 4. 获取自增ID并赋值给user.ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增ID失败：%w", err)
	}
	user.ID = uint64(id) // 匹配model.User.ID的uint64类型

	return nil
}
func (r *userRepoImpl) CheckUsernameExist(ctx context.Context, username string) (bool, error) {
	var count int64
	sqlStr := "SELECT COUNT(*) FROM users WHERE username = ?"
	err := r.db.QueryRowContext(ctx, sqlStr, username).Scan(&count)
	if err != nil {
		return false, errors.New("检查用户名失败：" + err.Error())
	}
	return count > 0, nil
}

// UpdateUser 原生SQL实现更新逻辑（替代GORM的WithContext）

func (r *userRepoImpl) UpdateUser(ctx context.Context, user *model.User) error {
	// ========== 步骤1：前置校验（确保更新条件有效） ==========
	if user == nil || strings.TrimSpace(user.UUID) == "" {
		return errors.New("用户 UUID 不能为空")
	}

	// ========== 步骤2：动态拼接更新字段（仅处理有值的字段） ==========
	var setClauses []string // 存储 SET 后的字段（如 "username = ?"）
	var args []interface{}  // 存储更新参数（与 setClauses 一一对应）

	// 1. 用户名（普通 string，非空则更新）
	if strings.TrimSpace(user.Username) != "" {
		setClauses = append(setClauses, "username = ?")
		args = append(args, user.Username)
	}

	// 2. 邮箱（*string，非 nil 且内容非空则更新）
	if user.Email != nil && strings.TrimSpace(*user.Email) != "" {
		setClauses = append(setClauses, "email = ?")
		args = append(args, *user.Email)
	}

	// 3. 手机号（*string，非 nil 且内容非空则更新）
	if user.Phone != nil && strings.TrimSpace(*user.Phone) != "" {
		setClauses = append(setClauses, "phone = ?")
		args = append(args, *user.Phone)
	}

	// 4. 真实姓名（*string，非 nil 且内容非空则更新）
	if user.RealName != nil && strings.TrimSpace(*user.RealName) != "" {
		setClauses = append(setClauses, "real_name = ?")
		args = append(args, *user.RealName)
	}

	// 5. 性别（*string，非 nil 且内容非空则更新）
	if user.Gender != nil && strings.TrimSpace(*user.Gender) != "" {
		setClauses = append(setClauses, "gender = ?")
		args = append(args, *user.Gender)
	}

	// 6. 出生日期（sql.NullString，Valid 为 true 则更新）
	if user.BirthDate.Valid && strings.TrimSpace(user.BirthDate.String) != "" {
		setClauses = append(setClauses, "birth_date = ?")
		args = append(args, user.BirthDate.String)
	}

	// ========== 步骤3：无更新字段校验 ==========
	if len(setClauses) == 0 {
		return errors.New("无有效更新字段")
	}

	// ========== 步骤4：拼接 SQL（用 strings.Builder，避免 fmt.Sprintf 报错） ==========
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("UPDATE users SET" + " ")
	sqlBuilder.WriteString(strings.Join(setClauses, ", "))
	sqlBuilder.WriteString(" WHERE uuid = ? LIMIT 1") // LIMIT 1 防止全表更新
	sqlStr := sqlBuilder.String()

	// ========== 步骤5：追加 UUID 参数（更新条件） ==========
	args = append(args, user.UUID)

	// ========== 步骤6：执行数据库更新（带上下文，支持超时/取消） ==========
	result, err := r.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		// 解析 MySQL 具体错误，返回友好提示（便于 Service 层识别）
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062: // 唯一约束冲突（username/phone/email 重复）
				return fmt.Errorf("Duplicate entry: %s", mysqlErr.Message)
			case 1054: // 字段不存在（表结构与代码不匹配）
				return fmt.Errorf("字段不存在：%s", mysqlErr.Message)
			case 1146: // 表不存在
				return fmt.Errorf("users 表不存在：%s", mysqlErr.Message)
			default: // 其他数据库错误（如连接超时、权限不足）
				return fmt.Errorf("数据库执行错误（码：%d）：%s", mysqlErr.Number, mysqlErr.Message)
			}
		}
		// 非 MySQL 错误（如上下文取消）
		return fmt.Errorf("执行更新 SQL 失败：%w", err)
	}

	// ========== 步骤7：检查更新结果（确保更新到数据） ==========
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新行数失败：%w", err)
	}
	if rowsAffected == 0 {
		return errors.New("未找到该用户（UUID 不存在）或数据无变化")
	}

	return nil
}

func (r *userRepoImpl) Getwod(ctx context.Context, uuid string) (*model.User, error) {
	sqlStr := `
			SELECT word
			FROM users 
			WHERE uuid = ? 
		`
	row := r.db.QueryRowContext(ctx, sqlStr, uuid)
	if row.Err() != nil {
		return nil, row.Err()
	}
	var user model.User
	err := row.Scan(&user.Wod)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *userRepoImpl) CheckEmailExist(ctx context.Context, email string) (bool, error) {
	var count int64
	sqlStr := "SELECT COUNT(1) FROM users WHERE email = ?"
	err := r.db.QueryRowContext(ctx, sqlStr, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CheckPhoneExist 检查手机号是否存在
func (r *userRepoImpl) CheckPhoneExist(ctx context.Context, phone string) (bool, error) {
	var count int64
	sqlStr := "SELECT COUNT(1) FROM users WHERE phone = ?"
	err := r.db.QueryRowContext(ctx, sqlStr, phone).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *userRepoImpl) UpdateAvatar(ctx context.Context, user *model.User) error {
	// ========== 1. 前置校验 ==========
	if user == nil || strings.TrimSpace(user.UUID) == "" {
		return errors.New("用户UUID不能为空")
	}
	if user.AvatarURL == nil || strings.TrimSpace(*user.AvatarURL) == "" {
		return errors.New("头像URL不能为空")
	}

	// ========== 2. 拼接SQL ==========
	sqlStr := "UPDATE users SET avatar_url = ? WHERE uuid = ? LIMIT 1"

	// ========== 3. 执行更新 ==========
	result, err := r.db.ExecContext(ctx, sqlStr, *user.AvatarURL, user.UUID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return fmt.Errorf("数据库错误（码：%d）：%s", mysqlErr.Number, mysqlErr.Message)
		}
		return fmt.Errorf("执行更新头像SQL失败：%w", err)
	}

	// ========== 4. 检查更新结果 ==========
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新行数失败：%w", err)
	}
	if rowsAffected == 0 {
		return errors.New("未找到该用户（UUID不存在）或头像无变化")
	}

	return nil
}
func (r *userRepoImpl) GetUserByUuid(ctx context.Context, uuid string) (*model.User, error) {
	// 1. 显式指定列（避免SELECT * 导致列顺序不一致问题），和model.User字段一一对应
	sqlStr := `
		SELECT id, username, email, phone, role, avatar_url, real_name, gender, birth_date 
		FROM users 
		WHERE uuid = ?
	`
	// 2. 执行查询
	row := r.db.QueryRowContext(ctx, sqlStr, uuid)

	// 3. 定义临时变量（处理*string/sql.NullString）
	var (
		// 普通字段
		id       uint64
		username string
		role     string
		// 指针字符串字段：先用sql.NullString接收，再转换为*string
		email     sql.NullString
		phone     sql.NullString
		avatarURL sql.NullString
		realName  sql.NullString
		gender    sql.NullString
		// sql.NullString字段直接接收
		birthDate sql.NullString
	)

	// 4. 逐个字段传入指针（顺序必须和SELECT列完全一致）
	err := row.Scan(
		&id,
		&username,
		&email,
		&phone,
		&role,
		&avatarURL,
		&realName,
		&gender,
		&birthDate,
	)
	// 5. 错误处理（核心）
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("用户不存在") // 友好的业务错误
		}
		return nil, errors.New("查询用户失败：" + err.Error()) // 包装系统错误
	}

	// 6. 转换临时变量为model.User的特殊类型
	user := &model.User{
		ID:        id,
		Username:  username,
		Role:      role,
		BirthDate: birthDate, // 直接赋值（类型一致）
	}
	// 把sql.NullString转换为*string（Valid=true则取地址，否则为nil）
	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if realName.Valid {
		user.RealName = &realName.String
	}
	if gender.Valid {
		user.Gender = &gender.String
	}

	return user, nil
}
func (r *userRepoImpl) GetUserById(ctx context.Context, id uint64) (*model.User, error) {
	// 1. 显式指定所有字段（与数据库/Model完全对齐，避免SELECT *的坑）
	sqlStr := `
		SELECT id, uuid, username, email, phone, password_hash, role, avatar_url, real_name, gender, birth_date
		FROM users 
		WHERE id = ? 
		LIMIT 1
	`

	// 2. 执行查询（带Context，支持超时/取消）
	row := r.db.QueryRowContext(ctx, sqlStr, id)
	if row.Err() != nil {
		return nil, fmt.Errorf("查询用户失败：%w", row.Err())
	}

	// 3. 定义临时变量（处理*string/sql.NullString类型转换）
	var (
		// 基础字段（非空）
		userID       uint64
		uuid         string
		username     string
		passwordHash string
		role         string
		// 可空字段（先用sql.NullString接收，再转换为*string）
		email     sql.NullString
		phone     sql.NullString
		avatarURL sql.NullString
		realName  sql.NullString
		gender    sql.NullString
		// 日期字段（sql.NullString）
		birthDate sql.NullString
	)

	// 4. 扫描字段（顺序必须与SELECT语句完全一致！）
	err := row.Scan(
		&userID,
		&uuid,
		&username,
		&email,
		&phone,
		&passwordHash,
		&role,
		&avatarURL,
		&realName,
		&gender,
		&birthDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("用户ID不存在") // 友好的业务错误
		}
		return nil, fmt.Errorf("扫描用户数据失败：%w", err) // 包装系统错误
	}

	// 5. 转换临时变量为model.User（适配*string类型）
	user := &model.User{
		ID:           userID,
		UUID:         uuid,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		BirthDate:    birthDate, // 直接赋值（类型一致）
	}

	// 6. 处理可空字段（Valid=true则赋值*string，否则为nil）
	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if realName.Valid {
		user.RealName = &realName.String
	}
	if gender.Valid {
		user.Gender = &gender.String
	}

	return user, nil
}
func (u *userRepoImpl) FindByemail(user *model.User) (*model.User, error) {
	var user1 model.User
	findsql := fmt.Sprintf("SELECT email FROM users WHERE email = ?")
	err := u.db.QueryRow(findsql, user.Email).Scan(&user1.Email)
	if err != nil {
		return nil, err
	}
	return &user1, err
}
func (r *userRepoImpl) GetByemail(ctx context.Context, Email string) (*model.User, error) {
	sqlStr := `
		SELECT id, uuid, username, phone, password_hash, role, avatar_url, real_name, gender, birth_date
		FROM users 
		WHERE email = ? 
		LIMIT 1
	`

	// 2. 执行查询（带Context，支持超时/取消）
	row := r.db.QueryRowContext(ctx, sqlStr, Email)
	if row.Err() != nil {
		return nil, fmt.Errorf("查询用户失败：%w", row.Err())
	}

	// 3. 定义临时变量（处理*string/sql.NullString类型转换）
	var (
		// 基础字段（非空）
		userID       uint64
		uuid         string
		username     string
		passwordHash string
		role         string
		// 可空字段（先用sql.NullString接收，再转换为*string）
		email     sql.NullString
		phone     sql.NullString
		avatarURL sql.NullString
		realName  sql.NullString
		gender    sql.NullString
		// 日期字段（sql.NullString）
		birthDate sql.NullString
	)

	// 4. 扫描字段（顺序必须与SELECT语句完全一致！）
	err := row.Scan(
		&userID,
		&uuid,
		&username,
		&phone,
		&passwordHash,
		&role,
		&avatarURL,
		&realName,
		&gender,
		&birthDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("用户ID不存在") // 友好的业务错误
		}
		return nil, fmt.Errorf("扫描用户数据失败：%w", err) // 包装系统错误
	}

	// 5. 转换临时变量为model.User（适配*string类型）
	user := &model.User{
		ID:           userID,
		UUID:         uuid,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		BirthDate:    birthDate, // 直接赋值（类型一致）
	}

	// 6. 处理可空字段（Valid=true则赋值*string，否则为nil）
	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if realName.Valid {
		user.RealName = &realName.String
	}
	if gender.Valid {
		user.Gender = &gender.String
	}

	return user, nil
}
