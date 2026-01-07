package service

import (
	"CMS/internal/dto"
	"CMS/internal/model"
	"CMS/internal/pkg/jwt"
	"CMS/internal/repository"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	MaxUsernameLen = 50
	MaxEmailLen    = 100
	MaxPhoneLen    = 20
	MaxRoleLen     = 20
)
const DefaultBirthDate = "2000-01-01"

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`) // 国内手机号正则
)

// StaffService 业务接口
type StaffService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Logout(ctx context.Context, token string) error
	UpdateUser(ctx context.Context, req *dto.UpdateUserReq) error
	GetWordText(ctx context.Context, uuid string) (*model.WordResponse, error)
	UpdateAvatar(ctx context.Context, file io.Reader, req *dto.UpdateAvatarReq) (string, error)
	GetUserByUuid(ctx context.Context, uuid string) (*model.User, error)
	Elogin(user *model.User) (*model.User, error)
	Verify(ctx context.Context, email string, code string) (*dto.LoginResponse, error)
}

// staffServiceImpl 实现StaffService
type staffServiceImpl struct {
	userRepo    repository.UserRepo
	useraccRepo repository.AccountRepo
	jwtCfg      pkg.JWTConfig // 改用pkg.JWTConfig
}

// NewStaffService 创建业务实例
func NewStaffService(userRepo repository.UserRepo, useraccRepo repository.AccountRepo) StaffService {
	return &staffServiceImpl{
		userRepo:    userRepo,
		useraccRepo: useraccRepo,
		jwtCfg: pkg.JWTConfig{ // 调用pkg包的结构体
			Secret: []byte("aiuegfiuewgfiuwfeiuwheqowhfoiqfiifenfeqnfeq"),
			Expire: 24 * time.Hour,
			Issuer: "cms-interview-platform",
		},
	}
}
func (s *staffServiceImpl) UpdateAvatar(ctx context.Context, file io.Reader, req *dto.UpdateAvatarReq) (string, error) {
	// ========== 步骤1：基础参数校验（兜底校验） ==========
	// 1.1 UUID非空校验（用户唯一标识）
	if strings.TrimSpace(req.UUID) == "" {
		return "", errors.New("业务校验失败：用户UUID不能为空")
	}
	// 1.2 文件流非空校验
	if file == nil {
		return "", errors.New("业务校验失败：头像文件不能为空")
	}
	// 1.3 文件名非空/格式校验
	if strings.TrimSpace(req.AvatarFileName) == "" {
		return "", errors.New("业务校验失败：头像文件名不能为空")
	}

	// ========== 修正后的后缀提取逻辑 ==========
	dotIndex := strings.LastIndex(req.AvatarFileName, ".")
	if dotIndex <= 0 || dotIndex == len(req.AvatarFileName)-1 {
		return "", errors.New("业务校验失败：头像文件无有效后缀（仅支持jpg/jpeg/png）")
	}
	ext := strings.ToLower(req.AvatarFileName[dotIndex+1:])
	allowExts := map[string]bool{"jpg": true, "jpeg": true, "png": true}
	if !allowExts[ext] {
		return "", errors.New("业务校验失败：仅支持jpg/jpeg/png格式的头像文件")
	}

	// ========== 步骤1.5：新增！查询用户原头像URL（核心补充） ==========
	// 先查询用户原有信息，获取原头像URL
	oldUser, err := s.userRepo.GetUserByUuid(ctx, req.UUID)
	if err != nil {
		// 区分「用户不存在」和「查询失败」
		if strings.Contains(err.Error(), "用户不存在") {
			return "", errors.New("业务校验失败：用户不存在")
		}
		return "", fmt.Errorf("查询用户原头像失败：%w", err)
	}
	// 提取原头像URL（注意：oldUser.AvatarURL是*string类型，需判空）
	var oldAvatarURL string
	if oldUser.AvatarURL != nil {
		oldAvatarURL = *oldUser.AvatarURL
	}

	// ========== 步骤2：调用pkg层保存新头像文件（本地/对象存储） ==========
	newAvatarURL, err := pkg.SaveAvatar(file, req.AvatarFileName)
	if err != nil {
		return "", fmt.Errorf("保存新头像失败：%w", err) // 调整错误提示更精准
	}

	// ========== 步骤2.5：新增！删除用户原头像文件（核心补充） ==========
	// 避免删除默认头像（根据你的业务规则调整）
	if oldAvatarURL != "" && oldAvatarURL != defaultAvatarURL {
		// 调用pkg层的删除文件方法（需新增该方法）
		if err := pkg.DeleteAvatar(oldAvatarURL); err != nil {
			// 注意：删除原文件失败不阻断流程（避免新头像保存成功但更新失败），仅记录日志
			log.Printf("[更新头像] 删除原头像失败 | UUID: %s | 原头像URL: %s | 错误：%v", req.UUID, oldAvatarURL, err)
			// 可选：如果业务要求必须删除原文件，可返回错误
			return "", fmt.Errorf("删除原头像失败：%w", err)
		}
	}

	// ========== 步骤3：DTO转换为Model（仅更新avatar_url字段） ==========
	userModel := &model.User{
		UUID:      req.UUID,
		AvatarURL: &newAvatarURL, // 新头像URL非空，直接取地址
	}

	// ========== 步骤4：调用Repo层更新数据库 ==========
	if err := s.userRepo.UpdateAvatar(ctx, userModel); err != nil {
		// ========== 步骤4.5：新增！数据库更新失败，回滚新头像文件（关键兜底） ==========
		// 新头像已保存但数据库更新失败，删除刚保存的新头像，避免垃圾文件
		rollbackErr := pkg.DeleteAvatar(newAvatarURL)
		if rollbackErr != nil {
			log.Printf("[更新头像] 数据库更新失败，回滚新头像失败 | UUID: %s | 新头像URL: %s | 错误：%v", req.UUID, newAvatarURL, rollbackErr)
		}
		return "", fmt.Errorf("更新头像数据失败：%w", err)
	}
	println(newAvatarURL)
	// ========== 步骤5：返回新头像访问URL ==========
	return newAvatarURL, nil
}

func (s *staffServiceImpl) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// ========== 1. 基础格式/长度校验 ==========
	// 用户名校验
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return nil, errors.New("用户名不能为空")
	}
	if len(req.Username) > MaxUsernameLen {
		return nil, fmt.Errorf("用户名长度不能超过%d个字符", MaxUsernameLen)
	}

	// 密码校验（非空）
	if strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("密码不能为空")
	}

	// Email校验（填则校验格式+长度）
	req.Email = strings.TrimSpace(req.Email)
	if req.Email != "" {
		if len(req.Email) > MaxEmailLen {
			return nil, fmt.Errorf("邮箱长度不能超过%d个字符", MaxEmailLen)
		}
		if !emailRegex.MatchString(req.Email) {
			return nil, errors.New("邮箱格式不合法")
		}
	}

	// Phone校验（填则校验格式+长度）
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone != "" {
		if len(req.Phone) > MaxPhoneLen {
			return nil, fmt.Errorf("手机号长度不能超过%d个字符", MaxPhoneLen)
		}
		if !phoneRegex.MatchString(req.Phone) {
			return nil, errors.New("手机号格式不合法（需为11位国内手机号）")
		}
	}

	// Role兜底（默认candidate）+ 长度校验
	req.Role = strings.TrimSpace(req.Role)
	if req.Role == "" {
		req.Role = "candidate"
	}
	if len(req.Role) > MaxRoleLen {
		return nil, fmt.Errorf("角色长度不能超过%d个字符", MaxRoleLen)
	}
	if req.Gender == "" {
		req.Gender = "female"
	}
	req.BirthDate = sql.NullString{
		String: DefaultBirthDate, // 默认生日字符串
		Valid:  true,             // 标记为“有值”，避免存NULL
	}
	if req.RealName == "" {
		req.RealName = "staff"
	}
	// ========== 2. 唯一性校验 ==========
	// 用户名唯一性
	exist, err := s.userRepo.CheckUsernameExist(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("查询用户名失败：%w", err)
	}
	if exist {
		return nil, errors.New("用户名已存在")
	}

	// Email唯一性（填则校验）
	if req.Email != "" {
		exist, err := s.userRepo.CheckEmailExist(ctx, req.Email)
		if err != nil {
			return nil, fmt.Errorf("查询邮箱失败：%w", err)
		}
		if exist {
			return nil, errors.New("邮箱已存在")
		}
	}

	// Phone唯一性（填则校验）
	if req.Phone != "" {
		exist, err := s.userRepo.CheckPhoneExist(ctx, req.Phone)
		if err != nil {
			return nil, fmt.Errorf("查询手机号失败：%w", err)
		}
		if exist {
			return nil, errors.New("手机号已存在")
		}
	}
	req.AvatarURL = strings.TrimSpace("https://example.com/avatar/lisi.jpg")
	// ========== 3. 密码加密 + UUID生成 ==========
	passwordHash, err := pkg.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败：%w", err)
	}

	uuidStr := pkg.GenerateUUID()
	if uuidStr == "" {
		return nil, errors.New("生成UUID失败")
	}

	// ========== 4. 构造用户模型 ==========
	newUser := &model.User{
		UUID:         uuidStr,
		Username:     req.Username,
		Email:        &req.Email,
		Phone:        &req.Phone,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Gender:       &req.Gender,
		RealName:     &req.RealName,
		AvatarURL:    &req.AvatarURL,
		BirthDate:    req.BirthDate,
	}

	db := s.userRepo.GetDB() // 需给UserRepo新增GetDB()方法，返回*sql.DB
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败：%w", err)
	}
	// 事务兜底：失败回滚，成功提交
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// ========== 5.1 执行：创建用户（传入事务） ==========
	// 需改造UserRepo的CreateUser方法，支持传入tx：CreateUserWithTx(ctx context.Context, tx *sql.Tx, user *model.User) error
	if err := s.userRepo.CreateUser(ctx, tx, newUser); err != nil {
		return nil, fmt.Errorf("创建用户失败：%w", err)
	}

	// ========== 5.2 执行：创建账户（通过AccountService，传入事务） ==========
	if err := s.useraccRepo.CreateAccount(ctx, tx, newUser.UUID); err != nil {
		return nil, fmt.Errorf("创建用户账户失败：%w", err)
	}
	// ========== 6. 返回响应（包含正确的自增ID） ==========
	return &dto.RegisterResponse{
		UserID:   newUser.ID, // 此时ID已由CreateUser赋值
		Username: newUser.Username,
		Role:     newUser.Role,
	}, nil
}

// Login 登录业务逻辑
func (s *staffServiceImpl) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	// 查询用户
	user, err := s.userRepo.GetUserByCredential(ctx, req.Username, req.Phone, req.Email)
	if err != nil {
		return nil, errors.New("查询用户失败：" + err.Error())
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 验证密码（调用pkg.CheckPassword）
	if !pkg.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("密码错误")
	}

	// 生成Token（调用pkg.GenerateToken）
	token, err := pkg.GenerateToken(user.UUID, user.Username, user.Role)
	if err != nil {
		return nil, errors.New("生成登录凭证失败：" + err.Error())
	}

	return &dto.LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

// Logout 退出登录逻辑
func (s *staffServiceImpl) Logout(ctx context.Context, token string) error {
	// 解析Token（调用pkg.ParseToken）
	_, err := pkg.ParseToken(s.jwtCfg, token)
	if err != nil {
		return errors.New("无效的登录凭证：" + err.Error())
	}
	return nil
}

var (
	dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`) // 日期格式正则（YYYY-MM-DD）
)

