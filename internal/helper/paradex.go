package helper

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type ParadexOrderHelper struct {
	svcCtx     *svc.ServiceContext
	userClient *paradex.UserClient
}

func GetParadexClient(svcCtx *svc.ServiceContext, record *ent.Strategy) (*paradex.UserClient, error) {
	dexAccount := record.ExchangeApiKey
	dexPrivateKey := record.ExchangeSecretKey
	return paradex.NewUserClient(svcCtx.ParadexClient, dexAccount, dexPrivateKey), nil
}

func NewParadexOrderHelper(svcCtx *svc.ServiceContext, userClient *paradex.UserClient) *ParadexOrderHelper {
	return &ParadexOrderHelper{svcCtx: svcCtx, userClient: userClient}
}

func (h *ParadexOrderHelper) UpdateLeverage(ctx context.Context, symbol string, leverage uint, marginMode exchange.MarginMode) error {
	market := paradex.FormatUsdPerpMarket(symbol)
	marginType := lo.If(marginMode == exchange.MarginModeCross, paradex.MarginTypeCross).Else(paradex.MarginTypeIsolated)
	_, err := h.userClient.UpsertAccountMargin(ctx, market, leverage, marginType)
	return err
}

func (h *ParadexOrderHelper) CancalAllOrders(ctx context.Context, symbol string) error {
	return h.userClient.CancelAllOpenOrders(ctx, paradex.FormatUsdPerpMarket(symbol))
}

func (h *ParadexOrderHelper) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]string, []string, error) {
	nextClientId := time.Now().UnixNano()
	limitOrderClientIds := make([]string, 0)
	batchOrders := make([]*paradex.CreateOrderReq, 0)

	for _, item := range limitOrders {
		clientId := strconv.FormatInt(nextClientId, 10)
		limitOrderClientIds = append(limitOrderClientIds, clientId)

		ord := &paradex.CreateOrderReq{
			Instruction: paradex.InstructionGTC,
			Market:      paradex.FormatUsdPerpMarket(item.Symbol),
			Price:       item.Price.String(),
			Side:        lo.If(item.IsAsk, paradex.OrderSideSell).Else(paradex.OrderSideBuy),
			Size:        item.Size,
			Type:        paradex.OrderTypeLimit,
			Flags:       make([]paradex.OrderFlag, 0),
			ClientID:    &clientId,
		}
		if item.ReduceOnly {
			ord.Flags = append(ord.Flags, paradex.OrderFlagReduceOnly)
		}

		nextClientId += 1
		batchOrders = append(batchOrders, ord)
	}

	marketOrderClientIds := make([]string, 0)
	for _, item := range marketOrders {
		clientId := strconv.FormatInt(nextClientId, 10)
		marketOrderClientIds = append(marketOrderClientIds, clientId)

		ord := &paradex.CreateOrderReq{
			Instruction: paradex.InstructionGTC,
			Market:      paradex.FormatUsdPerpMarket(item.Symbol),
			Side:        lo.If(item.IsAsk, paradex.OrderSideSell).Else(paradex.OrderSideBuy),
			Size:        item.Size,
			Type:        paradex.OrderTypeMarket,
			Flags:       make([]paradex.OrderFlag, 0),
			ClientID:    &clientId,
		}
		if item.ReduceOnly {
			ord.Flags = append(ord.Flags, paradex.OrderFlagReduceOnly)
		}

		nextClientId += 1
		batchOrders = append(batchOrders, ord)
	}

	res, err := h.userClient.CreateBatchOrders(ctx, batchOrders)
	if err != nil {
		return nil, nil, err
	}

	for _, item := range res.Errors {
		if item != nil {
			return nil, nil, item
		}
	}

	return limitOrderClientIds, marketOrderClientIds, err
}

func (h *ParadexOrderHelper) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (string, error) {
	p := CreateLimitOrderParams{
		Symbol:     symbol,
		IsAsk:      isAsk,
		ReduceOnly: reduceOnly,
		Price:      price,
		Size:       size,
	}
	clientIds, _, err := h.CreateOrderBatch(context.TODO(), []CreateLimitOrderParams{p}, nil)
	if err != nil {
		return "", err
	}

	return clientIds[0], nil
}

func (h *ParadexOrderHelper) CreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, acceptableExecutionPrice, size decimal.Decimal) (string, error) {
	p := CreateMarketOrderParams{
		Symbol:                   symbol,
		IsAsk:                    isAsk,
		ReduceOnly:               reduceOnly,
		AcceptableExecutionPrice: acceptableExecutionPrice,
		Size:                     size,
	}
	_, clientIds, err := h.CreateOrderBatch(context.TODO(), nil, []CreateMarketOrderParams{p})
	if err != nil {
		return "", err
	}

	return clientIds[0], nil
}

