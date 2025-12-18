package nado

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/carlmjohnson/requests"
	"github.com/ethereum/go-ethereum/common"
)

const (
	GATEWAY_ENDPOINT = "https://gateway.prod.nado.xyz/v1"
	ARCHIVE_ENDPOINT = "https://archive.prod.nado.xyz/v1"
)

type Client struct {
	httpClient      *http.Client
	gatewayEndpoint string
	archiveEndpoint string

	mutex     sync.Mutex
	contracts *ContractsRes
}

func NewClient(httpClient *http.Client) *Client {
	c := Client{
		httpClient:      httpClient,
		gatewayEndpoint: GATEWAY_ENDPOINT,
		archiveEndpoint: ARCHIVE_ENDPOINT,
	}
	return &c
}

func (c *Client) getContracts(ctx context.Context) (*ContractsRes, error) {
	var v ContractsRes

	params := make(url.Values, 0)
	params.Add("type", "contracts")
	if err := c.doGatewayQuery(ctx, params, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) GetContracts(ctx context.Context) (*ContractsRes, error) {
	c.mutex.Lock()
	if c.contracts != nil {
		v := *c.contracts
		c.mutex.Unlock()
		return &v, nil
	}

	contracts, err := c.getContracts(ctx)
	if err != nil {
		c.mutex.Unlock()
		return nil, err
	}

	c.contracts = contracts
	v := *c.contracts
	c.mutex.Unlock()

	return &v, nil
}

func (c *Client) GetNonces(ctx context.Context, address common.Address) (*NoncesRes, error) {
	var v NoncesRes

	params := make(url.Values, 0)
	params.Add("type", "nonces")
	params.Add("address", address.Hex())
	if err := c.doGatewayQuery(ctx, params, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) GetSubaccountInfo(ctx context.Context, sender Sender) (*SubaccountData, error) {
	var v SubaccountData

	params := make(url.Values, 0)
	params.Add("type", "subaccount_info")
	params.Add("subaccount", sender.String())
	if err := c.doGatewayQuery(ctx, params, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) FindSubaccountsByAddress(ctx context.Context, address common.Address) (*SubaccountsRes, error) {
	var v SubaccountsRes

	payload := map[string]any{
		"subaccounts": map[string]any{
			"address": address.Hex(),
		},
	}
	if err := c.doArchiveQuery(ctx, payload, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) doArchiveQuery(ctx context.Context, payload any, res any) error {
	var errorMessage string
	errorString := requests.ValidatorHandler(
		requests.DefaultValidator,
		requests.ToString(&errorMessage),
	)

	err := requests.URL(c.archiveEndpoint).Client(c.httpClient).Post().
		Header("Content-Type", "application/json").
		BodyJSON(payload).
		AddValidator(errorString).
		ToJSON(res).
		Fetch(ctx)
	if err != nil {
		if errorMessage != "" {
			return fmt.Errorf("nado gateway error: %s", errorMessage)
		}
		return err
	}

	return nil
}

func (c *Client) doGatewayQuery(ctx context.Context, params url.Values, res any) error {
	var result Result

	var errorMessage string
	errorString := requests.ValidatorHandler(
		requests.DefaultValidator,
		requests.ToString(&errorMessage),
	)

	err := requests.URL(fmt.Sprintf("%s/query", c.gatewayEndpoint)).Client(c.httpClient).
		Params(params).
		AddValidator(errorString).
		ToJSON(&result).
		Fetch(ctx)
	if err != nil {
		if errorMessage != "" {
			return fmt.Errorf("nado gateway error: %s", errorMessage)
		}
		return err
	}

	return json.Unmarshal(result.Data, &res)
}

func (c *Client) doGatewayExecute(ctx context.Context, payload any, res any) error {
	var result Result

	var errorMessage string
	errorString := requests.ValidatorHandler(
		requests.DefaultValidator,
		requests.ToString(&errorMessage),
	)

	err := requests.URL(fmt.Sprintf("%s/execute", c.gatewayEndpoint)).Client(c.httpClient).Post().
		Header("Content-Type", "application/json").
		BodyJSON(payload).
		AddValidator(errorString).
		ToJSON(&result).
		Fetch(ctx)
	if err != nil {
		if errorMessage != "" {
			return fmt.Errorf("nado gateway error: %s", errorMessage)
		}
		return err
	}

	if result.Error() != "" {
		return &result
	}

	return json.Unmarshal(result.Data, &res)
}
