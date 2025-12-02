package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/variational"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type VariationalOrderHelper struct {
	svcCtx     *svc.ServiceContext
	userClient *variational.UserClient
}

func GetVariationalClient(svcCtx *svc.ServiceContext, record *ent.Strategy) (*variational.UserClient, error) {
	dexAccount := record.ExchangeApiKey
	dexPrivateKey := record.ExchangeSecretKey
	return variational.NewUserClient(svcCtx.VariationalClient, dexAccount, dexPrivateKey), nil
}

func NewVariationalOrderHelper(svcCtx *svc.ServiceContext, userClient *variational.UserClient) *VariationalOrderHelper {
	return &VariationalOrderHelper{svcCtx: svcCtx, userClient: userClient}
}

func (h *VariationalOrderHelper) UpdateLeverage(ctx context.Context, symbol string, leverage uint, marginMode exchange.MarginMode) error {
	// variational 暂不支持逐仓保证金
	return h.userClient.SetLeverage(ctx, symbol, int(leverage))
}

func (h *VariationalOrderHelper) CancalAllOrders(ctx context.Context, symbol string) error {
	const limit = 100

	// 查询订单列表
	offset := 0
	pendingOrders := make([]string, 0)
	for {
		res, err := h.userClient.GetUserPendingOrders(ctx, offset, limit)
		if err != nil {
			return err
		}

		for _, item := range res.Result {
			pendingOrders = append(pendingOrders, item.RfqID)
		}

		if res.Pagination.NextPage == nil {
			break
		}
		n, err := strconv.ParseInt(res.Pagination.NextPage.Offset, 10, 64)
		if err != nil {
			break
		}
		offset = int(n)
	}

	// 关闭所有订单
	errorList := make([]error, 0)
	for _, rfqId := range pendingOrders {
		err := h.userClient.CancelOrder(ctx, rfqId)
		if err != nil {
			errorList = append(errorList, err)
			logger.Warnf("[VariationalOrderHelper] 取消订单失败, account: %s, rfqId: %s, %v", h.userClient.EthAccount(), rfqId, err)
		}
		time.Sleep(1 * time.Second)
	}

	if len(errorList) > 0 {
		return errorList[0]
	}
	return nil
}

func (h *VariationalOrderHelper) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]string, []string, error) {
	limitOrderClientIds := make([]string, 0)
	for _, ord := range limitOrders {
		rfqId, err := h.CreateLimitOrder(ctx, ord.Symbol, ord.IsAsk, ord.ReduceOnly, ord.Price, ord.Size)
		if err != nil {
			return nil, nil, err
		}
		limitOrderClientIds = append(limitOrderClientIds, rfqId)

		time.Sleep(1 * time.Second)
	}

	marketOrderClientIds := make([]string, 0)
	for _, ord := range marketOrders {
		rfqId, err := h.CreateMarketOrder(ctx, ord.Symbol, ord.IsAsk, ord.ReduceOnly, ord.SlippageBps, ord.Size)
		if err != nil {
			return nil, nil, err
		}
		marketOrderClientIds = append(marketOrderClientIds, rfqId)

		time.Sleep(1 * time.Second)
	}

	return limitOrderClientIds, marketOrderClientIds, nil
}

func (h *VariationalOrderHelper) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (string, error) {
	side := lo.If(isAsk, variational.OrderSideSell).Else(variational.OrderSideBuy)
	res, err := h.userClient.CreateLimitOrder(ctx, symbol, side, price, size, reduceOnly)
	if err != nil {
		return "", err
	}

	time.Sleep(1 * time.Second)

	return res.RfqId, nil
}

func (h *VariationalOrderHelper) CreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, slippageBps int, size decimal.Decimal) (string, error) {
	side := lo.If(isAsk, variational.OrderSideSell).Else(variational.OrderSideBuy)
	maxSlippage := decimal.NewFromInt(int64(slippageBps)).Div(decimal.NewFromInt(10000)).Truncate(4)
	res, err := h.userClient.CreateMarketOrder(ctx, symbol, side, size, maxSlippage, reduceOnly)
	if err != nil {
		return "", err
	}

	time.Sleep(1 * time.Second)

	return res.RfqId, nil
}

