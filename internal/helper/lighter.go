package helper

import (
	"context"
	"errors"
	"strconv"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/shopspring/decimal"
)

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

func GetSupportedDecimals(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) (sizeDecimals, priceDecimals uint8, minBaseAmount decimal.Decimal, err error) {
	switch record.Exchange {
	case exchange.Lighter:
		metadata, err := svcCtx.LighterCache.GetOrderBookMetadata(ctx, record.Symbol)
		if err != nil {
			return 0, 0, decimal.Zero, err
		}
		return metadata.SupportedSizeDecimals, metadata.SupportedPriceDecimals, metadata.MinBaseAmount, nil
	default:
		return 0, 0, decimal.Zero, errors.New("exchange unsupported")
	}
}
