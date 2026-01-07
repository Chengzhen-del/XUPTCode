package handler

import (
	"CMS/internal/dto"
	"CMS/internal/model"
	"CMS/internal/service"
	"errors"
	"log" // 新增：导入log包用于打印错误
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	dateRegex  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ---------------------- 统一响应结构体 ----------------------

// StaffHandler 员工/用户管理处理器
// 包含注册、登录、退出、信息更新等核心接口
type StaffHandler struct {
	svc         service.StaffService
	accsvc      service.AccountService
	resourcesvc service.ResourceService
}

// NewStaffHandler 创建用户管理处理器实例
func NewStaffHandler(svc service.StaffService, accsvc service.AccountService, resourcesvc service.ResourceService) *StaffHandler {
	return &StaffHandler{svc: svc, accsvc: accsvc, resourcesvc: resourcesvc}
}

// Register 注册接口
// @Summary 用户注册
// @Description 员工/用户注册接口，接收用户名、密码、手机号等信息完成注册
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param req body dto.RegisterRequest true "注册请求参数"
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.RegisterResponse} "注册成功"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=string} "参数校验失败"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=string} "注册业务处理失败"
// @Router /staff/register [post]
func (h *StaffHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 打印参数绑定错误原因
		log.Printf("[注册接口] 参数校验失败 | IP: %s | 错误原因: %v", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数校验失败",
			"error":   err.Error(),
		})
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		// 打印注册业务错误原因（关联用户名）
		log.Printf("[注册接口] 业务处理失败 | IP: %s | 用户名: %s | 错误原因: %v", c.ClientIP(), req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "注册失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "注册成功",
		"data":    resp,
	})
}

// Login 登录接口
// @Summary 用户登录
// @Description 支持用户名/手机号/邮箱+密码登录，返回登录凭证
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param req body dto.LoginRequest true "登录请求参数"
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.LoginResponse} "登录成功，Data返回token/登录信息"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=string} "参数校验失败/未输入登录凭证"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=string} "登录失败（账号密码错误等）"
// @Router /staff/login [post]
func (h *StaffHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 打印登录参数绑定错误原因
		log.Printf("[登录接口] 参数校验失败 | IP: %s | 错误原因: %v", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数校验失败",
			"error":   err.Error(),
		})
		return
	}

	if req.Username == "" && req.Phone == "" && req.Email == "" {
		// 该场景为用户输入问题，仅返回错误不打印日志（非系统错误）
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请输入用户名/手机号/邮箱",
		})
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		// 打印登录业务错误原因（关联登录凭证）
		log.Printf("[登录接口] 业务处理失败 | IP: %s | 登录凭证(用户名/手机号/邮箱): %s/%s/%s | 错误原因: %v",
			c.ClientIP(), req.Username, req.Phone, req.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "登录失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": resp,
	})
}

// Logout 退出接口
// @Summary 用户退出登录
// @Description 携带Bearer Token请求，销毁登录状态
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param Authorization header string true "登录凭证，格式：Bearer {token}" example(Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9)
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=nil} "退出登录成功"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "未传入登录凭证"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=string} "退出登录业务处理失败"
// @Router /staff/logout [post]
func (h *StaffHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		// 该场景为用户输入问题，仅返回错误不打印日志（非系统错误）
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "未传入登录凭证",
		})
		return
	}

	if err := h.svc.Logout(c.Request.Context(), token); err != nil {
		// Token脱敏（仅打印前8位），避免泄露敏感信息
		tokenMasked := token
		if len(tokenMasked) > 8 {
			tokenMasked = tokenMasked[:8] + "****"
		}
		// 打印退出业务错误原因（关联脱敏Token）
		log.Printf("[退出接口] 业务处理失败 | IP: %s | Token: %s | 错误原因: %v", c.ClientIP(), tokenMasked, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "退出登录失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "退出登录成功",
	})
}

// Success 成功响应（内部工具函数，无需Swagger注释）
func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "操作成功",
		Data:    data,
	})
}

