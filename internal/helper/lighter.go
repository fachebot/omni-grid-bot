package helper

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

const (
	ClientOrderIndexBegin = 77766600000000
)

type LighterOrderHelper struct {
	signer *lighter.Signer
	svcCtx *svc.ServiceContext
}

func GetLighterClient(svcCtx *svc.ServiceContext, record *ent.Strategy) (*lighter.Signer, error) {
	accountIndex, err := strconv.Atoi(record.ExchangeApiKey)
	if err != nil {
		return nil, err
	}

	apiKeyIndex, err := strconv.Atoi(record.ExchangeSecretKey)
	if err != nil {
		return nil, err
	}

	if len(record.ExchangePassphrase) != 80 {
		return nil, errors.New("invalid apiKeyPrivateKey")
	}

	return lighter.NewSigner(svcCtx.LighterClient, int64(accountIndex), record.ExchangePassphrase, uint8(apiKeyIndex))
}

func NewLighterOrderHelper(svcCtx *svc.ServiceContext, signer *lighter.Signer) *LighterOrderHelper {
	return &LighterOrderHelper{svcCtx: svcCtx, signer: signer}
}

func (h *LighterOrderHelper) UpdateLeverage(ctx context.Context, symbol string, leverage uint, marginMode exchange.MarginMode) error {
	metadata, err := h.svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get order book metadata: %w", err)
	}

	nonce, err := h.signer.Client().GetNextNonce(ctx, h.signer.GetAccountIndex(), h.signer.GetApiKeyIndex())
	if err != nil {
		return err
	}

	req := &lighter.UpdateLeverageTxReq{
		MarketIndex: metadata.MarketID,
		Leverage:    leverage,
		MarginMode:  marginMode,
	}
	txInfo, err := h.signer.SignUpdateLeverage(ctx, req, nonce)
	if err != nil {
		return err
	}

	_, err = h.signer.Client().SendRawTx(ctx, lighter.TX_TYPE_UPDATE_LEVERAGE, txInfo)
	return err
}

func (h *LighterOrderHelper) CancelOrderBatch(ctx context.Context, orders []CancelOrderParams) error {
	if len(orders) == 0 {
		return nil
	}

	nonce, err := h.signer.Client().GetNextNonce(ctx, h.signer.GetAccountIndex(), h.signer.GetApiKeyIndex())
	if err != nil {
		return err
	}

	txInfos := make([]string, 0, len(orders))
	txTypes := make([]lighter.TX_TYPE, 0, len(orders))
	for _, item := range orders {
		txInfo, err := h.signCancelOrder(ctx, item.Symbol, item.OrderID, nonce)
		if err != nil {
			return err
		}

		nonce += 1
		txInfos = append(txInfos, txInfo)
		txTypes = append(txTypes, lighter.TX_TYPE_CANCEL_ORDER)
	}

	_, err = h.signer.Client().SendRawTxBatch(ctx, txTypes, txInfos)
	return err
}

func (h *LighterOrderHelper) CancalAllOrders(ctx context.Context, symbol string) error {
	metadata, err := h.svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get order book metadata: %w", err)
	}

	orders, err := h.signer.GetAccountActiveOrders(ctx, uint(metadata.MarketID))
	if err != nil {
		return nil
	}

	if len(orders.Orders) == 0 {
		return nil
	}

	var cancelOrders []CancelOrderParams
	for _, ord := range orders.Orders {
		orderId, err := strconv.ParseInt(ord.OrderID, 10, 64)
		if err != nil {
			continue
		}
		cancelOrders = append(cancelOrders, CancelOrderParams{Symbol: symbol, OrderID: orderId})
	}
	return h.CancelOrderBatch(ctx, cancelOrders)
}

func (h *LighterOrderHelper) CancelOrder(ctx context.Context, symbol string, orderIndex int64) error {
	return h.CancelOrderBatch(ctx, []CancelOrderParams{{Symbol: symbol, OrderID: orderIndex}})
}

