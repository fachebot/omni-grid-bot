// Package variational 提供Variational交易所的工具函数
package variational

import "github.com/fachebot/omni-grid-bot/internal/ent/order"

// ConvertOrderStatus 转换订单状态
// 将Variational订单状态转换为内部订单状态
func ConvertOrderStatus(ord *Order) order.Status {
	switch ord.Status {
	case OrderStatusPending:
		return order.StatusOpen
	case OrderStatusCanceled, OrderStatusRejected:
		return order.StatusCanceled
	case OrderStatusCleared:
		return order.StatusFilled
	default:
		return order.StatusPending
	}
}