// fail 失败响应（内部工具函数，无需Swagger注释）
func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// UpdateUserHandler 更新用户信息接口
// @Summary 更新用户基本信息
// @Description 登录用户更新邮箱、手机号、姓名、性别、出生日期等信息（UUID从中间件获取，防止篡改）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param req body dto.UpdateUserReq true "用户信息更新参数"
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=gin.H} "更新成功，Data返回更新后的用户信息摘要"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "参数错误/手机号格式错误/出生日期格式错误"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到用户身份信息/UUID无效"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "更新用户信息失败"
// @Router /staff/update [put]
func (h *StaffHandler) UpdateUserHandler(c *gin.Context) {
	// ========== 步骤1：从中间件上下文获取真实 UUID（核心，防止前端篡改） ==========
	// 中间件解析的 UUID 存入上下文的 key 为 "uuid"（需和你的中间件保持一致）
	rawUUID, exists := c.Get("uuid")
	if !exists {
		fail(c, 401, "未获取到用户身份信息，请先登录")
		return
	}
	// 类型断言 + 空值校验（确保 UUID 是有效字符串）
	realUUID, ok := rawUUID.(string)
	if !ok || strings.TrimSpace(realUUID) == "" {
		fail(c, 401, "用户 UUID 无效")
		return
	}

	// ========== 步骤2：绑定前端参数到 UpdateUserReq（无文件字段，简化绑定） ==========
	var req dto.UpdateUserReq
	// 兼容 JSON/Form 格式（无需处理文件，直接绑定）
	if err := c.ShouldBind(&req); err != nil {
		fail(c, 400, "参数解析失败："+err.Error())
		return
	}

	// ========== 步骤3：Handler 层轻量参数校验（前置拦截无效请求） ==========
	// 手机号格式校验（空则跳过，Service 层可二次校验）
	if req.Phone != "" && !phoneRegex.MatchString(req.Phone) {
		fail(c, 400, "手机号格式错误（需为11位有效手机号，如13800138000）")
		return
	}
	// 出生日期格式校验（空则跳过）
	if req.BirthDate != "" && !dateRegex.MatchString(req.BirthDate) {
		fail(c, 400, "出生日期格式错误（需为YYYY-MM-DD，如2000-01-01）")
		return
	}

	// ========== 步骤4：强制覆盖 UUID（用中间件的真实 UUID，忽略前端传入的） ==========
	req.UUID = realUUID

	// ========== 步骤5：调用 Service 层执行更新逻辑 ==========
	err := h.svc.UpdateUser(c.Request.Context(), &req)
	if err != nil {
		// 区分错误类型，返回对应状态码
		switch {
		// 业务校验错误（如手机号重复、出生日期非法）
		case strings.Contains(err.Error(), "业务校验失败"):
			fail(c, 400, err.Error())
		// 系统错误（如数据库异常）
		default:
			fail(c, 500, "更新用户信息失败："+err.Error())
		}
		return
	}

	// ========== 步骤6：返回成功响应 ==========
	success(c, gin.H{
		"uuid":     realUUID,
		"username": req.Username,
		"message":  "用户信息更新成功",
		"updateInfo": gin.H{ // 返回更新的字段，便于前端确认
			"email":     req.Email,
			"phone":     req.Phone,
			"realName":  req.RealName,
			"gender":    req.Gender,
			"birthDate": req.BirthDate,
		},
	})
}

// GetWordText 获取用户关联文字内容接口
// @Summary 获取用户关联的文字内容
// @Description 根据登录用户的UUID（中间件获取）查询关联的文字文本
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} dto.CommonResponse{Code=int,Msg=string,Data=string} "查询成功，Data返回文字内容"
// @Failure 500 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "获取UUID失败/UUID格式错误/查询文字内容失败"
// @Router /staff/word-text [get]
func (h *StaffHandler) GetWordText(c *gin.Context) {
	uuidVal, exists := c.Get("uuid")
	if !exists {
		log.Printf("[ERROR] UpdateUser 获取uuid失败 | 请求ID: %s | 原因: 上下文无uuid", c.Request.Header.Get("X-Request-ID"))
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  "服务器内部错误：获取用户标识失败",
			Data: nil,
		})
		return
	}
	// 类型断言（确保uuid是字符串类型）
	uuid, ok := uuidVal.(string)
	if !ok || uuid == "" {
		log.Printf("[ERROR] UpdateUser uuid格式错误 | 请求ID: %s | uuid值: %+v", c.Request.Header.Get("X-Request-ID"), uuidVal)
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  "服务器内部错误：用户标识格式错误",
			Data: nil,
		})
		return
	}
	text, err := h.svc.GetWordText(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  err.Error(),
			Data: nil,
		})
	}
	c.JSON(http.StatusOK, dto.CommonResponse{
		Code: 200,
		Msg:  "success",
		Data: text,
	})
}

// Checktoken 校验Token有效性接口
// @Summary 校验登录Token有效性
// @Description 验证当前Token是否有效，返回用户UUID（从Header获取）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param uuid header string true "用户UUID" example(123e4567-e89b-12d3-a456-426614174000)
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=gin.H{uuid:string}} "Token有效，返回用户UUID"
// @Router /api/v1/user/check-token [get]
func (h *StaffHandler) Checktoken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"uuid": c.Request.Header.Get("uuid"),
		},
	})
}

