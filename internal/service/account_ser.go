package service

import (
	"CMS/internal/dto"
	"CMS/internal/repository"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// AccountService 账户业务接口（专属账户操作）
type AccountService interface {
	// CreateAccount 创建设户账户（支持事务，注册时调用）
	CreateAccount(ctx context.Context, tx *sql.Tx, userUUID string) error
	// Recharge 账户充值（含参数校验+业务逻辑）
	Recharge(ctx context.Context, req dto.RechargeRequest) (*dto.AccountResponse, error)
	// Deduct 扣减账户余额（含余额校验+业务逻辑）
	Deduct(ctx context.Context, req dto.DeductRequest) (*dto.AccountResponse, error)
	// GetAccountByUserUUID 查询用户账户信息
	GetAccountByUserUUID(ctx context.Context, userUUID string) (*dto.AccountResponse, error)
}

// accountServiceImpl 账户业务实现
type accountServiceImpl struct {
	accountRepo repository.AccountRepo // 依赖账户Repo
	userRepo    repository.UserRepo    // 可选：依赖用户Repo，校验用户是否存在
}

// NewAccountService 创建账户业务实例
func NewAccountService(accountRepo repository.AccountRepo, userRepo repository.UserRepo) AccountService {
	return &accountServiceImpl{
		accountRepo: accountRepo,
		userRepo:    userRepo,
	}
}

// CreateAccount 创建设户账户（透传Repo层，支持事务）
func (s *accountServiceImpl) CreateAccount(ctx context.Context, tx *sql.Tx, userUUID string) error {
	// 1. 业务参数校验（Service层兜底）
	if userUUID == "" {
		return errors.New("用户UUID不能为空")
	}

	// 2. 调用Repo层创建账户（传入事务）
	if err := s.accountRepo.CreateAccount(ctx, tx, userUUID); err != nil {
		return fmt.Errorf("创建账户失败：%w", err)
	}
	return nil
}

// Recharge 账户充值（完整业务逻辑）
func (s *accountServiceImpl) Recharge(ctx context.Context, req dto.RechargeRequest) (*dto.AccountResponse, error) {
	// 1. 严格参数校验（Service层核心职责）
	if req.UserUUID == "" {
		return nil, errors.New("用户UUID不能为空")
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("充值金额必须大于0")
	}

	// 2. 校验用户是否存在（可选，增强业务安全性）
	_, err := s.userRepo.GetUserByUuid(ctx, req.UserUUID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在：%w", err)
	}

	// 3. 调用Repo层执行充值
	if err := s.accountRepo.RechargeBalance(ctx, req.UserUUID, req.Amount); err != nil {
		return nil, fmt.Errorf("充值操作失败：%w", err)
	}

	// 4. 查询充值后的账户信息，返回给前端
	account, err := s.accountRepo.GetAccountByUserUUID(ctx, req.UserUUID)
	if err != nil {
		return nil, fmt.Errorf("查询充值后账户信息失败：%w", err)
	}
	if account == nil {
		return nil, errors.New("账户信息不存在")
	}

	// 5. 转换为响应DTO
	return &dto.AccountResponse{
		UserUUID:      account.UserUUID,
		Balance:       account.Balance,
		TotalRecharge: account.TotalRecharge,
		TotalConsume:  account.TotalConsume,
	}, nil
}

// Deduct 扣减账户余额（完整业务逻辑）
func (s *accountServiceImpl) Deduct(ctx context.Context, req dto.DeductRequest) (*dto.AccountResponse, error) {
	// 1. 严格参数校验
	if req.UserUUID == "" {
		return nil, errors.New("用户UUID不能为空")
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("扣减金额必须大于0")
	}

	// 2. 校验用户是否存在
	_, err := s.userRepo.GetUserByUuid(ctx, req.UserUUID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在：%w", err)
	}

	// 3. 调用Repo层执行扣减
	if err := s.accountRepo.DeductBalance(ctx, req.UserUUID, req.Amount); err != nil {
		return nil, fmt.Errorf("扣减余额失败：%w", err)
	}

	// 4. 查询扣减后的账户信息
	account, err := s.accountRepo.GetAccountByUserUUID(ctx, req.UserUUID)
	if err != nil {
		return nil, fmt.Errorf("查询扣减后账户信息失败：%w", err)
	}
	if account == nil {
		return nil, errors.New("账户信息不存在")
	}

	// 5. 转换为响应DTO
	return &dto.AccountResponse{
		UserUUID:      account.UserUUID,
		Balance:       account.Balance,
		TotalRecharge: account.TotalRecharge,
		TotalConsume:  account.TotalConsume,
	}, nil
}

// GetAccountByUserUUID 查询用户账户信息
func (s *accountServiceImpl) GetAccountByUserUUID(ctx context.Context, userUUID string) (*dto.AccountResponse, error) {
	// 1. 参数校验
	if userUUID == "" {
		return nil, errors.New("用户UUID不能为空")
	}

	// 2. 调用Repo层查询
	account, err := s.accountRepo.GetAccountByUserUUID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("查询账户信息失败：%w", err)
	}
	if account == nil {
		return nil, errors.New("未查询到用户账户信息")
	}

	// 3. 转换为响应DTO
	return &dto.AccountResponse{
		UserUUID:      account.UserUUID,
		Balance:       account.Balance,
		TotalRecharge: account.TotalRecharge,
		TotalConsume:  account.TotalConsume,
	}, nil
}
