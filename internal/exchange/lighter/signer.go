package lighter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elliottech/lighter-go/signer"
	"github.com/elliottech/lighter-go/types"
)

type Signer struct {
	client       *Client
	accountIndex int64
	apiKeyIndex  uint8
	keyManager   signer.KeyManager
}

func NewSigner(client *Client, accountIndex int64, apiKeyPrivateKey string, apiKeyIndex uint8) (*Signer, error) {
	if len(apiKeyPrivateKey) < 2 {
		return nil, fmt.Errorf("empty private key")
	}
	if apiKeyPrivateKey[:2] == "0x" {
		apiKeyPrivateKey = apiKeyPrivateKey[2:]
	}
	b, err := hex.DecodeString(apiKeyPrivateKey)
	if err != nil {
		return nil, err
	}

	keyManager, err := signer.NewKeyManager(b)
	if err != nil {
		return nil, err
	}

	signer := &Signer{
		client:       client,
		accountIndex: accountIndex,
		apiKeyIndex:  apiKeyIndex,
		keyManager:   keyManager,
	}
	return signer, nil
}

func (c *Signer) Client() *Client {
	return c.client
}

func (c *Signer) GetAccountIndex() int64 {
	return c.accountIndex
}

func (c *Signer) GetApiKeyIndex() uint8 {
	return c.apiKeyIndex
}

func (c *Signer) GetAccountTxs(ctx context.Context, accountIndex int64, limit int) (Transactions, error) {
	token, err := c.GetAuthToken(time.Now().Add(time.Second * 30))
	if err != nil {
		return Transactions{}, err
	}

	var result Transactions
	err = getAndParseL2HTTPResponse(
		ctx,
		c.client.httpClient,
		c.client.endpoint,
		"api/v1/accountTxs",
		map[string]any{"auth": token, "by": "account_index", "value": accountIndex, "limit": limit},
		&result,
	)
	if err != nil {
		return Transactions{}, err
	}
	return result, nil
}

func (c *Signer) GetAccountActiveOrders(ctx context.Context, marketId uint) (Orders, error) {
	token, err := c.GetAuthToken(time.Now().Add(time.Second * 30))
	if err != nil {
		return Orders{}, err
	}

	var result Orders
	err = getAndParseL2HTTPResponse(
		ctx,
		c.client.httpClient,
		c.client.endpoint,
		"api/v1/accountActiveOrders",
		map[string]any{"auth": token, "account_index": c.accountIndex, "market_id": marketId},
		&result,
	)
	if err != nil {
		return Orders{}, err
	}
	return result, nil
}

func (c *Signer) GetAccountInactiveOrders(ctx context.Context, cursor string, limit int) (Orders, error) {
	token, err := c.GetAuthToken(time.Now().Add(time.Second * 30))
	if err != nil {
		return Orders{}, err
	}

	var result Orders
	err = getAndParseL2HTTPResponse(
		ctx,
		c.client.httpClient,
		c.client.endpoint,
		"api/v1/accountInactiveOrders",
		map[string]any{"auth": token, "account_index": c.accountIndex, "cursor": cursor, "limit": limit},
		&result,
	)
	if err != nil {
		return Orders{}, err
	}
	return result, nil
}

func (c *Signer) SignCreateOrder(ctx context.Context, req *CreateOrderTxReq, nonce int64) (string, error) {
	tx := &types.CreateOrderTxReq{
		MarketIndex:      req.MarketIndex,
		ClientOrderIndex: req.ClientOrderIndex,
		BaseAmount:       req.BaseAmount,
		Price:            req.Price,
		IsAsk:            req.IsAsk,
		Type:             uint8(req.Type),
		TimeInForce:      uint8(req.TimeInForce),
		ReduceOnly:       req.ReduceOnly,
		TriggerPrice:     req.TriggerPrice,
		OrderExpiry:      req.OrderExpiry,
	}

	ops := new(types.TransactOpts)
	ops.Nonce = &nonce
	ops.ApiKeyIndex = &c.apiKeyIndex
	ops.FromAccountIndex = &c.accountIndex
	ops.ExpiredAt = time.Now().Add(time.Minute*10 - time.Second).UnixMilli()

	txInfo, err := types.ConstructCreateOrderTx(c.keyManager, c.client.chainId, tx, ops)
	if err != nil {
		return "", err
	}

	txInfoBytes, err := json.Marshal(txInfo)
	if err != nil {
		return "", err
	}

	txInfoStr := string(txInfoBytes)
	return txInfoStr, nil
}

func (c *Signer) SignCancelOrder(ctx context.Context, marketIndex uint8, orderIndex, nonce int64) (string, error) {
	tx := &types.CancelOrderTxReq{
		MarketIndex: marketIndex,
		Index:       orderIndex,
	}

	ops := new(types.TransactOpts)
	ops.Nonce = &nonce
	ops.ApiKeyIndex = &c.apiKeyIndex
	ops.FromAccountIndex = &c.accountIndex
	ops.ExpiredAt = time.Now().Add(time.Minute*10 - time.Second).UnixMilli()

	txInfo, err := types.ConstructL2CancelOrderTx(c.keyManager, c.client.chainId, tx, ops)
	if err != nil {
		return "", err
	}

	txInfoBytes, err := json.Marshal(txInfo)
	if err != nil {
		return "", err
	}

	txInfoStr := string(txInfoBytes)
	return txInfoStr, nil
}

func (c *Signer) SignUpdateLeverage(ctx context.Context, req *UpdateLeverageTxReq, nonce int64) (string, error) {
	imf := uint16(10_000 / req.Leverage)
	tx := &types.UpdateLeverageTxReq{
		MarketIndex:           req.MarketIndex,
		InitialMarginFraction: imf,
		MarginMode:            uint8(req.MarginMode),
	}

	ops := new(types.TransactOpts)
	ops.Nonce = &nonce
	ops.ApiKeyIndex = &c.apiKeyIndex
	ops.FromAccountIndex = &c.accountIndex
	ops.ExpiredAt = time.Now().Add(time.Minute*10 - time.Second).UnixMilli()

	txInfo, err := types.ConstructUpdateLeverageTx(c.keyManager, c.client.chainId, tx, ops)
	if err != nil {
		return "", err
	}

	txInfoBytes, err := json.Marshal(txInfo)
	if err != nil {
		return "", err
	}

	txInfoStr := string(txInfoBytes)
	return txInfoStr, nil
}

func (c *Signer) GetAuthToken(deadline time.Time) (string, error) {
	if time.Until(deadline) > (7 * time.Hour) {
		return "", fmt.Errorf("deadline should be within 7 hours")
	}

	return types.ConstructAuthToken(c.keyManager, deadline, &types.TransactOpts{
		ApiKeyIndex:      &c.apiKeyIndex,
		FromAccountIndex: &c.accountIndex,
	})
}
