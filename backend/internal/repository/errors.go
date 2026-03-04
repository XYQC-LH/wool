package repository

import "errors"

var (
	// ErrInsufficientBalance 表示用户余额不足（repository 层哨兵错误）。
	ErrInsufficientBalance = errors.New("余额不足")
	// ErrInsufficientQuota 表示 Token 配额不足（repository 层哨兵错误）。
	ErrInsufficientQuota = errors.New("配额不足")
)
