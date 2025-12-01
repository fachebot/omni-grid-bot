package variational

import (
	"context"
	"fmt"

	"github.com/enetx/g"
	httpx "github.com/enetx/http"
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

// post https://omni.variational.io/api/settlement_pools/set_leverage
// {"leverage":"36","asset":"BTC"}
// {"current":"36","max":"50"}

// post https://omni.variational.io/api/quotes/indicative
// {"instrument":{"underlying":"BTC","funding_interval_s":3600,"settlement_asset":"USDC","instrument_type":"perpetual_future"},"qty":"0.00011640447528645686"}
// {
//     "instrument": {
//         "instrument_type": "perpetual_future",
//         "underlying": "BTC",
//         "funding_interval_s": 3600,
//         "settlement_asset": "USDC"
//     },
//     "qty": "0.00011640447528645686",
//     "bid": "85927.31",
//     "ask": "85930.66",
//     "mark_price": "85932.0533973301",
//     "index_price": "85968.2715864207",
//     "quote_id": "c4920ea4-acf3-4d53-b0fc-128e19d6e2a7",
//     "margin_requirements": {
//         "existing_margin": {
//             "initial_margin": "2.386849",
//             "maintenance_margin": "1.193424"
//         },
//         "bid_margin_delta": {
//             "initial_margin": "-0.27784",
//             "maintenance_margin": "-0.13892"
//         },
//         "ask_margin_delta": {
//             "initial_margin": "0.277839",
//             "maintenance_margin": "0.13892"
//         },
//         "bid_max_notional_delta": "2023.531538",
//         "ask_max_notional_delta": "1851.668118",
//         "estimated_fees_bid": "0",
//         "estimated_fees_ask": "0",
//         "estimated_liquidation_price_bid": "25357.55201764097",
//         "estimated_liquidation_price_ask": "38240.51497661095"
//     },
//     "margin_params": {
//         "margin_mode": "simple",
//         "params": {
//             "asset_params": {
//                 "BTC": {
//                     "futures_initial_margin": "0.027778",
//                     "futures_maintenance_margin": "0.013889",
//                     "futures_leverage": "100000000000",
//                     "option_initial_margin": "0.15",
//                     "option_initial_margin_min": "0.1",
//                     "option_maintenance_margin": "0.075"
//                 }
//             },
//             "default_asset_param": {
//                 "futures_initial_margin": "0.2",
//                 "futures_maintenance_margin": "0.1",
//                 "futures_leverage": "100000000000",
//                 "option_initial_margin": "0.15",
//                 "option_initial_margin_min": "0.1",
//                 "option_maintenance_margin": "0.075"
//             },
//             "use_default_asset_param": false,
//             "liquidation_penalty": "0.1",
//             "auto_liquidation": true
//         }
//     },
//     "qty_limits": {
//         "bid": {
//             "min_qty_tick": "0.000001",
//             "min_qty": "0.000002",
//             "max_qty": "11637743.576518"
//         },
//         "ask": {
//             "min_qty_tick": "0.000001",
//             "min_qty": "0.000002",
//             "max_qty": "11637289.880003"
//         }
//     }
// }