// UpdateUser 业务逻辑处理
type UpdateUserReq struct {
	UUID      string // 仅用于定位用户，不可更新
	Username  string
	Email     string
	Phone     string
	AvatarURL string
	RealName  string
	Gender    string
	BirthDate string
}

func (s *staffServiceImpl) UpdateUser(ctx context.Context, req *dto.UpdateUserReq) error {
	// ========== 步骤1：基础参数校验（二次校验，防止直接调用 Service 绕过 Handler） ==========
	// 1.1 UUID 不能为空（Handler 已覆盖，但防兜底）
	if strings.TrimSpace(req.UUID) == "" {
		return errors.New("业务校验失败：用户 UUID 不能为空")
	}

	// 1.2 手机号格式校验（空则跳过）
	if req.Phone != "" && !phoneRegex.MatchString(req.Phone) {
		return errors.New("业务校验失败：手机号格式错误（需为11位有效手机号，如13800138000）")
	}

	// 1.3 出生日期格式 + 合法性校验
	if req.BirthDate != "" {
		// 格式校验
		if !dateRegex.MatchString(req.BirthDate) {
			return errors.New("业务校验失败：出生日期格式错误（需为YYYY-MM-DD，如2000-01-01）")
		}
		// 合法性校验：不能晚于当前时间
		bd, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			return errors.New("业务校验失败：出生日期解析失败，非有效日期")
		}
		if bd.After(time.Now()) {
			return errors.New("业务校验失败：出生日期不能晚于当前时间")
		}
	}

	// ========== 步骤2：DTO 转换为 Model（适配 Repo 层） ==========
	userModel := &model.User{
		UUID:     req.UUID, // Handler 已覆盖为中间件的真实 UUID
		Username: req.Username,
		Email:    strToPtr(req.Email),    // 空字符串转为 nil（数据库存 NULL）
		Phone:    strToPtr(req.Phone),    // 空字符串转为 nil
		RealName: strToPtr(req.RealName), // 空字符串转为 nil
		Gender:   strToPtr(req.Gender),   // 空字符串转为 nil
		// 出生日期：空则 Valid=false（数据库存 NULL）
		BirthDate: parseBirthDate(req.BirthDate),
	}

	// ========== 步骤3：调用 Repo 层执行数据库更新 ==========
	if err := s.userRepo.UpdateUser(ctx, userModel); err != nil {
		// 处理 Repo 层返回的唯一约束错误（如用户名/手机号/邮箱重复）
		if strings.Contains(err.Error(), "Duplicate entry") {
			// 提取重复字段，返回友好提示
			if strings.Contains(err.Error(), "username") {
				return errors.New("业务校验失败：用户名已存在")
			} else if strings.Contains(err.Error(), "phone") {
				return errors.New("业务校验失败：手机号已存在")
			} else if strings.Contains(err.Error(), "email") {
				return errors.New("业务校验失败：邮箱已存在")
			}
			return errors.New("业务校验失败：用户名/手机号/邮箱已存在")
		}
		// 其他数据库错误（非业务校验错误，Handler 会返回 500）
		return fmt.Errorf("更新用户数据失败：%w", err)
	}

	return nil
}

