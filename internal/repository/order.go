// repo/account_repo.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"CMS/internal/model"

	"github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
)

// AccountRepo 账户Repo接口（定义账户操作）
type AccountRepo interface {
	// CreateAccount 创建设户账户（支持传入事务，保证原子性）
	CreateAccount(ctx context.Context, tx *sql.Tx, userUUID string) error
	// RechargeBalance 账户充值
	RechargeBalance(ctx context.Context, userUUID string, amount decimal.Decimal) error
	// DeductBalance 账户扣减
	DeductBalance(ctx context.Context, userUUID string, amount decimal.Decimal) error
	// GetAccountByUserUUID 根据用户UUID查询账户
	GetAccountByUserUUID(ctx context.Context, userUUID string) (*model.UserAccount, error)
}

// accountRepoImpl AccountRepo实现
type accountRepoImpl struct {
	db *sql.DB // 数据库连接
}

// NewAccountRepo 创建AccountRepo实例
func NewAccountRepo(db *sql.DB) AccountRepo {
	return &accountRepoImpl{db: db}
}

// CreateAccount 创建设户账户（核心：支持传入外部事务）
func (r *accountRepoImpl) CreateAccount(ctx context.Context, tx *sql.Tx, userUUID string) error {
	// 适配外部事务：有tx则用tx执行，无则用db执行（兼容单独创建账户场景）
	execFunc := r.db.ExecContext
	if tx != nil {
		execFunc = tx.ExecContext
	}

	// 插入账户（匹配你的user_account表结构）
	sqlStr := `INSERT INTO user_account (user_uuid) VALUES (?)`
	_, err := execFunc(ctx, sqlStr, userUUID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062: // uk_user_uuid唯一索引冲突
				return fmt.Errorf("账户表user_uuid重复：%s", mysqlErr.Message)
			case 1048: // user_uuid非空约束
				return fmt.Errorf("账户表user_uuid不能为空：%s", mysqlErr.Message)
			}
		}
		return fmt.Errorf("插入账户失败：%w", err)
	}
	return nil
}

// RechargeBalance 充值（复用原有逻辑）
func (r *accountRepoImpl) RechargeBalance(ctx context.Context, userUUID string, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("充值金额必须大于0")
	}

	sqlStr := `
	UPDATE user_account 
	SET balance = balance + ?, total_recharge = total_recharge + ?
	WHERE user_uuid = ?
	`
	result, err := r.db.ExecContext(ctx, sqlStr,
		amount.String(),
		amount.String(),
		userUUID,
	)
	if err != nil {
		return fmt.Errorf("充值失败：%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取充值影响行数失败：%w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("用户账户不存在或更新失败")
	}
	return nil
}

// DeductBalance 扣减余额（复用原有逻辑）
func (r *accountRepoImpl) DeductBalance(ctx context.Context, userUUID string, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("消费金额必须大于0")
	}

	// 查询余额
	var balanceStr string
	querySql := `SELECT balance FROM user_account WHERE user_uuid = ?`
	err := r.db.QueryRowContext(ctx, querySql, userUUID).Scan(&balanceStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("用户账户不存在")
		}
		return fmt.Errorf("查询余额失败：%w", err)
	}

	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return fmt.Errorf("解析余额失败：%w", err)
	}
	if balance.LessThan(amount) {
		return fmt.Errorf("账户余额不足（当前：%s，需扣减：%s）", balance.String(), amount.String())
	}

	// 扣减
	updateSql := `
	UPDATE user_account 
	SET balance = balance - ?, total_consume = total_consume + ?
	WHERE user_uuid = ?
	`
	result, err := r.db.ExecContext(ctx, updateSql,
		amount.String(),
		amount.String(),
		userUUID,
	)
	if err != nil {
		return fmt.Errorf("扣减余额失败：%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取扣减影响行数失败：%w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("余额扣减失败（账户状态异常）")
	}
	return nil
}

// GetAccountByUserUUID 查询账户
func (r *accountRepoImpl) GetAccountByUserUUID(ctx context.Context, userUUID string) (*model.UserAccount, error) {
	var account model.UserAccount
	sqlStr := `
	SELECT id, user_uuid, balance, total_recharge, total_consume 
	FROM user_account 
	WHERE user_uuid = ?
	`
	err := r.db.QueryRowContext(ctx, sqlStr, userUUID).Scan(
		&account.ID,
		&account.UserUUID,
		&account.Balance,
		&account.TotalRecharge,
		&account.TotalConsume,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 无账户返回nil，不抛错
		}
		return nil, fmt.Errorf("查询账户失败：%w", err)
	}
	return &account, nil
}
