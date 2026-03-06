package hyperliquid

import (
	"github.com/shopspring/decimal"
)

const (
	MainnetBaseURL      = "https://api.hyperliquid.xyz"
	MainnetWebSocketURL = "wss://api.hyperliquid.xyz/ws"
	TestnetBaseURL      = "https://api.hyperliquid-testnet.xyz"
	TestnetWebSocketURL = "wss://api.hyperliquid-testnet.xyz/ws"
	MainnetChainID      = 1
	TestnetChainID      = 421613
)

type OrderResponse struct {
	Status   string      `json:"status"`
	Response interface{} `json:"response,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type Order struct {
	Coin      string          `json:"coin"`
	LimitPx   decimal.Decimal `json:"limitPx"`
	Oid       int64           `json:"oid"`
	Side      string          `json:"side"`
	Sz        decimal.Decimal `json:"sz"`
	Timestamp int64           `json:"timestamp"`
	Status    string          `json:"status"`
}

type Position struct {
	Coin           string          `json:"coin"`
	EntryPx        decimal.Decimal `json:"entryPx"`
	Leverage       Leverage        `json:"leverage"`
	LiquidationPx  decimal.Decimal `json:"liquidationPx"`
	MarginUsed     decimal.Decimal `json:"marginUsed"`
	PositionValue  decimal.Decimal `json:"positionValue"`
	ReturnOnEquity decimal.Decimal `json:"returnOnEquity"`
	Szi            decimal.Decimal `json:"szi"`
	UnrealizedPnl  decimal.Decimal `json:"unrealizedPnl"`
}

type Leverage struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type AccountState struct {
	AssetPositions     []AssetPosition `json:"assetPositions"`
	CrossMarginSummary MarginSummary   `json:"crossMarginSummary"`
	Withdrawable       decimal.Decimal `json:"withdrawable"`
}

type AssetPosition struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
}

type MarginSummary struct {
	AccountValue    decimal.Decimal `json:"accountValue"`
	TotalMarginUsed decimal.Decimal `json:"totalMarginUsed"`
	TotalNtlPos     decimal.Decimal `json:"totalNtlPos"`
	TotalRawUsd     decimal.Decimal `json:"totalRawUsd"`
}

type Meta struct {
	Universe []struct {
		Name         string `json:"name"`
		SzDecimals   int    `json:"szDecimals"`
		MaxLeverage  int    `json:"maxLeverage"`
		OnlyIsolated bool   `json:"onlyIsolated"`
	} `json:"universe"`
}

type AssetContext struct {
	DayNtlVlm    decimal.Decimal   `json:"dayNtlVlm"`
	Funding      decimal.Decimal   `json:"funding"`
	ImpactPxs    []decimal.Decimal `json:"impactPxs"`
	MarkPx       decimal.Decimal   `json:"markPx"`
	MidPx        decimal.Decimal   `json:"midPx"`
	OraclePx     decimal.Decimal   `json:"oraclePx"`
	Premium      decimal.Decimal   `json:"premium"`
	PrevDayPx    decimal.Decimal   `json:"prevDayPx"`
	OpenInterest decimal.Decimal   `json:"openInterest"`
}

type MetaAndAssetCtxs struct {
	Meta      Meta
	AssetCtxs []AssetContext
}

type OrderRequest struct {
	Coin       string    `json:"coin"`
	IsBuy      bool      `json:"is_buy"`
	Sz         string    `json:"sz"`
	LimitPx    string    `json:"limit_px"`
	OrderType  OrderType `json:"order_type"`
	ReduceOnly bool      `json:"reduce_only"`
	Cloid      string    `json:"cloid,omitempty"`
}

type OrderType struct {
	Limit   *LimitOrderType   `json:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty"`
}

type LimitOrderType struct {
	Tif string `json:"tif"`
}

type TriggerOrderType struct {
	TriggerPx decimal.Decimal `json:"triggerPx"`
	IsMarket  bool            `json:"isMarket"`
	Tpsl      string          `json:"tpsl"`
}

type CancelRequest struct {
	Coin string `json:"coin"`
	Oid  int64  `json:"oid"`
}

type WsMessage struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

type OrderUpdate struct {
	Order  Order  `json:"order"`
	Status string `json:"status"`
}

type Fill struct {
	Coin          string          `json:"coin"`
	Side          string          `json:"side"`
	Sz            decimal.Decimal `json:"sz"`
	Px            decimal.Decimal `json:"px"`
	Time          int64           `json:"time"`
	Oid           int64           `json:"oid"`
	Hash          string          `json:"hash"`
	ClosedPnl     decimal.Decimal `json:"closedPnl"`
	Crossed       bool            `json:"crossed"`
	Dir           string          `json:"dir"`
	StartPosition decimal.Decimal `json:"startPosition"`
}

type UserEvent struct {
	Order    *Order    `json:"order,omitempty"`
	OrderOID int64     `json:"orderOid,omitempty"`
	Transfer *Transfer `json:"transfer,omitempty"`
	Nonce    int64     `json:"nonce"`
}

type Transfer struct {
	Delta string `json:"delta"`
	From  string `json:"from"`
	To    string `json:"to"`
	Token string `json:"token"`
}

type Asset struct {
	Asset      int    `json:"asset"`
	IsBuy      bool   `json:"isBuy"`
	LimitPx    string `json:"limitPx"`
	Sz         string `json:"sz"`
	ReduceOnly bool   `json:"r"`
}

type Action struct {
	Type     string         `json:"type"`
	Orders   []Asset        `json:"orders,omitempty"`
	Grouping string         `json:"grouping,omitempty"`
	Cancels  []CancelAction `json:"cancels,omitempty"`
}

type CancelAction struct {
	Asset int   `json:"a"`
	Oid   int64 `json:"o"`
}

type UpdateLeverageAction struct {
	Type     string `json:"type"`
	Asset    int    `json:"asset"`
	IsCross  bool   `json:"isCross"`
	Leverage int    `json:"leverage"`
}
