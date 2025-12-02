package variational

import "github.com/fachebot/omni-grid-bot/internal/ent/order"

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
