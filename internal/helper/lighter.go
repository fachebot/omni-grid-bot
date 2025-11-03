package helper

import (
	"errors"
	"strconv"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
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