func (h *ParadexOrderHelper) SyncUserOrders(ctx context.Context) error {
	account := h.userClient.DexAccount()
	logger.Debugf("[ParadexOrderHelper] 同步用户订单开始, account: %s", account)

	syncProgress, err := h.svcCtx.SyncProgressModel.Ensure(ctx, exchange.Paradex, account)
	if err != nil {
		return err
	}

	// 计算开始时间
	var startTime *time.Time
	grids, err := h.svcCtx.GridModel.FindAllByAccount(ctx, account)
	if err != nil {
		logger.Debugf("[ParadexOrderHelper] 查询账户网格失败, account: %s,  %v", account, err)
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
	cursor := ""
	const limit = 1000
	userOrders := make([]*paradex.Order, 0)
	for {
		logger.Debugf("[ParadexOrderHelper] 查询用户订单记录开始, account: %s, cursor: %s, limit: %d", account, cursor, limit)
		userOrdersRes, err := h.userClient.GetUserOrders(ctx, startTime, cursor, limit)
		if err != nil {
			logger.Debugf("[ParadexOrderHelper] 查询用户订单记录失败, account: %s, cursor: %s, limit: %d, %v", account, cursor, limit, err)
			return err
		}
		logger.Debugf("[ParadexOrderHelper] 查询用户订单记录结束, account: %s, cursor: %s, limit: %d", account, cursor, limit)

		cursor = userOrdersRes.Next
		userOrders = append(userOrders, userOrdersRes.Results...)
		if cursor == "" {
			break
		}
	}

	// 本地排序订单
	slices.SortFunc(userOrders, func(a, b *paradex.Order) int {
		if a.LastUpdatedAt > b.LastUpdatedAt {
			return -1
		} else if a.LastUpdatedAt == b.LastUpdatedAt {
			return 0
		}
		return 1
	})
	for idx, item := range userOrders {
		if item.LastUpdatedAt <= syncProgress.Timestamp {
			userOrders = userOrders[:idx]
			break
		}
	}

	logger.Debugf("[ParadexOrderHelper] 同步用户订单结束, account: %s, count: %d", account, len(userOrders))

	// 本地化存储用户订单
	return util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, item := range userOrders {
			symbol, err := paradex.ParseUsdPerpMarket(item.Market)
			if err != nil {
				continue
			}

			filledQuoteAmount := decimal.Zero
			if item.AvgFillPrice != "" {
				avgFillPrice, err := decimal.NewFromString(item.AvgFillPrice)
				if err == nil {
					filledQuoteAmount = item.Size.Mul(avgFillPrice)
				}
			}

			args := ent.Order{
				Exchange:          exchange.Paradex,
				Account:           account,
				Symbol:            symbol,
				OrderId:           item.ID,
				ClientOrderId:     item.ClientID,
				Side:              lo.If(item.Side == paradex.OrderSideSell, order.SideSell).Else(order.SideBuy),
				Price:             item.Price,
				BaseAmount:        item.Size,
				FilledBaseAmount:  item.Size.Sub(item.RemainingSize),
				FilledQuoteAmount: filledQuoteAmount,
				Status:            paradex.ConvertOrderStatus(item),
				Timestamp:         item.LastUpdatedAt,
			}
			err = h.svcCtx.OrderModel.Upsert(ctx, args)
			if err != nil {
				return err
			}
		}

		if len(userOrders) > 0 {
			ts := userOrders[0].LastUpdatedAt
			err = h.svcCtx.SyncProgressModel.UpdateTimestampByAccount(ctx, exchange.Paradex, account, ts)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *ParadexOrderHelper) ClosePosition(ctx context.Context, symbol string, side Side, slippageBps int) error {
	// 更新最大滑点
	market := paradex.FormatUsdPerpMarket(symbol)
	_, err := h.userClient.UpdateAccountMarketMaxSlippage(ctx, market, slippageBps)
	if err != nil {
		return err
	}

	// 查找指定仓位
	positions, err := h.userClient.GetPositions(ctx)
	if err != nil {
		return err
	}

	positionSide := lo.If(side == LONG, paradex.PositionSideLong).Else(paradex.PositionSideShort)
	position, ok := lo.Find(positions.Results, func(item *paradex.Position) bool {
		return item.Market == market && item.Side == positionSide && !item.Size.IsZero()
	})
	if !ok {
		return nil
	}

	// 查询当前价格
	price, err := GetLastTradePrice(ctx, h.svcCtx, exchange.Paradex, symbol)
	if err != nil {
		return err
	}

	switch position.Side {
	case paradex.PositionSideLong:
		acceptableExecutionPrice := price.Sub(price.Mul(decimal.NewFromInt(int64(slippageBps)).Div(decimal.NewFromInt(10000))))
		_, err = h.CreateMarketOrder(ctx, symbol, true, true, acceptableExecutionPrice, position.Size.Abs())
	case paradex.PositionSideShort:
		acceptableExecutionPrice := price.Add(price.Mul(decimal.NewFromInt(int64(slippageBps)).Div(decimal.NewFromInt(10000))))
		_, err = h.CreateMarketOrder(ctx, symbol, false, true, acceptableExecutionPrice, position.Size.Abs())
	}

	if err != nil {
		logger.Errorf("[ParadexOrderHelper] 关闭仓位失败, account: %s, symbol: %s, size: %s, %v", h.userClient.DexAccount(), symbol, position.Size.Abs(), err)
	}

	return err
}
