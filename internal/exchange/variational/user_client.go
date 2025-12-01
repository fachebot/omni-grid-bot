package variational

import (
	"context"
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
