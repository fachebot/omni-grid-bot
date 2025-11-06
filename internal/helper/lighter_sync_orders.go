package helper

import (
	"context"
	"strconv"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
	"github.com/samber/lo"
)

func (h *LighterOrderHelper) SyncInactiveOrders(ctx context.Context) error {
	logger.Debugf("[LighterOrderHelper] 同步用户非活跃订单开始, account: %d", h.signer.GetAccountIndex())

	cursor := ""
	const limit = 100

	account := strconv.Itoa(int(h.signer.GetAccountIndex()))
	syncProgress, err := h.svcCtx.SyncProgressModel.Ensure(ctx, exchange.Lighter, account)
	if err != nil {
		return err
	}

	// 查询未活跃订单列表
	inactiveOrders := make([]*lighter.Order, 0)
exit:
	for {
		logger.Debugf("[LighterOrderHelper] 查询用户非活跃订单开始, account: %d, cursor: %s, limit: %d", h.signer.GetAccountIndex(), cursor, limit)
		orders, err := h.signer.GetAccountInactiveOrders(ctx, cursor, limit)
		if err != nil {
			return err
		}
		logger.Debugf("[LighterOrderHelper] 查询用户非活跃订单结束, account: %d, cursor: %s, limit: %d", h.signer.GetAccountIndex(), cursor, limit)

		for _, ord := range orders.Orders {
			if syncProgress.Timestamp > 0 &&
				ord.Timestamp <= syncProgress.Timestamp {
				break exit
			}
			inactiveOrders = append(inactiveOrders, ord)
		}

		cursor = orders.NextCursor
		if len(orders.Orders) < limit {
			break
		}
	}

	// 查询市场ID对应的symbol
	markets := make(map[uint8]string)
	for _, item := range inactiveOrders {
		symbol, err := h.svcCtx.LighterCache.GetSymbolByMarketId(ctx, item.MarketIndex)
		if err != nil {
			return err
		}
		markets[item.MarketIndex] = symbol
	}

	logger.Debugf("[LighterOrderHelper] 同步用户非活跃订单结束, account: %d, count: %d", h.signer.GetAccountIndex(), len(inactiveOrders))

	// 本地化存储未活跃订单
	return util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, item := range inactiveOrders {
			symbol := markets[item.MarketIndex]
			args := ent.Order{
				Exchange:          exchange.Lighter,
				Account:           account,
				Symbol:            symbol,
				OrderID:           item.OrderIndex,
				ClientOrderID:     item.ClientOrderIndex,
				Side:              lo.If(item.IsAsk, "sell").Else("buy"),
				Price:             item.Price,
				BaseAmount:        item.InitialBaseAmount,
				FilledBaseAmount:  item.FilledBaseAmount,
				FilledQuoteAmount: item.FilledQuoteAmount,
				Status:            lighter.ConvertOrderStatus(item.Status),
				Timestamp:         item.Timestamp,
			}
			err = h.svcCtx.OrderModel.Upsert(ctx, args)
			if err != nil {
				return err
			}
		}

		if len(inactiveOrders) > 0 {
			ts := inactiveOrders[0].Timestamp
			err = h.svcCtx.SyncProgressModel.UpdateTimestampByAccount(ctx, exchange.Lighter, account, ts)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
