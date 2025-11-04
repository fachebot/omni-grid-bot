package lighter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	lighterclient "github.com/elliottech/lighter-go/client"
	"github.com/shopspring/decimal"
)

const (
	MainnetBaseURL = "https://mainnet.zklighter.elliot.ai"
	MainnetChainId = uint32(304)
)

type Client struct {
	chainId  uint32
	endpoint string

	httpClient *http.Client
}

func NewClient() *Client {
	c := Client{
		chainId:    MainnetChainId,
		endpoint:   MainnetBaseURL,
		httpClient: new(http.Client),
	}
	return &c
}

func (c *Client) GetNextNonce(ctx context.Context, accountIndex int64, apiKeyIndex uint8) (int64, error) {
	result := &lighterclient.NextNonce{}
	err := getAndParseL2HTTPResponse(
		ctx,
		c.httpClient,
		c.endpoint,
		"api/v1/nextNonce",
		map[string]any{"account_index": accountIndex, "api_key_index": apiKeyIndex},
		result,
	)
	if err != nil {
		return -1, err
	}
	if result.Code != 200 {
		return 0, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}
	return result.Nonce, nil
}

func (c *Client) GetApiKey(ctx context.Context, accountIndex int64, apiKeyIndex uint8) (*lighterclient.AccountApiKeys, error) {
	result := &lighterclient.AccountApiKeys{}
	err := getAndParseL2HTTPResponse(
		ctx,
		c.httpClient,
		c.endpoint,
		"api/v1/apikeys",
		map[string]any{"account_index": accountIndex, "api_key_index": apiKeyIndex},
		result,
	)
	if err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}
	return result, nil
}

func (c *Client) GetTxByHash(ctx context.Context, hash string) (*Transaction, error) {
	var result Transaction
	err := getAndParseL2HTTPResponse(
		ctx,
		c.httpClient,
		c.endpoint,
		"api/v1/tx",
		map[string]any{"by": "hash", "value": hash},
		&result,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetAccountByIndex(ctx context.Context, accountIndex int64) (Accounts, error) {
	var result Accounts
	err := getAndParseL2HTTPResponse(
		ctx,
		c.httpClient,
		c.endpoint,
		"api/v1/account",
		map[string]any{"by": "index", "value": accountIndex},
		&result,
	)
	if err != nil {
		return Accounts{}, err
	}
	if result.Code != 200 {
		return Accounts{}, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}
	return result, nil
}

func (c *Client) GetLastTradePrice(ctx context.Context, marketId uint) (decimal.Decimal, error) {
	var result OrderBookDetails
	err := getAndParseL2HTTPResponse(
		ctx,
		c.httpClient,
		c.endpoint,
		"api/v1/orderBookDetails",
		map[string]any{"market_id": marketId},
		&result,
	)
	if err != nil {
		return decimal.Zero, err
	}
	if result.Code != 200 {
		return decimal.Zero, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}

	if len(result.OrderBookDetails) == 0 {
		return decimal.Zero, nil
	}
	return result.OrderBookDetails[0].LastTradePrice, nil
}

func (c *Client) GetOrderBooksMetadata(ctx context.Context, marketId ...uint) (OrderBooksMetadata, error) {
	params := map[string]any{}
	if len(marketId) > 0 {
		params["market_id"] = marketId[0]
	}

	var result OrderBooksMetadata
	err := getAndParseL2HTTPResponse(
		ctx,
		c.httpClient,
		c.endpoint,
		"api/v1/orderBooks",
		params,
		&result,
	)
	if err != nil {
		return OrderBooksMetadata{}, err
	}
	if result.Code != 200 {
		return OrderBooksMetadata{}, fmt.Errorf("code: %d, message: %s", result.Code, result.Message)
	}

	return result, nil
}

func (c *Client) SendRawTx(ctx context.Context, txType TX_TYPE, txInfo string) (string, error) {
	data := url.Values{"tx_type": {strconv.Itoa(int(txType))}, "tx_info": {txInfo}}
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/v1/sendTx", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}

	if err = parseResultStatus(body); err != nil {
		return "", err
	}

	res := &lighterclient.TxHash{}
	if err := json.Unmarshal(body, res); err != nil {
		return "", err
	}

	return res.TxHash, nil
}

func (c *Client) SendRawTxBatch(ctx context.Context, txTypes []TX_TYPE, txInfos []string) ([]string, error) {
	if len(txTypes) != len(txInfos) {
		return nil, errors.New("transaction types and info count mismatch")
	}

	txTypesData, err := json.Marshal(txTypes)
	if err != nil {
		return nil, err
	}

	txInfosData, err := json.Marshal(txInfos)
	if err != nil {
		return nil, err
	}

	data := url.Values{"tx_types": {string(txTypesData)}, "tx_infos": {string(txInfosData)}}
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/v1/sendTxBatch", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}

	if err = parseResultStatus(body); err != nil {
		return nil, err
	}

	res := &RespSendTxBatch{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, err
	}

	return res.TxHash, nil
}
