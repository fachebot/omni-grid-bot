# Hyperliquid API 数据适配分析

## 1. 概述

本文档分析 Hyperliquid API 返回的数据格式与当前系统通用类型的差异，并列出需要实现的转换逻辑。

---

## 2. 数据转换对照表

### 2.1 订单数据 (Order)

| 字段 | 当前系统 (exchange.Order) | Hyperliquid API | 转换说明 |
|------|-------------------------|-----------------|----------|
| Symbol | `BTC` (统一格式) | `BTC` | 无需转换，内部已统一 |
| OrderID | `string` | `oid: int64` | 转为字符串 |
| ClientOrderID | `string` | `cloid: string` (可选) | 客户端订单ID |
| Side | `order.Side` (buy/sell) | `side: "A"/"B"` | A=卖, B=买 → sell/buy |
| Price | `decimal.Decimal` | `limitPx: float64` | 转为 Decimal，8位小数 |
| BaseAmount | `decimal.Decimal` | `sz: string` | 转为 Decimal，8位小数 |
| FilledBaseAmount | `decimal.Decimal` | 需要计算 | sz - remaining |
| FilledQuoteAmount | `decimal.Decimal` | 需要计算 | filled * price |
| Timestamp | `int64` (毫秒) | `timestamp: int64` | 已是毫秒，保持不变 |
| Status | `order.Status` | 字符串 | 需要状态映射 |

**订单状态映射**:

```go
// Hyperliquid 订单状态 → 当前系统
// Hyperliquid: "open", "filled", "canceled", "partiallyFilled"
// 当前系统: in-progress, pending, open, filled, canceled

Hyperliquid "open"      → StatusOpen
Hyperliquid "filled"    → StatusFilled  
Hyperliquid "canceled"  → StatusCanceled
Hyperliquid "partiallyFilled" → StatusOpen (部分成交)
```

**订单方向映射**:

```go
// Hyperliquid: "A" = Ask (卖), "B" = Bid (买)
// 当前系统: SideBuy, SideSell

Hyperliquid "A" → SideSell
Hyperliquid "B" → SideBuy
```

### 2.2 持仓数据 (Position)

| 字段 | 当前系统 (exchange.Position) | Hyperliquid API | 转换说明 |
|------|---------------------------|-----------------|----------|
| Symbol | `BTC` (统一格式) | `coin: "BTC"` | 无需转换 |
| Side | `PositionSide` (1/-1) | `szi` 符号 | 正=1(多头), 负=-1(空头) |
| Position | `decimal.Decimal` | `szi: string` | 转为 Decimal |
| AvgEntryPrice | `decimal.Decimal` | `entryPx: string` | 转为 Decimal |
| UnrealizedPnl | `decimal.Decimal` | `unrealizedPnl: string` | 转为 Decimal |
| RealizedPnl | `decimal.Decimal` | 无 | 需从成交记录计算 |
| LiquidationPrice | `decimal.Decimal` | `liquidationPx: string` | 转为 Decimal |
| MarginMode | `MarginMode` | `leverage.type` | cross/isolated |
| TotalFundingPaidOut | `decimal.Decimal` | 无 | 暂不支持 |

**持仓方向转换**:

```go
// szi: 正数=多头, 负数=空头
szi := decimal.NewFromString(position.Szi)
if szi.IsPositive() {
    side = PositionSideLong   // 1
} else if szi.IsNegative() {
    side = PositionSideShort // -1
} else {
    // 无持仓
}
```

### 2.3 账户数据 (Account)

| 字段 | 当前系统 (exchange.Account) | Hyperliquid API | 转换说明 |
|------|---------------------------|-----------------|----------|
| AvailableBalance | `decimal.Decimal` | `withdrawable: string` | 转为 Decimal |
| Positions | `[]*Position` | `assetPositions[]` | 需要遍历转换 |
| TotalAssetValue | `decimal.Decimal` | `crossMarginSummary.accountValue` | 转为 Decimal |

### 2.4 市场数据 (MarketStats)

| 字段 | 当前系统 (exchange.MarketStats) | Hyperliquid API | 转换说明 |
|------|-------------------------------|-----------------|----------|
| Symbol | `BTC` (统一格式) | `coin: "BTC"` | 无需转换 |
| Price | `decimal.Decimal` | 需指定交易对 | 从 allMids 获取 |
| MarkPrice | `decimal.Decimal` | `markPx: float64` | 转为 Decimal |

