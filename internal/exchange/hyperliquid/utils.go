package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/shopspring/decimal"
)

func ConvertOrderStatus(status string) order.Status {
	switch status {
	case "open":
		return order.StatusOpen
	case "filled":
		return order.StatusFilled
	case "canceled":
		return order.StatusCanceled
	case "partiallyFilled":
		return order.StatusOpen
	default:
		return order.StatusOpen
	}
}

func ConvertOrderSide(side string) order.Side {
	if side == "B" {
		return order.SideBuy
	}
	return order.SideSell
}

func ConvertPositionSide(szi decimal.Decimal) exchange.PositionSide {
	if szi.IsPositive() {
		return exchange.PositionSideLong
	} else if szi.IsNegative() {
		return exchange.PositionSideShort
	}
	return 0
}

func ConvertMarginMode(leverageType string) exchange.MarginMode {
	if leverageType == "cross" {
		return exchange.MarginModeCross
	}
	return exchange.MarginModeIsolated
}

func ParseDecimal(s string) decimal.Decimal {
	if s == "" || s == "0" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func (o *Order) ToExchangeOrder() *exchange.Order {
	return &exchange.Order{
		Symbol:            o.Coin,
		OrderID:           fmt.Sprintf("%d", o.Oid),
		Side:              ConvertOrderSide(o.Side),
		Price:             o.LimitPx,
		BaseAmount:        o.Sz,
		FilledBaseAmount:  decimal.Zero,
		FilledQuoteAmount: decimal.Zero,
		Timestamp:         o.Timestamp,
		Status:            ConvertOrderStatus(o.Status),
	}
}

func (p *Position) ToExchangePosition() *exchange.Position {
	return &exchange.Position{
		Symbol:           p.Coin,
		Side:             ConvertPositionSide(p.Szi),
		Position:         p.Szi.Abs(),
		AvgEntryPrice:    p.EntryPx,
		UnrealizedPnl:    p.UnrealizedPnl,
		RealizedPnl:      decimal.Zero,
		LiquidationPrice: p.LiquidationPx,
		MarginMode:       ConvertMarginMode(p.Leverage.Type),
	}
}

func (a *AccountState) ToExchangeAccount() *exchange.Account {
	positions := make([]*exchange.Position, 0, len(a.AssetPositions))

	for _, ap := range a.AssetPositions {
		if !ap.Position.Szi.IsZero() {
			positions = append(positions, ap.Position.ToExchangePosition())
		}
	}

	return &exchange.Account{
		AvailableBalance: a.Withdrawable,
		Positions:        positions,
		TotalAssetValue:  a.CrossMarginSummary.AccountValue,
	}
}

func CoinToAsset(coin string, meta *Meta) int {
	if meta == nil {
		return 0
	}
	for i, u := range meta.Universe {
		if u.Name == coin {
			return i + 1
		}
	}
	return 0
}

func AssetToCoin(asset int, meta *Meta) string {
	if meta == nil || asset <= 0 || asset > len(meta.Universe) {
		return ""
	}
	return meta.Universe[asset-1].Name
}

func floatToWire(x float64) string {
	return fmt.Sprintf("%.8f", x)
}

func floatToWireWithDecimals(x float64, decimals int) string {
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, x)
}

func getAndParseHTTPResponse(ctx context.Context, httpClient *http.Client, endpoint, path string, params map[string]interface{}, result interface{}) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	u.Path = path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}

func postAndParseHTTPResponse(ctx context.Context, httpClient *http.Client, endpoint, path string, payload interface{}, result interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewReader(jsonData))

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}
