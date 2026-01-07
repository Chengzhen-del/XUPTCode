package handler

import (
	"CMS/internal/dto"
	"CMS/internal/service"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateResourceHandler 创建资源接口
// @Summary 创建资源
// @Description 登录用户创建文本/代码类资源（需先通过Token获取UUID，关联用户ID）
// @Tags 资源管理
// @Accept json
// @Produce json
// @Param req body dto.CreateResourceReq true "创建资源请求参数" example({"title":"Go入门教程","text_content":"基础语法讲解","code_content":"package main\nimport fmt\nfunc main() {fmt.Println(\"hello\")}"})
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.ResourceItem} "创建成功"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到UUID/UUID无效"
// @Failure 404 {object} dto.Response{Code=int,Message=string,Data=nil} "用户不存在"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "参数校验失败"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "查询用户失败/创建资源失败"
// @Router /resource/create [post]
func (h *StaffHandler) CreateResourceHandler(c *gin.Context) {
	// ========== 步骤1：从Context中获取用户UUID（认证中间件解析Token后注入） ==========
	rawUserUUID, exists := c.Get("uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未获取到用户UUID，请先登录",
			"data":    nil,
		})
		return
	}

	// 转换UUID为字符串（Token解析的uuid是string类型）
	userUUID, ok := rawUserUUID.(string)
	if !ok || strings.TrimSpace(userUUID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户UUID无效",
			"data":    nil,
		})
		return
	}

	// ========== 步骤2：通过UUID查询用户，获取用户主键ID ==========
	ctx := c.Request.Context() // 复用HTTP请求的Context（支持超时/取消）
	user, err := h.svc.GetUserByUuid(ctx, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("通过UUID查询用户失败：%v", err),
			"data":    nil,
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在（UUID错误）",
			"data":    nil,
		})
		return
	}
	// 拿到用户主键ID（uint64类型，匹配数据库）
	userID := user.ID

	// ========== 步骤3：绑定并校验前端请求参数 ==========
	var req dto.CreateResourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("参数校验失败：%v", err),
			"data":    nil,
		})
		return
	}

	// ========== 步骤4：调用Service层创建资源 ==========
	err = h.resourcesvc.(*service.ResourceServiceImpl).CreateResource(
		ctx,
		userID, // 传入查询到的用户主键ID
		req.Title,
		req.TextContent,
		req.CodeContent,
	)
	if err != nil {
		// 区分不同错误类型，返回对应状态码
		switch {
		case strings.Contains(err.Error(), "查询用户失败"):
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": err.Error(),
				"data":    nil,
			})
		case strings.Contains(err.Error(), "创建资源失败"):
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": err.Error(),
				"data":    nil,
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": fmt.Sprintf("创建资源异常：%v", err),
				"data":    nil,
			})
		}
		return
	}

	// ========== 步骤5：构造成功响应（新增：点赞/浏览/评论量初始值0） ==========
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "资源创建成功",
		"data": gin.H{
			"user_id":       userID,
			"user_uuid":     userUUID, // 可选：返回uuid
			"title":         req.Title,
			"publish_time":  time.Now().Format("2006-01-02 15:04:05"),
			"like_count":    0, // 新增：点赞量初始值
			"view_count":    0, // 新增：浏览量初始值
			"comment_count": 0, // 新增：评论量初始值
		},
	})
}