---

## 3. 需要创建的转换函数

### 3.1 utils.go 新增函数

```go
package hyperliquid

import (
    "strings"

    "github.com/fachebot/omni-grid-bot/internal/ent/order"
    "github.com/fachebot/omni-grid-bot/internal/exchange"
    "github.com/shopspring/decimal"
)

// ConvertOrderStatus 转换订单状态
func ConvertOrderStatus(status string) order.Status {
    switch status {
    case "open":
        return order.StatusOpen
    case "filled":
        return order.StatusFilled
    case "canceled":
        return order.StatusCanceled
    case "partiallyFilled":
        return order.StatusOpen // 部分成交视为open
    default:
        return order.StatusOpen
    }
}

// ConvertOrderSide 转换订单方向
// Hyperliquid: "A"=Ask(卖), "B"=Bid(买)
// 当前系统: SideBuy, SideSell
func ConvertOrderSide(side string) order.Side {
    if side == "B" {
        return order.SideBuy
    }
    return order.SideSell
}

// ConvertPositionSide 转换持仓方向
// szi: 正数=多头, 负数=空头
func ConvertPositionSide(szi decimal.Decimal) exchange.PositionSide {
    if szi.IsPositive() {
        return exchange.PositionSideLong   // 1
    } else if szi.IsNegative() {
        return exchange.PositionSideShort // -1
    }
    return 0 // 无持仓
}

// ConvertMarginMode 转换保证金模式
func ConvertMarginMode(leverageType string) exchange.MarginMode {
    if leverageType == "cross" {
        return exchange.MarginModeCross
    }
    return exchange.MarginModeIsolated
}

// ParseDecimal 解析字符串为 Decimal
func ParseDecimal(s string) decimal.Decimal {
    if s == "" || s == "0" {
        return decimal.Zero
    }
    d, _ := decimal.NewFromString(s)
    return d
}
```

### 3.2 订单转换函数

```go
// ToExchangeOrder 转换为通用订单
// 注意: 内部系统使用统一符号格式 (如 BTC)，无需额外转换
func (o *Order) ToExchangeOrder() *exchange.Order {
    return &exchange.Order{
        Symbol:            o.Coin,  // 统一格式: BTC
        OrderID:          strconv.FormatInt(o.Oid, 10),
        Side:              ConvertOrderSide(o.Side),
        Price:             ParseDecimal(o.LimitPx.String()),
        BaseAmount:        ParseDecimal(o.Sz.String()),
        FilledBaseAmount:  decimal.Zero, // 需要从订单状态计算
        FilledQuoteAmount: decimal.Zero,
        Timestamp:         o.Timestamp,
        Status:            ConvertOrderStatus(o.Status),
    }
}
```

### 3.3 持仓转换函数

```go
// ToExchangePosition 转换为通用持仓
// 注意: 内部系统使用统一符号格式 (如 BTC)，无需额外转换
func (p *Position) ToExchangePosition() *exchange.Position {
    szi := ParseDecimal(p.Szi)
    
    return &exchange.Position{
        Symbol:           p.Coin,  // 统一格式: BTC
        Side:             ConvertPositionSide(szi),
        Position:         szi.Abs(),
        AvgEntryPrice:    ParseDecimal(p.EntryPx),
        UnrealizedPnl:    ParseDecimal(p.UnrealizedPnl),
        RealizedPnl:      decimal.Zero, // 从成交记录计算
        LiquidationPrice: ParseDecimal(p.LiquidationPx),
        MarginMode:       ConvertMarginMode(p.Leverage.Type),
    }
}
```

### 3.4 账户转换函数

```go
// ToExchangeAccount 转换为通用账户
func (a *AccountState) ToExchangeAccount() *exchange.Account {
    positions := make([]*exchange.Position, 0, len(a.AssetPositions))
    
    for _, ap := range a.AssetPositions {
        if ap.Position.Szi != "0" {
            positions = append(positions, ap.Position.ToExchangePosition())
        }
    }
    
    return &exchange.Account{
        AvailableBalance: ParseDecimal(a.Withdrawable),
        Positions:        positions,
        TotalAssetValue:  ParseDecimal(a.CrossMarginSummary.AccountValue),
    }
}
```

---

## 4. 特殊处理事项

### 4.1 交易对符号

**内部系统**: 使用统一符号格式，如 `BTC`、`ETH`、`SOL`
**Hyperliquid API**: 使用相同格式 `BTC`、`ETH`、`SOL`

