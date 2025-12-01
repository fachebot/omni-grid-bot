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
