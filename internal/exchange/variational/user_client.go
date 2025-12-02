package variational

import (
	"context"
	"fmt"

	"github.com/enetx/g"
	httpx "github.com/enetx/http"
	"github.com/shopspring/decimal"
)

type UserClient struct {
	client        *Client
	ethAccount    string
	ethPrivateKey string
}

func NewUserClient(client *Client, ethAccount, ethPrivateKey string) *UserClient {
	return &UserClient{client: client, ethAccount: ethAccount, ethPrivateKey: ethPrivateKey}
}

func (c *UserClient) EthAccount() string {
	return c.ethAccount
}

func (c *UserClient) EnsureJwtToken(ctx context.Context) (string, error) {
	return c.client.EnsureJwtToken(ctx, c.ethAccount, c.ethPrivateKey)
}

func (c *UserClient) GetPositions(ctx context.Context) ([]*Position, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/positions", c.client.endpoint)
	res := c.client.client.Get(g.String(url)).WithContext(ctx).AddCookies(cookies...).Do()

	var positions []*Position
	if err := c.client.parseRespone(res, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

func (c *UserClient) GetPortfolio(ctx context.Context, computeMargin bool) (*PortfolioRes, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/portfolio?compute_margin=%v", c.client.endpoint, computeMargin)
	res := c.client.client.Get(g.String(url)).WithContext(ctx).AddCookies(cookies...).Do()

	var portfolio PortfolioRes
	if err := c.client.parseRespone(res, &portfolio); err != nil {
		return nil, err
	}
	return &portfolio, nil
}

func (c *UserClient) GetUserOrders(ctx context.Context, offset, limit int) (*OrdersRes, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/orders/v2?order_by=created_at&limit=%d&offset=%d&order=desc", c.client.endpoint, limit, offset)
	res := c.client.client.Get(g.String(url)).WithContext(ctx).AddCookies(cookies...).Do()

	var orders OrdersRes
	if err := c.client.parseRespone(res, &orders); err != nil {
		return nil, err
	}
	return &orders, nil
}

func (c *UserClient) GetUserPendingOrders(ctx context.Context, offset, limit int) (*OrdersRes, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/orders/v2?status=pending&order_by=created_at&limit=%d&offset=%d&order=desc", c.client.endpoint, limit, offset)
	res := c.client.client.Get(g.String(url)).WithContext(ctx).AddCookies(cookies...).Do()

	var orders OrdersRes
	if err := c.client.parseRespone(res, &orders); err != nil {
		return nil, err
	}
	return &orders, nil
}

func (c *UserClient) SetLeverage(ctx context.Context, asset string, leverage int) error {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/settlement_pools/set_leverage", c.client.endpoint)
	payload := fmt.Sprintf(`{"leverage":"%d","asset":"%s"}`, leverage, asset)
	res := c.client.client.Post(g.String(url), payload).WithContext(ctx).AddCookies(cookies...).Do()

	var r SetLeverageRes
	if err := c.client.parseRespone(res, &r); err != nil {
		return err
	}
	return nil
}

func (c *UserClient) IndicativeQuote(ctx context.Context, symbol string, qty decimal.Decimal) (*IndicativeQuoteRes, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}

	url := fmt.Sprintf("%s/quotes/indicative", c.client.endpoint)
	payload := fmt.Sprintf(`{"instrument":{"underlying":"%s","funding_interval_s":3600,"settlement_asset":"USDC","instrument_type":"perpetual_future"},"qty":"%s"}`, symbol, qty)
	res := c.client.client.Post(g.String(url), payload).WithContext(ctx).AddCookies(cookies...).Do()

	var r IndicativeQuoteRes
	if err := c.client.parseRespone(res, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *UserClient) CreateLimitOrder(ctx context.Context, symbol string, side OrderSide, limitPrice, qty decimal.Decimal, isReduceOnly bool) (*CreateOrderRes, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	payload := `
		{
		"order_type": "limit",
		"limit_price": "%s",
		"side": "%s",
		"instrument": {
			"underlying": "%s",
			"instrument_type": "perpetual_future",
			"settlement_asset": "USDC",
			"funding_interval_s": 3600
		},
		"qty": "%s",
		"is_auto_resize": false,
		"use_mark_price": false,
		"is_reduce_only": %v
		}
	`
	payload = fmt.Sprintf(payload, limitPrice, side, symbol, qty, isReduceOnly)

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/orders/new/limit", c.client.endpoint)
	res := c.client.client.Post(g.String(url), payload).WithContext(ctx).AddCookies(cookies...).Do()

	var r CreateOrderRes
	if err := c.client.parseRespone(res, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *UserClient) CreateMarketOrder(ctx context.Context, symbol string, side OrderSide, qty, maxSlippage decimal.Decimal, isReduceOnly bool) (*CreateOrderRes, error) {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	quoteRes, err := c.IndicativeQuote(ctx, symbol, qty)
	if err != nil {
		return nil, err
	}

	payload := `{"quote_id":"%s","side":"%s","max_slippage":%v,"is_reduce_only":%v}`
	payload = fmt.Sprintf(payload, quoteRes.QuoteID, side, maxSlippage.InexactFloat64(), isReduceOnly)

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/orders/new/market", c.client.endpoint)
	res := c.client.client.Post(g.String(url), payload).WithContext(ctx).AddCookies(cookies...).Do()

	var r CreateOrderRes
	if err := c.client.parseRespone(res, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *UserClient) CancelOrder(ctx context.Context, rfqId string) error {
	token, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return err
	}

	cookies := []*httpx.Cookie{
		{Name: "vr-token", Value: token},
	}
	url := fmt.Sprintf("%s/orders/cancel", c.client.endpoint)
	payload := fmt.Sprintf(`{"rfq_id":"%s"}`, rfqId)
	res := c.client.client.Post(g.String(url), payload).WithContext(ctx).AddCookies(cookies...).Do()

	if err := c.client.parseRespone(res, nil); err != nil {
		return err
	}
	return nil
}
