package service

import (
	"errors"

	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DeductUserBalance 扣减用户余额，并将“余额不足”收敛为强类型业务错误。
//
// 注意：这是“结算扣费”阶段的兜底保护（例如并发消费导致预校验通过但最终扣费失败）。
func DeductUserBalance(userRepo repository.UserRepository, userID uuid.UUID, amount decimal.Decimal, knownBalance *decimal.Decimal) error {
	if userRepo == nil {
		return errors.New("userRepo 不能为空")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	err := userRepo.DeductBalance(userID, amount)
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrInsufficientBalance) {
		balance := decimal.Zero
		if knownBalance != nil {
			balance = *knownBalance
		}
		return &InsufficientFundsError{Needed: amount, Balance: balance}
	}

	return err
}