// UpdateAvatarHandler 更新用户头像接口
// @Summary 更新用户头像
// @Description 登录用户上传头像文件，更新头像信息（UUID从中间件获取）
// @Tags 用户管理
// @Accept multipart/form-data
// @Produce json
// @Param avatar_file formData file true "头像文件（支持jpg/png/jpeg格式）"
// @Param avatar_file_name formData string false "头像文件名（不传则使用文件原始名）" example(avatar.jpg)
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=gin.H{uuid:string,avatarURL:string,message:string}} "头像更新成功，返回头像URL"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "未上传文件/解析文件失败/打开文件失败"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到用户身份信息/UUID无效"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "更新头像失败"
// @Router /staff/update-avatar [post]
func (h *StaffHandler) UpdateAvatarHandler(c *gin.Context) {
	// ========== 步骤1：从中间件获取UUID（逻辑不变） ==========
	rawUUID, exists := c.Get("uuid")
	if !exists {
		fail(c, 401, "未获取到用户身份信息，请先登录")
		return
	}
	realUUID, ok := rawUUID.(string)
	if !ok || strings.TrimSpace(realUUID) == "" {
		fail(c, 401, "用户UUID无效")
		return
	}

	// ========== 步骤2：正确解析文件（核心！按Gin官方签名） ==========
	// 1. 调用FormFile获取FileHeader（仅2个返回值：FileHeader + error）
	fileHeader, err := c.FormFile("avatar_file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			fail(c, 400, "请上传头像文件（字段名必须为avatar_file）")
		} else {
			fail(c, 400, "解析头像文件失败："+err.Error())
		}
		return
	}

	// 2. 通过FileHeader.Open()获取文件流（multipart.File）
	file, err := fileHeader.Open()
	if err != nil {
		fail(c, 400, "打开头像文件失败："+err.Error())
		return
	}
	defer file.Close() // 必须关闭文件句柄，避免内存泄漏

	// ========== 步骤3：手动解析普通字段（文件名） ==========
	avatarFileName := c.PostForm("avatar_file_name")
	if avatarFileName == "" {
		avatarFileName = fileHeader.Filename // 用文件原始名兜底
	}

	// ========== 步骤4：构造请求参数 ==========
	req := &dto.UpdateAvatarReq{
		UUID:           realUUID,
		AvatarFileName: avatarFileName,
	}

	// ========== 步骤5：调用Service层 ==========
	avatarURL, err := h.svc.UpdateAvatar(c.Request.Context(), file, req)
	if err != nil {
		if strings.Contains(err.Error(), "业务校验失败") {
			fail(c, 400, err.Error())
		} else {
			fail(c, 500, "更新头像失败："+err.Error())
		}
		return
	}

	// ========== 步骤6：成功响应 ==========
	success(c, gin.H{
		"uuid":      realUUID,
		"avatarURL": avatarURL,
		"message":   "头像更新成功",
	})
}

// GetUserByUuid 根据UUID查询用户信息接口
// @Summary 根据UUID查询用户完整信息
// @Description 登录用户查询自身信息（UUID从中间件获取）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=model.User} "查询成功，返回用户信息"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到用户身份信息/UUID无效"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "查询用户信息失败"
// @Router /staff/get-info [get]
func (h *StaffHandler) GetUserByUuid(c *gin.Context) {
	// 1. 解析请求参数（UUID从URL路径/查询参数获取）
	uuid, exists := c.Get("uuid")
	if !exists {
		fail(c, 401, "未获取到用户身份信息，请先登录")
		return
	}
	realUUID, ok := uuid.(string)
	if !ok || strings.TrimSpace(realUUID) == "" {
		fail(c, 401, "用户 UUID 无效")
		return
	}
	// 2. 调用Service层
	user, err := h.svc.GetUserByUuid(c.Request.Context(), realUUID)
	if err != nil {
		// 3. 统一返回错误响应
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	// 4. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    user,
	})
}

// Elogin 邮箱登录-发送验证码接口
// @Summary 邮箱登录-发送验证码
// @Description 传入邮箱信息，发送验证码到指定邮箱（用于邮箱登录）
// @Tags 邮箱登录
// @Accept json
// @Produce json
// @Param req body dto.EmailLoginReq true "邮箱登录请求参数（仅传email字段）"
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=model.User} "验证码发送成功"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "参数绑定失败/发送验证码失败"
// @Router /staff/elogin [post]
func (h *StaffHandler) Elogin(c *gin.Context) {
	var req dto.EmailLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	// 构造model.User仅用于传email（适配原Service层逻辑）
	var user model.User
	user.Email = &req.Email
	user1, err := h.svc.Elogin(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"data":    user1,
		"message": "发送成功",
	})
}

// Eres 验证邮箱验证码接口
// @Summary 验证邮箱验证码
// @Description 传入邮箱和验证码，验证通过后完成邮箱登录
// @Tags 邮箱登录
// @Accept json
// @Produce json
// @Param req body dto.EmailVerifyReq true "邮箱验证码验证参数"
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.LoginResponse} "验证码验证成功，返回登录信息"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "参数绑定失败/验证码验证失败"
// @Router /staff/elogin/res [post]
func (h *StaffHandler) Eres(c *gin.Context) {
	var req dto.EmailVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	resp, err := h.svc.Verify(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": resp,
	})
}
