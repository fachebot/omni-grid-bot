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

// Signer Lighter交易签名器
// 负责对交易请求进行签名，生成可发送到链上的交易信息
type Signer struct {
	client       *Client           // HTTP客户端
	accountIndex int64             // 账户索引
	apiKeyIndex  uint8             // API密钥索引
	keyManager   signer.KeyManager // 密钥管理器
}

// NewSigner 创建Lighter签名器
// client HTTP客户端，accountIndex 账户索引，apiKeyPrivateKey API密钥私钥，apiKeyIndex API密钥索引
func NewSigner(client *Client, accountIndex int64, apiKeyPrivateKey string, apiKeyIndex uint8) (*Signer, error) {
	if len(apiKeyPrivateKey) < 2 {
		return nil, fmt.Errorf("empty private key")
	}
	if apiKeyPrivateKey[:2] == "0x" ||
		apiKeyPrivateKey[:2] == "0X" {
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

// Client 获取HTTP客户端
func (c *Signer) Client() *Client {
	return c.client
}

// GetAccountIndex 获取账户索引
func (c *Signer) GetAccountIndex() int64 {
	return c.accountIndex
}

// GetApiKeyIndex 获取API密钥索引
func (c *Signer) GetApiKeyIndex() uint8 {
	return c.apiKeyIndex
}

// GetAccountTxs 获取账户交易记录
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
	if result.Code != 200 {
		return Transactions{}, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}
	return result, nil
}

// GetAccountActiveOrders 获取账户活跃订单
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
	if result.Code != 200 {
		return Orders{}, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}
	return result, nil
}

// GetAccountInactiveOrders 获取账户非活跃订单
// 包括已成交、已取消、已过期的订单
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
	if result.Code != 200 {
		return Orders{}, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}
	return result, nil
}

// SignCreateOrder 签名创建订单请求
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

// SignCancelOrder 签名取消订单请求
func (c *Signer) SignCancelOrder(ctx context.Context, marketIndex int16, orderIndex, nonce int64) (string, error) {
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

// SignUpdateLeverage 签名更新杠杆请求
func (c *Signer) SignUpdateLeverage(ctx context.Context, req *UpdateLeverageTxReq, nonce int64) (string, error) {
	imf := uint16(10000 / req.Leverage)
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

// GetAuthToken 获取认证令牌
// 用于需要认证的API调用
func (c *Signer) GetAuthToken(deadline time.Time) (string, error) {
	if time.Until(deadline) > (7 * time.Hour) {
		return "", fmt.Errorf("deadline should be within 7 hours")
	}

	return types.ConstructAuthToken(c.keyManager, deadline, &types.TransactOpts{
		ApiKeyIndex:      &c.apiKeyIndex,
		FromAccountIndex: &c.accountIndex,
	})
}