// ---------------------- 5. 辅助函数（简化 DTO 转 Model） ----------------------
// strToPtr：空字符串转为 nil，非空转为 *string
func strToPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// parseBirthDate：解析出生日期为 sql.NullString
func parseBirthDate(s string) sql.NullString { // 需导入 "database/sql"
	if strings.TrimSpace(s) == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
func (s *staffServiceImpl) GetWordText(ctx context.Context, uuid string) (*model.WordResponse, error) {
	// 1. 调用Repo层获取原始数据

	user, err := s.userRepo.Getwod(ctx, uuid)
	if err != nil {
		return nil, err // 透传Repo层错误（上层Controller统一处理）
	}

	// 2. 无数据场景（返回nil，Controller层返回data:null）
	if user == nil {
		return nil, nil
	}

	// 3. 格式化数据（匹配前端字段）
	resp := &model.WordResponse{
		Text:      user.Wod,                        // 核心：将word映射为前端的text
		Date:      time.Now().Format("2006-01-02"), // 日期（可替换为数据库的create_time）
		Signature: "朱子墨对你说",                        // 落款（默认值，可从配置/数据库读取）
	}

	return resp, nil
}

const defaultAvatarURL = "default-avatar.png"

// UploadMdFileToStorage 保存md文件并生成访问URL（完全对齐UpdateAvatar风格）

// GetUserByUuid 实现接口方法：根据UUID查询用户
func (s *staffServiceImpl) GetUserByUuid(ctx context.Context, uuid string) (*model.User, error) {
	// ========== 1. 业务参数校验（Service层核心职责） ==========
	// 校验UUID非空（Repo层只处理数据访问，不做业务校验）
	if uuid == "" {
		return nil, errors.New("用户UUID不能为空")
	}
	// 可选：校验UUID格式（如是否符合UUIDv4规范）
	// if !isValidUUID(uuid) {
	//     return nil, errors.New("UUID格式错误")
	// }

	// ========== 2. 调用Repo层获取数据 ==========
	user, err := s.userRepo.GetUserByUuid(ctx, uuid)
	if err != nil {
		// ========== 3. 错误转换（Repo技术错误 → 业务错误） ==========
		// 区分Repo层的错误类型，返回用户友好的业务错误
		switch {
		// 匹配Repo层返回的“用户不存在”错误
		case errors.Is(err, errors.New("用户不存在")):
			return nil, errors.New("未查询到该用户")
		// 其他错误（如数据库连接失败、SQL语法错误等）→ 包装为系统错误
		default:
			return nil, errors.New("查询用户信息失败：" + err.Error())
		}
	}

	// ========== 4. 可选：业务逻辑扩展（如数据脱敏、权限校验） ==========
	// 示例1：脱敏手机号（如果手机号存在）
	if user.Phone != nil {
		phone := *user.Phone
		if len(phone) == 11 {
			maskedPhone := phone[:3] + "****" + phone[7:]
			user.Phone = &maskedPhone
		}
	}
	// 示例2：权限校验（如非管理员不能查看敏感字段）
	// if currentUser.Role != "admin" {
	//     user.RealName = nil // 隐藏真实姓名
	// }

	// ========== 5. 返回处理后的用户数据 ==========
	return user, nil
}

// 可选：UUID格式校验辅助函数（示例）
//
//	func isValidUUID(uuid string) bool {
//	    regex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
//	    return regex.MatchString(uuid)
//	}
func (u *staffServiceImpl) Elogin(user *model.User) (*model.User, error) {
	user1, err := u.userRepo.FindByemail(user)
	if err != nil {
		return nil, err
	}
	verifyCode, err := pkg.GenerateVerifyCode()
	if err != nil {
		return nil, err
	}

	err = pkg.SendVerifyCodeMail(*user1.Email, verifyCode)
	if err != nil {
		return nil, fmt.Errorf("发送验证码失败：%w", err)
	}
	return user1, nil
}
func (u *staffServiceImpl) Verify(ctx context.Context, email string, code string) (*dto.LoginResponse, error) {

	if err := pkg.VerifyCodeFromMap(email, code); err != nil {
		return nil, err
	}
	var user *model.User
	user, _ = u.userRepo.GetByemail(ctx, email)
	// 生成Token（调用pkg.GenerateToken）
	token, err := pkg.GenerateToken(user.UUID, user.Username, user.Role)
	if err != nil {
		return nil, errors.New("生成登录凭证失败：" + err.Error())
	}

	return &dto.LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
