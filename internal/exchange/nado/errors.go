// Package nado 提供Nado交易所的错误定义
package nado

import "errors"

// 错误变量定义
var (
	ErrInvalidSender = errors.New("invalid sender") // 无效的发送者地址
)
