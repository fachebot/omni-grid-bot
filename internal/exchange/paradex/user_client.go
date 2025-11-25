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

func (c *UserClient) GetAccount(ctx context.Context) (*AccountInfoRes, error) {
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
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

func (c *UserClient) GetPositions(ctx context.Context) (*PositionRes, error) {
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
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
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
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

	return &res, nil
}

func (c *UserClient) GetUserOrder(ctx context.Context, clientId int64) (*OrdersRes, error) {
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
	if err != nil {
		return nil, err
	}

	params := make(url.Values)
	params.Set("client_id", strconv.FormatInt(clientId, 10))

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

	return &res, nil
}

func (c *UserClient) GetUserOrders(ctx context.Context, startAt *time.Time, cursor string, size int) (*OrdersRes, error) {
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
	if err != nil {
		return nil, err
	}

	params := make(url.Values)
	params.Set("cursor", cursor)
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

	return &res, nil
}

func (c *UserClient) UpsertAccountMargin(ctx context.Context, dexAccount, dexPrivateKey, market string, leverage int, marginType MarginType) (*MarginConfig, error) {
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
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

func (c *UserClient) UpdateAccountMarketMaxSlippage(ctx context.Context, dexAccount, dexPrivateKey, market string, maxSlippageBps int) (*UserProfile, error) {
	jwtToken, err := c.client.EnsureJwtToken(ctx, c.dexAccount, c.dexPrivateKey)
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
