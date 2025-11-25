package helper

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

func ParseUsdPerpMarket(market string) (string, error) {
	parts := strings.Split(market, "-")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid market format: %s", market)
	}

	baseCurrency := parts[0]
	quoteCurrency := parts[1]
	contractType := parts[2]

	if strings.ToUpper(quoteCurrency) != "USD" {
		return "", fmt.Errorf("not a USD trading pair: %s", market)
	}

	if !strings.HasSuffix(strings.ToUpper(contractType), "PERP") {
		return "", fmt.Errorf("not a perpetual market: %s", market)
	}

	return baseCurrency, nil
}

func FormatUsdPerpMarket(baseCurrency string) string {
	return fmt.Sprintf("%s-USD-PERP", strings.ToUpper(baseCurrency))
}

func GetParadexClient(svcCtx *svc.ServiceContext, record *ent.Strategy) (*paradex.UserClient, error) {
	dexAccount := record.ExchangeApiKey
	dexPrivateKey := record.ExchangeSecretKey
	return paradex.NewUserClient(svcCtx.ParadexClient, dexAccount, dexPrivateKey), nil
}

func NewParadexOrderHelper(svcCtx *svc.ServiceContext, userClient *paradex.UserClient) *ParadexOrderHelper {
	return &ParadexOrderHelper{svcCtx: svcCtx, userClient: userClient}
}

func (h *ParadexOrderHelper) SyncUserOrders(ctx context.Context) error {
	account := h.userClient.DexAccount()
	logger.Debugf("[ParadexOrderHelper] 同步用户订单开始, account: %s", account)

	syncProgress, err := h.svcCtx.SyncProgressModel.Ensure(ctx, exchange.Paradex, account)
	if err != nil {
		return err
	}

	// 查询成交记录
	var startAt *time.Time
	if syncProgress.Timestamp != 0 {
		ts := time.UnixMilli(syncProgress.Timestamp + 1)
		startAt = &ts
	}

	cursor := ""
	const limit = 100
	userFills := make([]*paradex.Fill, 0)

	for {
		logger.Debugf("[ParadexOrderHelper] 查询用户成交记录开始, account: %s, cursor: %s, limit: %d", account, cursor, limit)
		fillsRes, err := h.userClient.ListFills(ctx, startAt, cursor, limit)
		if err != nil {
			logger.Debugf("[ParadexOrderHelper] 查询用户成交记录失败, account: %s, cursor: %s, limit: %d, %v", account, cursor, limit, err)
			return err
		}
		logger.Debugf("[ParadexOrderHelper] 查询用户成交记录结束, account: %s, cursor: %s, limit: %d", account, cursor, limit)

		cursor = fillsRes.Next
		userFills = append(userFills, fillsRes.Results...)
		if cursor == "" {
			break
		}
	}

	// 记录订单ID
	orderIdSet := make(map[string]struct{})
	for _, fill := range userFills {
		orderIdSet[fill.OrderID] = struct{}{}
	}

	// 同步所有订单
	userOrders := make([]*paradex.Order, 0)
	if len(orderIdSet) > 0 {
		for {
			logger.Debugf("[ParadexOrderHelper] 查询用户订单记录开始, account: %s, cursor: %s, limit: %d", account, cursor, limit)
			userOrdersRes, err := h.userClient.GetUserOrders(ctx, cursor, limit)
			if err != nil {
				logger.Debugf("[ParadexOrderHelper] 查询用户订单记录失败, account: %s, cursor: %s, limit: %d, %v", account, cursor, limit, err)
				return err
			}
			logger.Debugf("[ParadexOrderHelper] 查询用户订单记录结束, account: %s, cursor: %s, limit: %d", account, cursor, limit)

			for idx, ord := range userOrdersRes.Results {
				if len(orderIdSet) == 0 {
					userOrdersRes.Results = userOrdersRes.Results[:idx]
					break
				}

				delete(orderIdSet, ord.ID)

				if len(orderIdSet) == 0 {
					userOrdersRes.Results = userOrdersRes.Results[:idx+1]
					break
				}
			}

			cursor = userOrdersRes.Next
			userOrders = append(userOrders, userOrdersRes.Results...)
			if cursor == "" || len(orderIdSet) == 0 {
				break
			}
		}
	}

	logger.Debugf("[ParadexOrderHelper] 同步用户订单结束, account: %s, count: %d", account, len(userOrders))

	// 本地化存储用户订单
	return util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, item := range userOrders {
			symbol, err := ParseUsdPerpMarket(item.Market)
			if err != nil {
				continue
			}

			var clientId int64
			if item.ClientID != "" {
				clientId, err = strconv.ParseInt(item.ClientID, 10, 64)
				if err != nil {
					clientId = 0
				}
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
				ClientOrderId:     clientId,
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

		if len(userFills) > 0 {
			ts := userFills[0].CreatedAt
			err = h.svcCtx.SyncProgressModel.UpdateTimestampByAccount(ctx, exchange.Paradex, account, ts)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
