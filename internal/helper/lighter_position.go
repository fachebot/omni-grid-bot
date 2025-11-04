package helper

import (
	"context"
	"errors"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/shopspring/decimal"
)

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