func (h *VariationalOrderHelper) SyncUserOrders(ctx context.Context) error {
	account := h.userClient.EthAccount()
	logger.Debugf("[VariationalOrderHelper] 同步用户订单开始, account: %s", account)

	// 计算开始时间
	var startTime *time.Time
	grids, err := h.svcCtx.GridModel.FindAllByAccount(ctx, account)
	if err != nil {
		logger.Debugf("[VariationalOrderHelper] 查询账户网格失败, account: %s,  %v", account, err)
		return err
	}
	nums := make([]int64, 0)
	for _, item := range grids {
		if item.BuyClientOrderTime != nil {
			nums = append(nums, *item.BuyClientOrderTime)
		}
		if item.SellClientOrderTime != nil {
			nums = append(nums, *item.SellClientOrderTime)
		}
	}
	if len(nums) > 0 {
		t := time.UnixMilli(lo.Min(nums) - 5*1000)
		startTime = &t
	}

	// 同步所有订单
	offset := 0
	const limit = 100
	userOrders := make([]*variational.Order, 0)
exit:
	for {
		logger.Debugf("[VariationalOrderHelper] 查询用户订单记录开始, account: %s, offset: %d, limit: %d", account, offset, limit)
		userOrdersRes, err := h.userClient.GetUserOrders(ctx, offset, limit)
		if err != nil {
			logger.Debugf("[VariationalOrderHelper] 查询用户订单记录失败, account: %s, offset: %d, limit: %d, %v", account, offset, limit, err)
			return err
		}
		logger.Debugf("[VariationalOrderHelper] 查询用户订单记录结束, account: %s, offset: %d, limit: %d", account, offset, limit)

		for _, item := range userOrdersRes.Result {
			if startTime != nil && item.CreatedAt.Compare(*startTime) == -1 {
				break exit
			}
			userOrders = append(userOrders, item)
		}

		if userOrdersRes.Pagination.NextPage == nil {
			break
		}
		n, err := strconv.ParseInt(userOrdersRes.Pagination.NextPage.Offset, 10, 64)
		if err != nil {
			break
		}
		offset = int(n)
	}
	logger.Debugf("[VariationalOrderHelper] 同步用户订单结束, account: %s, count: %d", account, len(userOrders))

	// 本地化存储用户订单
	return util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, item := range userOrders {
			price := decimal.Zero
			if item.LimitPrice != nil {
				price = *item.LimitPrice
			}
			if item.Price != nil {
				price = *item.Price
			}

			timestamp := item.CreatedAt
			if item.ExecutionTimestamp != nil {
				timestamp = *item.ExecutionTimestamp
			}

			filledBaseAmount := decimal.Zero
			filledQuoteAmount := decimal.Zero
			if item.Price != nil {
				filledBaseAmount = item.Qty
				filledQuoteAmount = item.Qty.Mul(*item.Price)
			}

			args := ent.Order{
				Exchange:          exchange.Variational,
				Account:           account,
				Symbol:            item.Instrument.Underlying,
				OrderId:           item.RfqID,
				ClientOrderId:     item.RfqID,
				Side:              lo.If(item.Side == variational.OrderSideSell, order.SideSell).Else(order.SideBuy),
				Price:             price,
				BaseAmount:        item.Qty,
				FilledBaseAmount:  filledBaseAmount,
				FilledQuoteAmount: filledQuoteAmount,
				Status:            variational.ConvertOrderStatus(item),
				Timestamp:         timestamp.UnixMilli(),
			}
			err = h.svcCtx.OrderModel.Upsert(ctx, args)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *VariationalOrderHelper) ClosePosition(ctx context.Context, symbol string, side Side, slippageBps int) error {
	positions, err := h.userClient.GetPositions(ctx)
	if err != nil {
		return err
	}

	position, ok := lo.Find(positions, func(item *variational.Position) bool {
		if item.PositionInfo.Instrument.Underlying != symbol {
			return false
		}
		if side == LONG && item.PositionInfo.Qty.GreaterThan(decimal.Zero) {
			return true
		}
		if side == SHORT && item.PositionInfo.Qty.LessThan(decimal.Zero) {
			return true
		}
		return false
	})
	if !ok {
		return nil
	}

	if position.PositionInfo.Qty.GreaterThan(decimal.Zero) {
		_, err = h.CreateMarketOrder(ctx, symbol, true, true, slippageBps, position.PositionInfo.Qty.Abs())
	} else if position.PositionInfo.Qty.LessThan(decimal.Zero) {
		_, err = h.CreateMarketOrder(ctx, symbol, false, true, slippageBps, position.PositionInfo.Qty.Abs())
	}
	if err != nil {
		logger.Errorf("[VariationalOrderHelper] 关闭仓位失败, account: %s, symbol: %s, size: %s, %v",
			h.userClient.EthAccount(), symbol, position.PositionInfo.Qty.Abs(), err)
	}

	return err
}
