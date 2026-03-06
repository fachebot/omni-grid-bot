package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	signer     *Signer
}

func NewClient(httpClient *http.Client, signer *Signer, isMainnet bool) *Client {
	url := MainnetBaseURL
	if !isMainnet {
		url = TestnetBaseURL
	}

	return &Client{
		baseURL:    url,
		httpClient: httpClient,
		signer:     signer,
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Signer() *Signer {
	return c.signer
}

func (c *Client) UserState(ctx context.Context, address string) (*AccountState, error) {
	payload := map[string]interface{}{
		"type": "clearinghouseState",
		"user": address,
	}

	var result AccountState
	err := c.post(ctx, "/info", payload, &result)
	return &result, err
}

func (c *Client) OpenOrders(ctx context.Context, address string) ([]Order, error) {
	payload := map[string]interface{}{
		"type": "openOrders",
		"user": address,
	}

	var result []Order
	err := c.post(ctx, "/info", payload, &result)
	return result, err
}

func (c *Client) UserFills(ctx context.Context, address string) ([]Fill, error) {
	payload := map[string]interface{}{
		"type": "userFills",
		"user": address,
	}

	var result []Fill
	err := c.post(ctx, "/info", payload, &result)
	return result, err
}

func (c *Client) Meta(ctx context.Context) (*Meta, error) {
	payload := map[string]interface{}{
		"type": "meta",
	}

	var result Meta
	err := c.post(ctx, "/info", payload, &result)
	return &result, err
}

func (c *Client) MetaAndAssetCtxs(ctx context.Context) (*MetaAndAssetCtxs, error) {
	payload := map[string]interface{}{
		"type": "metaAndAssetCtxs",
	}

	var result []json.RawMessage
	err := c.post(ctx, "/info", payload, &result)
	if err != nil {
		return nil, err
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid response")
	}

	var meta Meta
	if err := json.Unmarshal(result[0], &meta); err != nil {
		return nil, err
	}

	var assetCtxs []AssetContext
	if err := json.Unmarshal(result[1], &assetCtxs); err != nil {
		return nil, err
	}

	return &MetaAndAssetCtxs{
		Meta:      meta,
		AssetCtxs: assetCtxs,
	}, nil
}

func (c *Client) AllMids(ctx context.Context) (map[string]decimal.Decimal, error) {
	payload := map[string]interface{}{
		"type": "allMids",
	}

	var result map[string]interface{}
	err := c.post(ctx, "/info", payload, &result)
	if err != nil {
		return nil, err
	}

	mids := make(map[string]decimal.Decimal)
	for k, v := range result {
		switch val := v.(type) {
		case float64:
			mids[k] = decimal.NewFromFloat(val)
		case string:
			mids[k] = ParseDecimal(val)
		}
	}
	return mids, nil
}

func (c *Client) PlaceOrder(ctx context.Context, order *OrderRequest) (*OrderResponse, error) {
	asset := CoinToAsset(order.Coin, nil)

	orderWire := map[string]interface{}{
		"a": asset,
		"b": order.IsBuy,
		"p": order.LimitPx,
		"s": order.Sz,
		"r": order.ReduceOnly,
		"t": map[string]string{"limit": order.OrderType.Limit.Tif},
	}

	if order.Cloid != "" {
		orderWire["c"] = order.Cloid
	}

	action := map[string]interface{}{
		"type":     "order",
		"orders":   []map[string]interface{}{orderWire},
		"grouping": "na",
	}

	nonce := time.Now().UnixMilli()
	signature, err := c.signer.SignOrder(action, nonce)
	if err != nil {
		return nil, err
	}

	reqPayload := map[string]interface{}{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}

	var result OrderResponse
	err = c.post(ctx, "/exchange", reqPayload, &result)
	return &result, err
}

func (c *Client) CancelOrder(ctx context.Context, coin string, oid int64) error {
	asset := CoinToAsset(coin, nil)

	action := map[string]interface{}{
		"type": "cancel",
		"cancels": []map[string]interface{}{
			{"a": asset, "o": oid},
		},
	}

	nonce := time.Now().UnixMilli()
	signature, err := c.signer.SignCancel(action, nonce)
	if err != nil {
		return err
	}

	reqPayload := map[string]interface{}{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}

	var result OrderResponse
	return c.post(ctx, "/exchange", reqPayload, &result)
}

func (c *Client) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error {
	action := map[string]interface{}{
		"type":     "updateLeverage",
		"asset":    asset,
		"isCross":  isCross,
		"leverage": leverage,
	}

	nonce := time.Now().UnixMilli()
	signature, err := c.signer.SignAction(action, nonce)
	if err != nil {
		return err
	}

	reqPayload := map[string]interface{}{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}

	var result OrderResponse
	return c.post(ctx, "/exchange", reqPayload, &result)
}

func (c *Client) post(ctx context.Context, path string, payload interface{}, result interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result == nil {
		return nil
	}

	return json.Unmarshal(body, result)
}
