package nado

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fachebot/omni-grid-bot/internal/util/ethutil"
	"github.com/shopspring/decimal"
)

type UserClient struct {
	client       *Client
	privateKey   *ecdsa.PrivateKey
	subaccountID *big.Int
	sender       Sender
}

type PlaceOrderParams struct {
	ProductId int
	Price     decimal.Decimal
	Amount    decimal.Decimal
	Appendix  *Appendix
}

func NewUserClient(client *Client, hexKey string, subaccountID *big.Int) (*UserClient, error) {
	if len(hexKey) >= 2 &&
		(hexKey[:2] == "0x" || hexKey[:2] == "0X") {
		hexKey = hexKey[2:]
	}
	privateKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, err
	}

	address, err := ethutil.GetAddress(privateKey)
	if err != nil {
		return nil, err
	}

	c := &UserClient{
		client:       client,
		privateKey:   privateKey,
		subaccountID: subaccountID,
		sender:       Sender{Address: address, SubaccountID: subaccountID},
	}
	return c, nil
}

func (c *UserClient) Sender() Sender {
	return c.sender
}

func (c *UserClient) PlaceOrders(ctx context.Context, orders []PlaceOrderParams) (PlaceOrdersRes, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	contracts, err := c.client.GetContracts(ctx)
	if err != nil {
		return nil, err
	}

	var payload PlaceOrdersReq
	for idx, item := range orders {
		nonce := GenerateNonceWithRandom(60*1000, int64(idx))
		signedOrder, err := c.signPlaceOrder(contracts.ChainId.IntPart(), item, nonce)
		if err != nil {
			return nil, err
		}
		payload.PlaceOrders.Orders = append(payload.PlaceOrders.Orders, signedOrder)
	}

	var res PlaceOrdersRes
	err = c.client.doGatewayExecute(ctx, payload, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *UserClient) CancelProductOrders(ctx context.Context, productIds []int) (*CancelProductOrdersRes, error) {
	if len(productIds) == 0 {
		return nil, nil
	}

	contracts, err := c.client.GetContracts(ctx)
	if err != nil {
		return nil, err
	}

	nonce := GenerateNonceWithRandom(60*1000, 0)
	params := SignCancellationProductsParams{
		Sender:     c.sender,
		Nonce:      nonce,
		ProductIds: make([]*big.Int, 0, len(productIds)),
	}
	for _, productId := range productIds {
		params.ProductIds = append(params.ProductIds, big.NewInt(int64(productId)))
	}
	signature, err := SignCancellationProducts(c.privateKey, contracts.ChainId.IntPart(), contracts.EndpointAddr, params)
	if err != nil {
		return nil, err
	}

	payload := CancelProductOrdersReq{
		CancelProductOrders: CancelProductOrders{
			Tx: CancelProductOrdersTx{
				Sender:     c.sender,
				ProductIds: productIds,
				Nonce:      strconv.FormatUint(nonce, 10),
			},
			Signature: signature,
		},
	}

	var res CancelProductOrdersRes
	err = c.client.doGatewayExecute(ctx, payload, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *UserClient) signPlaceOrder(chainId int64, params PlaceOrderParams, nonce uint64) (*PlaceOrderInfo, error) {
	priceX18 := ethutil.FormatUnits(params.Price, 18)
	amountX18 := ethutil.FormatUnits(params.Amount, 18)
	expiration := time.Now().Add(time.Hour * 24 * 30).Unix()

	signParams := SignPlaceOrderParams{
		Sender:     c.sender,
		PriceX18:   priceX18,
		Amount:     amountX18,
		Expiration: uint64(expiration),
		Nonce:      nonce,
		Appendix:   params.Appendix.ToBigInt(),
	}
	verifyingContract := GenOrderVerifyingContract(uint64(params.ProductId))
	signature, err := SignPlaceOrder(c.privateKey, chainId, verifyingContract, signParams)
	if err != nil {
		return nil, err
	}

	payload := &PlaceOrderInfo{
		ProductID: params.ProductId,
		Order: OrderParams{
			Sender:     c.sender.String(),
			PriceX18:   priceX18.String(),
			Amount:     amountX18.String(),
			Expiration: strconv.FormatInt(expiration, 10),
			Nonce:      strconv.FormatUint(nonce, 10),
			Appendix:   params.Appendix.ToBigInt().String(),
		},
		Signature: signature,
	}
	return payload, nil
}
