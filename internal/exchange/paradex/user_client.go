package paradex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/shopspring/decimal"
)

type UserClient struct {
	client        *Client
	dexAccount    string
	dexPrivateKey string
}

func NewUserClient(client *Client, dexAccount, dexPrivateKey string) *UserClient {
	return &UserClient{client: client, dexAccount: dexAccount, dexPrivateKey: dexPrivateKey}
}

func (c *UserClient) DexAccount() string {
	return c.dexAccount
}

func (c *UserClient) EnsureJwtToken(ctx context.Context) (string, error) {
	return c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
}

func (c *UserClient) GetAccount(ctx context.Context) (*AccountInfoRes, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	var errRes *ErrorRes
	var res AccountInfoRes
	err = requests.URL(fmt.Sprintf("%s/account", c.client.endpoint)).Client(c.client.httpClient).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) GetAccountSummaries(ctx context.Context) (AccountSummaries, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	var errRes *ErrorRes
	var res AccountSummaries
	err = requests.URL(fmt.Sprintf("%s/account/summary", c.client.endpoint)).Client(c.client.httpClient).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return res, nil
}

func (c *UserClient) GetPositions(ctx context.Context) (*PositionRes, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	var res PositionRes
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/positions", c.client.endpoint)).Client(c.client.httpClient).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) GetOpenOrders(ctx context.Context) (*OpenOrdersRes, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	var res OpenOrdersRes
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/orders", c.client.endpoint)).Client(c.client.httpClient).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	for _, item := range res.Results {
		if item.AvgFillPrice == "" || !item.Price.IsZero() {
			continue
		}

		avgFillPrice, err := decimal.NewFromString(item.AvgFillPrice)
		if err == nil {
			item.Price = avgFillPrice
		}
	}

	return &res, nil
}

func (c *UserClient) GetUserOrders(ctx context.Context, startAt *time.Time, cursor string, size int) (*OrdersRes, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	params := make(url.Values)
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	params.Set("page_size", strconv.Itoa(size))
	if startAt != nil {
		params.Set("start_at", strconv.FormatInt(startAt.UnixMilli(), 10))
	}

	var res OrdersRes
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/orders-history", c.client.endpoint)).Client(c.client.httpClient).
		Params(params).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	c.processOrderPrices(res.Results)

	return &res, nil
}

func (c *UserClient) GetMarginConfig(ctx context.Context) (*AccounttMarginConfig, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	var errRes *ErrorRes
	var res AccounttMarginConfig
	err = requests.URL(fmt.Sprintf("%s/account/margin", c.client.endpoint)).Client(c.client.httpClient).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) ListFills(ctx context.Context, startAt *time.Time, cursor string, size int) (*FillRes, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	params := make(url.Values)
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	params.Set("page_size", strconv.Itoa(size))
	if startAt != nil {
		params.Set("start_at", strconv.FormatInt(startAt.UnixMilli(), 10))
	}

	var res FillRes
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/fills", c.client.endpoint)).Client(c.client.httpClient).
		Params(params).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) UpsertAccountMargin(ctx context.Context, market string, leverage uint, marginType MarginType) (*MarginConfig, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"leverage":    leverage,
		"margin_type": marginType,
	}

	var res MarginConfig
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/account/margin/%s", c.client.endpoint, market)).Client(c.client.httpClient).Post().
		BodyJSON(payload).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) UpdateAccountMarketMaxSlippage(ctx context.Context, market string, maxSlippageBps int) (*UserProfile, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"max_slippage": decimal.NewFromInt(int64(maxSlippageBps)).Div(decimal.NewFromInt(10000)).String(),
	}

	var res UserProfile
	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/account/profile/market_max_slippage/%s", c.client.endpoint, market)).Client(c.client.httpClient).Post().
		BodyJSON(payload).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) processOrderPrices(orders []*Order) {
	for _, order := range orders {
		if order == nil {
			continue
		}

		if order.AvgFillPrice == "" || !order.Price.IsZero() {
			continue
		}

		avgFillPrice, err := decimal.NewFromString(order.AvgFillPrice)
		if err == nil {
			order.Price = avgFillPrice
		}
	}
}

func (c *UserClient) sendSingleBatch(ctx context.Context, batch []*CreateOrderReq, jwtToken string) (*CreateBatchOrdersRes, error) {
	var errRes *ErrorRes
	var res CreateBatchOrdersRes

	err := requests.
		URL(fmt.Sprintf("%s/orders/batch", c.client.endpoint)).
		Client(c.client.httpClient).
		Post().
		BodyJSON(batch).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		ToJSON(&res).
		Fetch(ctx)

	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	return &res, nil
}

func (c *UserClient) sendBatchOrdersInChunks(ctx context.Context, orders []*CreateOrderReq, jwtToken string) ([]*Order, []*CreateOrderError, error) {
	const maxBatchSize = 10

	var allOrders []*Order
	var allErrors []*CreateOrderError

	// 按批次处理订单
	for i := 0; i < len(orders); i += maxBatchSize {
		// 计算当前批次的结束位置
		end := min(i+maxBatchSize, len(orders))

		// 发送当前批次
		batchOrders, err := c.sendSingleBatch(ctx, orders[i:end], jwtToken)
		if err != nil {
			return nil, nil, err
		}

		for _, err := range batchOrders.Errors {
			if err != nil {
				return nil, nil, err
			}
		}

		allOrders = append(allOrders, batchOrders.Orders...)
		allErrors = append(allErrors, batchOrders.Errors...)
	}

	return allOrders, allErrors, nil
}

func (c *UserClient) CreateBatchOrders(ctx context.Context, batchOrders []*CreateOrderReq) (*CreateBatchOrdersRes, error) {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range batchOrders {
		if item.Signature != "" {
			continue
		}
		err = PopulateOrderSignature(item, *c.client.systemConfig, c.dexAccount, c.dexPrivateKey)
		if err != nil {
			return nil, err
		}
	}

	allOrders, allErrors, err := c.sendBatchOrdersInChunks(ctx, batchOrders, jwtToken)
	if err != nil {
		return nil, err
	}

	c.processOrderPrices(allOrders)

	return &CreateBatchOrdersRes{Orders: allOrders, Errors: allErrors}, nil
}

func (c *UserClient) CancelAllOpenOrders(ctx context.Context, market string) error {
	jwtToken, err := c.EnsureJwtToken(ctx)
	if err != nil {
		return err
	}

	var errRes *ErrorRes
	err = requests.URL(fmt.Sprintf("%s/orders?market=%s", c.client.endpoint, market)).Client(c.client.httpClient).Delete().
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		ErrorJSON(&errRes).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return errRes
		}
		return err
	}

	return nil
}