func (h *LighterOrderHelper) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]int64, []int64, error) {
	nonce, err := h.signer.Client().GetNextNonce(ctx, h.signer.GetAccountIndex(), h.signer.GetApiKeyIndex())
	if err != nil {
		return nil, nil, err
	}

	limitClientOrderIds := make([]int64, 0)
	txInfos := make([]string, 0, len(limitOrders)+len(marketOrders))
	txTypes := make([]lighter.TX_TYPE, 0, len(limitOrders)+len(marketOrders))
	for _, item := range limitOrders {
		clientOrderIndex := ClientOrderIndexBegin + nonce
		txInfo, err := h.signCreateLimitOrder(ctx, item.Symbol, item.IsAsk, item.ReduceOnly, item.Price, item.Size, clientOrderIndex, nonce)
		if err != nil {
			return nil, nil, err
		}

		nonce += 1
		txInfos = append(txInfos, txInfo)
		txTypes = append(txTypes, lighter.TX_TYPE_CREATE_ORDER)

		limitClientOrderIds = append(limitClientOrderIds, clientOrderIndex)
	}

	marketClientOrderIds := make([]int64, 0)
	for _, item := range marketOrders {
		clientOrderIndex := ClientOrderIndexBegin + nonce
		txInfo, err := h.signCreateMarketOrder(ctx, item.Symbol, item.IsAsk, item.ReduceOnly, item.AcceptableExecutionPrice, item.Size, clientOrderIndex, nonce)
		if err != nil {
			return nil, nil, err
		}

		nonce += 1
		txInfos = append(txInfos, txInfo)
		txTypes = append(txTypes, lighter.TX_TYPE_CREATE_ORDER)

		marketClientOrderIds = append(marketClientOrderIds, clientOrderIndex)
	}

	_, err = h.signer.Client().SendRawTxBatch(ctx, txTypes, txInfos)
	if err != nil {
		return nil, nil, err
	}

	return limitClientOrderIds, marketClientOrderIds, nil
}

func (h *LighterOrderHelper) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (int64, error) {
	clientOrderIds, _, err := h.CreateOrderBatch(ctx, []CreateLimitOrderParams{{
		Symbol:     symbol,
		IsAsk:      isAsk,
		ReduceOnly: reduceOnly,
		Price:      price,
		Size:       size,
	}}, nil)
	if err != nil {
		return 0, err
	}

	return clientOrderIds[0], nil
}

func (h *LighterOrderHelper) CreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, acceptableExecutionPrice, size decimal.Decimal) (int64, error) {
	_, clientOrderIds, err := h.CreateOrderBatch(ctx, nil, []CreateMarketOrderParams{{
		Symbol:                   symbol,
		IsAsk:                    isAsk,
		ReduceOnly:               reduceOnly,
		AcceptableExecutionPrice: acceptableExecutionPrice,
		Size:                     size,
	}})
	if err != nil {
		return 0, err
	}

	return clientOrderIds[0], nil
}

func (h *LighterOrderHelper) signCancelOrder(ctx context.Context, symbol string, orderIndex int64, nonce int64) (string, error) {
	metadata, err := h.svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get order book metadata: %w", err)
	}

	return h.signer.SignCancelOrder(ctx, metadata.MarketID, orderIndex, nonce)
}

func (h *LighterOrderHelper) signCreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal, clientOrderIndex int64, nonce int64) (string, error) {
	metadata, err := h.svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get order book metadata: %w", err)
	}

	if size.LessThan(metadata.MinBaseAmount) {
		return "", fmt.Errorf("order size %s is less than the minimum base amount %s",
			size.String(), metadata.MinBaseAmount.String())
	}

	sizeN := decimal.NewFromBigInt(util.FormatUnits(size, metadata.SupportedSizeDecimals), 0).IntPart()
	if size.LessThanOrEqual(decimal.Zero) {
		return "", errors.New("order size must be greater than zero")
	}

	priceN := decimal.NewFromBigInt(util.FormatUnits(price, metadata.SupportedPriceDecimals), 0).IntPart()
	if price.LessThanOrEqual(decimal.Zero) {
		return "", errors.New("order price must be greater than zero")
	}

	req := &lighter.CreateOrderTxReq{
		MarketIndex:      metadata.MarketID,
		ClientOrderIndex: clientOrderIndex,
		BaseAmount:       sizeN,
		Price:            uint32(priceN),
		IsAsk:            uint8(lo.If(isAsk, 1).Else(0)),
		Type:             lighter.ORDER_TYPE_LIMIT,
		TimeInForce:      lighter.ORDER_TIME_IN_FORCE_GOOD_TILL_TIME,
		ReduceOnly:       uint8(lo.If(reduceOnly, 1).Else(0)),
		OrderExpiry:      time.Now().Add(time.Hour * 24 * 28).UnixMilli(),
	}
	return h.signer.SignCreateOrder(ctx, req, nonce)
}