// ResourceListHandler 查询资源列表接口
// @Summary 查询资源列表
// @Description 登录用户分页查询资源列表，支持关键词模糊搜索（需先登录验证UUID）
// @Tags 资源管理
// @Accept json
// @Produce json
// @Param req body dto.ResourceListReq true "资源列表查询参数" example({"page":1,"size":10,"keyword":"Go教程"})
// @Success 200 {object} dto.CommonResponse{Code=int,Msg=string,Data=dto.ResourceListResp} "查询成功，返回资源列表及分页信息"
// @Failure 401 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "未检测到登录状态"
// @Failure 400 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "参数校验失败"
// @Failure 500 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "查询资源列表失败"
// @Router /resource/list [post]
func (h *StaffHandler) ResourceListHandler(c *gin.Context) {
	// ========== 步骤1：认证校验 ==========
	_, exists := c.Get("uuid") // 确保与认证中间件的Key一致（如user_uuid）
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.CommonResponse{
			Code: 401,
			Msg:  "未检测到登录状态，请先登录",
			Data: nil,
		})
		return
	}

	// ========== 步骤2：绑定并校验POST JSON参数（使用ResourceListReq） ==========
	var req dto.ResourceListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 参数校验失败（如page<1、size>50），返回标准化400响应
		c.JSON(http.StatusBadRequest, dto.CommonResponse{
			Code: 400,
			Msg:  fmt.Sprintf("参数校验失败：%v", err),
			Data: nil,
		})
		return
	}

	// ========== 步骤3：调用Service层查询资源列表 ==========
	ctx := c.Request.Context()
	// Service层返回 []dto.ResourceItem + 总条数（已包含新字段）
	resourceItems, total, err := h.resourcesvc.GetResourceList(ctx, req.Page, req.Size, req.Keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  fmt.Sprintf("查询资源列表失败：%v", err),
			Data: nil,
		})
		return
	}

	// ========== 步骤4：封装响应数据（使用ResourceListResp） ==========
	respData := dto.ResourceListResp{
		List:  resourceItems, // 资源列表（自动包含点赞/浏览/评论量）
		Total: total,         // 总条数
		Page:  req.Page,      // 当前页（与请求一致）
		Size:  req.Size,      // 每页条数（与请求一致）
	}

	// ========== 步骤5：返回标准化成功响应 ==========
	c.JSON(http.StatusOK, dto.CommonResponse{
		Code: 200,
		Msg:  "查询成功",
		Data: respData, // Data字段为ResourceListResp结构体
	})
}

// ResourceDetailHandler 查询资源详情接口
// @Summary 查询资源详情
// @Description 根据资源ID（Query参数）查询资源完整信息，包含点赞/浏览/评论量
// @Tags 资源管理
// @Accept json
// @Produce json
// @Param id query string true "资源ID" example(1)
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=model.Resource} "查询成功，返回资源详情"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "资源ID为空/格式错误"
// @Failure 404 {object} dto.Response{Code=int,Message=string,Data=nil} "资源不存在"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "查询资源失败"
// @Router /resource/detail [get]
func (h *StaffHandler) ResourceDetailHandler(c *gin.Context) {
	// 1. 获取URL中的id参数（query参数）
	idStr := c.Query("id")
	if idStr == "" {
		// 参数缺失，返回400错误
		c.JSON(http.StatusOK, dto.Response{
			Code:    400,
			Message: "参数错误：资源ID不能为空",
			Data:    nil,
		})
		return
	}

	// 2. 转换ID为整数并校验（适配uint64类型）
	idUint64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || idUint64 <= 0 {
		c.JSON(http.StatusOK, dto.Response{
			Code:    400,
			Message: "参数错误：资源ID必须为正整数",
			Data:    nil,
		})
		return
	}

	// 3. 调用服务层查询资源详情（真实DB查询）
	resource, err := h.resourcesvc.GetResourceByID(c.Request.Context(), idUint64)
	if err != nil {
		// 数据库查询出错（如连接失败、SQL错误等）
		c.JSON(http.StatusOK, dto.Response{
			Code:    500,
			Message: "服务器内部错误：查询资源失败 - " + err.Error(),
			Data:    nil,
		})
		return
	}

	// 4. 检查资源是否存在
	if resource == nil {
		c.JSON(http.StatusOK, dto.Response{
			Code:    404,
			Message: "资源不存在：未找到ID为" + idStr + "的资源",
			Data:    nil,
		})
		return
	}

	// 5. 格式化响应数据（新增：点赞/浏览/评论量字段）
	responseData := struct {
		ID           uint64 `json:"id"`
		Title        string `json:"title"`
		Author       string `json:"author"`
		PublishTime  string `json:"publish_time"` // 转为字符串格式
		TextContent  string `json:"text_content"`
		CodeContent  string `json:"code_content"`
		LikeCount    uint64 `json:"like_count"`    // 新增：点赞量
		ViewCount    uint64 `json:"view_count"`    // 新增：浏览量
		CommentCount uint64 `json:"comment_count"` // 新增：评论量
		// 如需返回user_id可添加，前端没要求则可省略
	}{
		ID:           resource.ID,
		Title:        resource.Title,
		Author:       resource.Author,
		PublishTime:  resource.PublishTime.Format("2006-01-02 15:04:05"), // 关键：时间格式化
		TextContent:  resource.TextContent,
		CodeContent:  resource.CodeContent,
		LikeCount:    resource.LikeCount,    // 新增：赋值点赞量
		ViewCount:    resource.ViewCount,    // 新增：赋值浏览量
		CommentCount: resource.CommentCount, // 新增：赋值评论量
	}

	// 6. 返回成功响应（完全匹配前端要求的格式）
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success", // 对应前端的msg字段
		Data:    responseData,
	})
}