系统已统一处理，无需额外转换。各交易所的差异在交易所适配层处理：
- Lighter: 直接使用 `BTC`
- Paradex: 转换为 `BTC-USD-PERP`
- Hyperliquid: 直接使用 `BTC`

### 4.2 价格和数量精度

**Hyperliquid**: 使用字符串表示的浮点数
**当前系统**: 使用 `decimal.Decimal`

```go
// Hyperliquid 价格格式示例: "50000.12345678"
// 需要解析为 8 位小数的 Decimal
```

### 4.3 订单类型

**支持的订单类型**:

| 类型 | 说明 | 当前系统支持 |
|------|------|-------------|
| Limit (GTC) | 限价单 | ✅ |
| Limit (IOC) | 即时成交或取消 | ✅ |
| Limit (Alo) | PostOnly | ✅ |
| Trigger | 止损/止盈单 | 需扩展 |

### 4.4 强平价格

Hyperliquid 的 `liquidationPx` 可能在以下情况为空:
- 无持仓
- 全仓模式且风险较低

```go
if p.LiquidationPx == "" || p.LiquidationPx == "null" {
    liquidationPrice = decimal.Zero
}
```

---

## 5. WebSocket 消息转换

### 5.1 订单更新 (orderUpdates)

```go
// Hyperliquid WebSocket 消息格式
{
    "channel": "orderUpdates",
    "data": [
        {
            "order": {
                "coin": "BTC",
                "limitPx": "50000",
                "oid": 12345,
                "side": "B",
                "sz": "0.1",
                "timestamp": 1234567890
            },
            "status": "filled"
        }
    ]
}
```

**转换**:

```go
func convertOrderUpdate(data map[string]interface{}) *exchange.Order {
    orderData := data["order"].(map[string]interface{})
    return &exchange.Order{
        Symbol:   SymbolToTradingPair(orderData["coin"].(string)),
        OrderID:  strconv.FormatInt(int64(orderData["oid"].(float64)), 10),
        Side:     ConvertOrderSide(orderData["side"].(string)),
        Price:    ParseDecimal(orderData["limitPx"].(string)),
        // ...
        Status:   ConvertOrderStatus(data["status"].(string)),
    }
}
```

### 5.2 成交记录 (userFills)

```go
// Hyperliquid 成交记录格式
{
    "coin": "BTC",
    "side": "B",  // B=买, A=卖
    "sz": "0.1",
    "px": "50000",
    "time": 1234567890,
    "oid": 12345,
    "closedPnl": "0"  // 已实现盈亏
}
```

### 5.3 市场价格 (allMids)

```go
// Hyperliquid allMids 格式
{
    "BTC": "50000.0",
    "ETH": "3000.0",
    "SOL": "100.0"
}

// 转换为 MarketStats
// 注意: 内部系统使用统一符号格式，无需转换
for coin, price := range mids {
    stats := &exchange.MarketStats{
        Symbol: coin,  // 统一格式: BTC
        Price:  ParseDecimal(price),
    }
    // 推送到引擎
}
```

---

## 6. 需要扩展的功能

### 6.1 当前系统不支持的功能

| 功能 | 说明 | 优先级 |
|------|------|--------|
| 已实现盈亏 (RealizedPnl) | 需要从成交记录汇总 | 中 |
| 资金费率历史 | funding history | 低 |
| 订单簿数据 | l2Book 订阅 | 低 |
| 止盈止损触发 | trigger order | 高 |

### 6.2 建议的扩展

1. **支持 Trigger Order**: 在 `OrderType` 中添加 trigger 支持
2. **计算已实现盈亏**: 定期汇总成交记录的 `closedPnl`

---

## 7. 总结

| 类别 | 转换复杂度 | 需要新增函数 |
|------|----------|------------|
| 订单数据 | 中等 | 4个转换函数 |
| 持仓数据 | 简单 | 3个转换函数 |
| 账户数据 | 简单 | 1个转换函数 |
| 市场数据 | 简单 | 2个转换函数 |
| WebSocket | 中等 | 3个消息处理函数 |

**主要工作**:
1. 在 `hyperliquid/utils.go` 中实现转换函数
2. 处理交易对符号转换 (`BTC` → `BTC-USDT`)
3. 状态和方向枚举映射
4. Decimal 精度处理

---

*文档版本: 1.0*
*更新时间: 2025-03-06*