func (h *LighterOrderHelper) signCreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, acceptableExecutionPrice, size decimal.Decimal, clientOrderIndex int64, nonce int64) (string, error) {
	metadata, err := h.svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get order book metadata: %w", err)
	}

	if size.LessThan(metadata.MinBaseAmount) {
		return "", fmt.Errorf("order size %s is less than the minimum base amount %s",
			size.String(), metadata.MinBaseAmount.String())
	}

	sizeN := decimal.NewFromBigInt(util.FormatUnits(size, metadata.SupportedSizeDecimals), 0).IntPart()
	if size.LessThanOrEqual(decimal.Zero) {
		return "", errors.New("order size must be greater than zero")
	}

	priceN := decimal.NewFromBigInt(util.FormatUnits(acceptableExecutionPrice, metadata.SupportedPriceDecimals), 0).IntPart()
	if acceptableExecutionPrice.LessThanOrEqual(decimal.Zero) {
		return "", errors.New("order price must be greater than zero")
	}

	req := &lighter.CreateOrderTxReq{
		MarketIndex:      metadata.MarketID,
		ClientOrderIndex: clientOrderIndex,
		BaseAmount:       sizeN,
		Price:            uint32(priceN),
		IsAsk:            uint8(lo.If(isAsk, 1).Else(0)),
		Type:             lighter.ORDER_TYPE_MARKET,
		TimeInForce:      lighter.ORDER_TIME_IN_FORCE_IMMEDIATE_OR_CANCEL,
		ReduceOnly:       uint8(lo.If(reduceOnly, 1).Else(0)),
		OrderExpiry:      0,
	}
	return h.signer.SignCreateOrder(ctx, req, nonce)
}

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
				OrderId:           item.OrderIndex,
				ClientOrderId:     item.ClientOrderIndex,
				Side:              lo.If(item.IsAsk, order.SideSell).Else(order.SideBuy),
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

func (h *LighterOrderHelper) ClosePosition(ctx context.Context, symbol string, side Side, slippageBps int) error {
	metadata, err := h.svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get order book metadata: %w", err)
	}

	// 查询账户信息
	accounts, err := h.svcCtx.LighterClient.GetAccountByIndex(ctx, h.signer.GetAccountIndex())
	if err != nil {
		return err
	}

	var account *lighter.Account
	for _, item := range accounts.Accounts {
		if item.AccountIndex == h.signer.GetAccountIndex() {
			account = item
			break
		}
	}

	if account == nil {
		return errors.New("account not found")
	}

	// 查找指定仓位
	var p *lighter.Position
	for _, item := range account.Positions {
		if item.MarketID == metadata.MarketID && item.Sign == int32(side) {
			p = item
			break
		}
	}

	if p == nil || p.Position.IsZero() {
		return nil
	}

	// 查询当前价格
	price, err := h.svcCtx.LighterClient.GetLastTradePrice(ctx, uint(metadata.MarketID))
	if err != nil {
		return err
	}

	switch p.Sign {
	case 1:
		acceptableExecutionPrice := price.Sub(price.Mul(decimal.NewFromInt(10000).Mul(decimal.NewFromInt(int64(slippageBps)))))
		_, err = h.CreateMarketOrder(ctx, symbol, true, true, acceptableExecutionPrice, p.Position)
	case -1:
		acceptableExecutionPrice := price.Add(price.Mul(decimal.NewFromInt(10000).Mul(decimal.NewFromInt(int64(slippageBps)))))
		_, err = h.CreateMarketOrder(ctx, symbol, false, true, acceptableExecutionPrice, p.Position)
	}

	return err
}
