package handler

import (
	"CMS/internal/dto"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// 补充缺失的结构体定义（保证代码可编译）

// Success 成功响应（内部工具函数，无需Swagger注释）
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// Fail 失败响应（内部工具函数，无需Swagger注释）
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// Error 系统错误响应（HTTP 500）（内部工具函数，无需Swagger注释）
func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, dto.Response{
		Code:    500,
		Message: msg,
		Data:    nil,
	})
}

// Recharge 账户充值接口
// @Summary 账户充值
// @Description 登录用户为自身账户充值（UUID从中间件获取，自动关联用户，仅需传入充值金额）
// @Tags 账户管理
// @Accept json
// @Produce json
// @Param req body dto.RechargeRequest true "充值请求参数" example({"amount":100.00})
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.AccountResponse} "充值成功，Data返回账户充值后信息"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到用户UUID/UUID无效"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "参数校验失败/充值金额格式错误/充值金额≤0/用户不存在"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "充值失败（数据库异常等系统错误）"
// @Router /account/recharge [post]
func (h *StaffHandler) Recharge(c *gin.Context) {
	// ========== 从上下文获取登录用户的UUID ==========
	rawUUID, exists := c.Get("uuid")
	if !exists {
		Fail(c, 401, "未获取到用户身份信息，请先登录")
		return
	}
	// 类型断言 + 空值校验
	realUUID, ok := rawUUID.(string)
	if !ok || strings.TrimSpace(realUUID) == "" {
		Fail(c, 401, "用户 UUID 无效")
		return
	}

	// 1. 绑定并校验请求参数（仅绑定amount，无需user_uuid）
	var req dto.RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, fmt.Sprintf("参数校验失败：%v", err))
		return
	}

	// 2. 校验decimal金额（格式+数值合法性）
	if req.Amount.IsZero() && req.Amount.String() != "0" {
		Fail(c, 400, "充值金额格式错误（请传入合法数字，如100.00）")
		return
	}
	// 新增：校验充值金额必须大于0
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		Fail(c, 400, "充值金额必须大于0")
		return
	}

	// ========== 将上下文的UUID赋值给请求体 ==========
	req.UserUUID = realUUID

	// 3. 调用Service层处理业务逻辑
	resp, err := h.accsvc.Recharge(c.Request.Context(), req)
	if err != nil {
		// 区分业务错误和系统错误
		switch {
		// 业务错误（参数/逻辑问题，返回400）
		case errors.Is(err, errors.New("充值金额必须大于0")):
			fallthrough
		case strings.Contains(err.Error(), "用户不存在"):
			Fail(c, 400, err.Error())
		// 系统错误（如数据库异常，返回500）
		default:
			Error(c, fmt.Sprintf("充值失败：%v", err))
		}
		return
	}

	// 4. 返回成功响应
	Success(c, resp)
}

// Deduct 账户余额扣减接口
// @Summary 扣减账户余额
// @Description 登录用户扣减自身账户余额（UUID从中间件获取，自动关联用户，仅需传入扣减金额）
// @Tags 账户管理
// @Accept json
// @Produce json
// @Param req body dto.DeductRequest true "扣减请求参数" example({"amount":50.00})
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.AccountResponse} "扣减成功，Data返回账户扣减后信息"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到用户UUID/UUID无效"
// @Failure 400 {object} dto.Response{Code=int,Message=string,Data=nil} "参数校验失败/扣减金额格式错误/扣减金额≤0/用户不存在/账户余额不足"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "扣减余额失败（数据库异常等系统错误）"
// @Router /account/deduct [post]
func (h *StaffHandler) Deduct(c *gin.Context) {
	// ========== 从上下文获取登录用户的UUID ==========
	rawUUID, exists := c.Get("uuid")
	if !exists {
		Fail(c, 401, "未获取到用户身份信息，请先登录")
		return
	}
	// 类型断言 + 空值校验
	realUUID, ok := rawUUID.(string)
	if !ok || strings.TrimSpace(realUUID) == "" {
		Fail(c, 401, "用户 UUID 无效")
		return
	}

	// 1. 绑定并校验请求参数（仅绑定amount，无需user_uuid）
	var req dto.DeductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, fmt.Sprintf("参数校验失败：%v", err))
		return
	}

	// 2. 校验decimal金额（格式+数值合法性）
	if req.Amount.IsZero() && req.Amount.String() != "0" {
		Fail(c, 400, "扣减金额格式错误（请传入合法数字，如100.00）")
		return
	}
	// 新增：校验扣减金额必须大于0
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		Fail(c, 400, "扣减金额必须大于0")
		return
	}

	// ========== 将上下文的UUID赋值给请求体 ==========
	req.UserUUID = realUUID

	// 3. 调用Service层处理业务逻辑
	resp, err := h.accsvc.Deduct(c.Request.Context(), req)
	if err != nil {
		// 区分业务错误和系统错误
		switch {
		// 业务错误（400）
		case errors.Is(err, errors.New("扣减金额必须大于0")):
			fallthrough
		case strings.Contains(err.Error(), "用户不存在"):
			fallthrough
		case strings.Contains(err.Error(), "账户余额不足"):
			Fail(c, 400, err.Error())
		// 系统错误（500）
		default:
			Error(c, fmt.Sprintf("扣减余额失败：%v", err))
		}
		return
	}

	// 4. 返回成功响应
	Success(c, resp)
}

// GetAccountByUserUUID 查询账户信息接口
// @Summary 查询账户信息
// @Description 登录用户查询自身账户信息（UUID从中间件获取，无需传入参数）
// @Tags 账户管理
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{Code=int,Message=string,Data=dto.AccountResponse} "查询成功，Data返回账户余额、UUID等信息"
// @Failure 401 {object} dto.Response{Code=int,Message=string,Data=nil} "未获取到用户UUID/UUID无效"
// @Failure 404 {object} dto.Response{Code=int,Message=string,Data=nil} "未查询到用户账户信息"
// @Failure 500 {object} dto.Response{Code=int,Message=string,Data=nil} "查询账户信息失败（数据库异常等系统错误）"
// @Router /account/get-account [get]
func (h *StaffHandler) GetAccountByUserUUID(c *gin.Context) {
	// ========== 从上下文获取登录用户的UUID ==========
	rawUUID, exists := c.Get("uuid")
	if !exists {
		Fail(c, 401, "未获取到用户身份信息，请先登录")
		return
	}
	// 类型断言 + 空值校验
	realUUID, ok := rawUUID.(string)
	if !ok || strings.TrimSpace(realUUID) == "" {
		Fail(c, 401, "用户 UUID 无效")
		return
	}

	// 2. 调用Service层查询账户信息（直接用上下文的UUID）
	resp, err := h.accsvc.GetAccountByUserUUID(c.Request.Context(), realUUID)
	if err != nil {
		// 区分业务错误和系统错误
		switch {
		// 业务错误（404）
		case errors.Is(err, errors.New("未查询到用户账户信息")):
			Fail(c, 404, err.Error())
		// 系统错误（500）
		default:
			Error(c, fmt.Sprintf("查询账户信息失败：%v", err))
		}
		return
	}

	// 3. 返回成功响应
	Success(c, resp)
}