// IncrViewCountHandler 增加资源浏览量接口
// @Summary 增加资源浏览量
// @Description 传入资源ID，将该资源的浏览量+1
// @Tags 资源管理
// @Accept json
// @Produce json
// @Param req body dto.ResourceIDReq true "资源ID参数" example({"id":1})
// @Success 200 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "浏览量更新成功"
// @Failure 400 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "参数校验失败（ID为空/小于1）"
// @Failure 500 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "增加浏览量失败"
// @Router /resource/incr-view [post]
func (h *StaffHandler) IncrViewCountHandler(c *gin.Context) {
	// 1. 绑定请求参数
	var req dto.ResourceIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.CommonResponse{
			Code: 400,
			Msg:  fmt.Sprintf("参数校验失败：%v", err),
			Data: nil,
		})
		return
	}

	// 2. 调用Service层增加浏览量
	ctx := c.Request.Context()
	err := h.resourcesvc.IncrViewCount(ctx, req.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  fmt.Sprintf("增加浏览量失败：%v", err),
			Data: nil,
		})
		return

	}

	// 3. 返回成功响应
	c.JSON(http.StatusOK, dto.CommonResponse{
		Code: 200,
		Msg:  "浏览量更新成功",
		Data: nil,
	})
}

// IncrLikeCountHandler 点赞接口：增加资源点赞数
// @Summary 资源点赞
// @Description 传入资源ID，将该资源的点赞量+1
// @Tags 资源管理
// @Accept json
// @Produce json
// @Param req body dto.ResourceIDReq true "资源ID参数" example({"id":1})
// @Success 200 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "点赞成功"
// @Failure 400 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "参数校验失败（ID为空/小于1）"
// @Failure 500 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "点赞失败"
// @Router /resource/like [post]
func (h *StaffHandler) IncrLikeCountHandler(c *gin.Context) {
	// 1. 绑定并校验请求参数（和浏览量接口格式一致）
	var req dto.ResourceIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.CommonResponse{
			Code: 400,
			Msg:  fmt.Sprintf("参数校验失败：%v", err),
			Data: nil,
		})
		println(req.ID)
		return
	}

	// 2. 调用Service层增加点赞量（需确保Service已实现IncrLikeCount方法）
	ctx := c.Request.Context()
	err := h.resourcesvc.IncrLikeCount(ctx, req.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  "点赞失败", // 严格匹配你要求的返回文案
			Data: nil,
		})
		return
	}

	// 3. 返回成功响应（匹配你要求的格式）
	c.JSON(http.StatusOK, dto.CommonResponse{
		Code: 200,
		Msg:  "点赞成功", // 严格匹配你要求的返回文案
		Data: nil,
	})
}

// CreateCommentHandler 评论接口：提交资源评论并增加评论数
// @Summary 提交资源评论
// @Description 传入资源ID和评论内容，创建评论并将该资源的评论量+1
// @Tags 资源管理
// @Accept json
// @Produce json
// @Param req body dto.CommentReq true "评论请求参数" example({"id":1,"content":"这篇教程很实用！"})
// @Success 200 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "评论成功"
// @Failure 400 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "参数校验失败（ID为空/小于1或评论内容为空）"
// @Failure 500 {object} dto.CommonResponse{Code=int,Msg=string,Data=nil} "评论失败"
// @Router /resource/comment [post]
func (h *StaffHandler) CreateCommentHandler(c *gin.Context) {
	// 1. 绑定并校验请求参数（新增content字段，必填且非空）
	var req dto.CommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.CommonResponse{
			Code: 400,
			Msg:  fmt.Sprintf("参数校验失败：%v", err),
			Data: nil,
		})
		return
	}

	// 2. 调用Service层创建评论（需确保Service已实现CreateComment方法）
	ctx := c.Request.Context()
	err := h.resourcesvc.CreateComment(ctx, req.ID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.CommonResponse{
			Code: 500,
			Msg:  "评论失败", // 严格匹配你要求的返回文案
			Data: nil,
		})
		return
	}

	// 3. 返回成功响应（匹配你要求的格式）
	c.JSON(http.StatusOK, dto.CommonResponse{
		Code: 200,
		Msg:  "评论成功", // 严格匹配你要求的返回文案
		Data: nil,
	})
}
