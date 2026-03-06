# Hyperliquid 交易所接入详细实施计划

## 1. 概述

本文档详细描述了将 Hyperliquid 交易所接入 omni-grid-bot 项目的实施计划。Hyperliquid 是一个高性能的去中心化永续合约交易所，使用 Ethereum EIP-712 签名进行认证。

### 1.1 技术特点

| 特性 | 说明 |
|------|------|
| 网络 | 主网: `api.hyperliquid.xyz` / 测试网: `api.hyperliquid-testnet.xyz` |
| 签名方式 | Ethereum EIP-712 签名 |
| 认证方式 | 钱包地址 + 私钥签名 |
| 协议 | REST API + WebSocket |

---

## 2. 目录结构设计

```
internal/exchange/hyperliquid/
├── client.go          # HTTP API 客户端
├── subscriber.go      # WebSocket 订阅器
├── signer.go          # 签名工具
├── types.go           # 类型定义
├── utils.go           # 工具函数
└── cache.go           # 数据缓存
```

---

## 3. 模块详细设计

### 3.1 类型定义 (types.go)

```go
package hyperliquid

import "github.com/shopspring/decimal"

// API 响应类型
type OrderResponse struct {
    Status   string `json:"status"`   // "ok" | "err"
    Response any    `json:"response"` // 成功响应数据
    Error    string `json:"error"`    // 错误信息
}

// 订单信息
type Order struct {
    Coin       string          `json:"coin"`        // 交易对 (如 "BTC")
    LimitPx    decimal.Decimal `json:"limitPx"`    // 限价
    Oid        int64           `json:"oid"`         // 订单ID
    Side       string          `json:"side"`        // "A"=Ask(卖), "B"=Bid(买)
    Sz         decimal.Decimal `json:"sz"`          // 数量
    Timestamp  int64           `json:"timestamp"`   // 时间戳
}

// 持仓信息
type Position struct {
    Coin            string          `json:"coin"`
    EntryPx        decimal.Decimal `json:"entryPx"`
    Leverage       Leverage        `json:"leverage"`
    LiquidationPx  decimal.Decimal `json:"liquidationPx"`
    MarginUsed      decimal.Decimal `json:"marginUsed"`
    PositionValue   decimal.Decimal `json:"positionValue"`
    ReturnOnEquity  decimal.Decimal `json:"returnOnEquity"`
    Szi            decimal.Decimal `json:"szi"`           // 正=多头, 负=空头
    UnrealizedPnl   decimal.Decimal `json:"unrealizedPnl"`
}

type Leverage struct {
    Type  string `json:"type"`  // "cross" | "isolated"
    Value int    `json:"value"`
}

// 账户状态
type AccountState struct {
    AssetPositions    []AssetPosition `json:"assetPositions"`
    CrossMarginSummary MarginSummary  `json:"crossMarginSummary"`
    Withdrawable      decimal.Decimal `json:"withdrawable"`
}

type AssetPosition struct {
    Position Position `json:"position"`
}

type MarginSummary struct {
    AccountValue    decimal.Decimal `json:"accountValue"`
    TotalMarginUsed decimal.Decimal `json:"totalMarginUsed"`
    TotalNtlPos     decimal.Decimal `json:"totalNtlPos"`
    TotalRawUsd     decimal.Decimal `json:"totalRawUsd"`
}

// 市场数据
type MarketStats struct {
    Coin      string          `json:"coin"`
    MarkPx    decimal.Decimal `json:"markPx"`
    MidPx     decimal.Decimal `json:"midPx"`
    PrevDayPx decimal.Decimal `json:"prevDayPx"`
}

// 交易对元数据
type Meta struct {
    Universe []struct {
        Name       string `json:"name"`
        SzDecimals int    `json:"szDecimals"`
        MaxLeverage int   `json:"maxLeverage"`
    } `json:"universe"`
}

// 订单请求
type OrderRequest struct {
    Coin       string      `json:"coin"`
    IsBuy      bool        `json:"is_buy"`
    Sz         string      `json:"sz"`
    LimitPx    string      `json:"limit_px"`
    OrderType  OrderType   `json:"order_type"`
    ReduceOnly bool        `json:"reduce_only"`
}

type OrderType struct {
    Limit *LimitOrderType `json:"limit,omitempty"`
    Trigger *TriggerOrderType `json:"trigger,omitempty"`
}

type LimitOrderType struct {
    Tif string `json:"tif"` // "Alo"(PostOnly), "Ioc"(IOC), "Gtc"(GTC)
}

type TriggerOrderType struct {
    TriggerPx decimal.Decimal `json:"triggerPx"`
    IsMarket  bool            `json:"isMarket"`
    Tpsl      string          `json:"tpsl"` // "tp" | "sl"
}

// 取消订单请求
type CancelRequest struct {
    Coin string `json:"coin"`
    Oid  int64  `json:"oid"`
}

// WebSocket 消息
type WsMessage struct {
    Channel string      `json:"channel"`
    Data    interface{} `json:"data"`
}

// WebSocket 订单更新
type OrderUpdate struct {
    Order  Order  `json:"order"`
    Status string `json:"status"` // "open" | "filled" | "canceled"
}

// WebSocket 成交记录
type Fill struct {
    Coin      string          `json:"coin"`
    Side      string          `json:"side"`
    Size      decimal.Decimal `json:"sz"`
    Price     decimal.Decimal `json:"px"`
    Time      int64           `json:"time"`
    Oid       int64           `json:"oid"`
    Hash      string          `json:"hash"`
    ClosedPnl decimal.Decimal `json:"closedPnl"`
}
```

### 3.2 签名模块 (signer.go)

**签名流程**:

```
1. 构建 Action 对象
   action = {
       type: "order",
       orders: [{
           a: asset,      // asset ID
           b: is_buy,    // bool
           p: price,     // string (8位小数)
           s: size,      // string (8位小数)
           r: reduceOnly,
           t: {limit: {tif}}
       }],
       grouping: "na"
   }

2. 使用 msgpack 编码 action
   data = msgpack(action)

3. 添加 nonce (8 bytes)
   data += nonce.to_bytes(8, 'big')

4. 添加 vaultAddress 标记 (1 byte)
   data += b"\x00" (无 vault) 或 b"\x01" + vaultAddress

5. Keccak256 哈希
   hash = keccak256(data)

6. 构建 Phantom Agent
   agent = {source: "a"(mainnet)|"b"(testnet), connectionId: hash}

7. EIP-712 签名 agent
   signature = sign_typed_data(agent)
```

**核心实现**:

```go
package hyperliquid

import (
    "encoding/hex"
    "fmt"
    "math/big"
    "time"
    
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/ethereum/go-ethereum/common/hexutil"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Signer Hyperliquid 签名器
type Signer struct {
    privateKey *crypto.Key   // Ethereum 私钥
    publicKey  string        // 钱包地址 (0x...)
    isMainnet  bool          // 是否主网
    vaultAddr  string        // Vault 地址 (可选)
}

// NewSigner 创建签名器
func NewSigner(privateKeyHex string, isMainnet bool) (*Signer, error) {
    privateKey, err := crypto.HexToKey(privateKeyHex)
    if err != nil {
        return nil, fmt.Errorf("invalid private key: %w", err)
    }
    
    publicKey := crypto.PubkeyToAddress(privateKey.PublicKey)
    
    return &Signer{
        privateKey: privateKey,
        publicKey:  publicKey.Hex(),
        isMainnet:  isMainnet,
    }, nil
}

// SignOrder 签名订单
func (s *Signer) SignOrder(action map[string]interface{}, nonce int64) (string, error) {
    // 1. msgpack 编码 action
    // 2. 添加 nonce
    // 3. 添加 vault 标记
    // 4. Keccak256 哈希
    // 5. EIP-712 签名
    
    // ... 详细实现
    
    return signature, nil
}

// GetAddress 获取钱包地址
func (s *Signer) GetAddress() string {
    return s.publicKey
}
```

### 3.3 HTTP 客户端 (client.go)

```go
package hyperliquid

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "github.com/shopspring/decimal"
)

// Client Hyperliquid HTTP 客户端
type Client struct {
    baseURL   string           // API 地址
    httpClient *http.Client    // HTTP 客户端
    signer    *Signer          // 签名器
}

// NewClient 创建客户端
func NewClient(httpClient *http.Client, signer *Signer, isMainnet bool) *Client {
    url := "https://api.hyperliquid.xyz"
    if !isMainnet {
        url = "https://api.hyperliquid-testnet.xyz"
    }
    
    return &Client{
        baseURL:    url,
        httpClient: httpClient,
        signer:     signer,
    }
}

// UserState 获取用户状态 (持仓)
func (c *Client) UserState(ctx context.Context, address string) (*AccountState, error) {
    payload := map[string]interface{}{
        "type": "clearinghouseState",
        "user": address,
    }
    
    var result AccountState
    err := c.post(ctx, "/info", payload, &result)
    return &result, err
}

// OpenOrders 获取用户挂单
func (c *Client) OpenOrders(ctx context.Context, address string) ([]Order, error) {
    payload := map[string]interface{}{
        "type": "openOrders",
        "user": address,
    }
    
    var result []Order
    err := c.post(ctx, "/info", payload, &result)
    return result, err
}

// UserFills 获取用户成交记录
func (c *Client) UserFills(ctx context.Context, address string) ([]Fill, error) {
    payload := map[string]interface{}{
        "type": "userFills",
        "user": address,
    }
    
    var result []Fill
    err := c.post(ctx, "/info", payload, &result)
    return result, err
}

// Meta 获取交易对元数据
func (c *Client) Meta(ctx context.Context) (*Meta, error) {
    payload := map[string]interface{}{
        "type": "meta",
    }
    
    var result Meta
    err := c.post(ctx, "/info", payload, &result)
    return &result, err
}

// AllMids 获取所有交易对中间价
func (c *Client) AllMids(ctx context.Context) (map[string]decimal.Decimal, error) {
    payload := map[string]interface{}{
        "type": "allMids",
    }
    
    var result map[string]float64
    err := c.post(ctx, "/info", payload, &result)
    
    // 转换为 decimal
    mids := make(map[string]decimal.Decimal)
    for k, v := range result {
        mids[k] = decimal.NewFromFloat(v)
    }
    return mids, err
}

// PlaceOrder 下单
func (c *Client) PlaceOrder(ctx context.Context, order *OrderRequest) (*OrderResponse, error) {
    // 转换 order 为 wire 格式
    orderWire := orderToWire(order)
    
    // 构建 action
    action := map[string]interface{}{
        "type":   "order",
        "orders": []interface{}{orderWire},
    }
    
    // 签名
    nonce := time.Now().UnixMilli()
    signature, err := c.signer.SignOrder(action, nonce)
    if err != nil {
        return nil, err
    }
    
    // 构建请求
    reqPayload := map[string]interface{}{
        "action":    action,
        "nonce":     nonce,
        "signature": signature,
    }
    
    var result OrderResponse
    err = c.post(ctx, "/exchange", reqPayload, &result)
    return &result, err
}

// CancelOrder 取消订单
func (c *Client) CancelOrder(ctx context.Context, coin string, oid int64) error {
    action := map[string]interface{}{
        "type": "cancel",
        "cancels": []map[string]interface{}{
            {"a": coinToAsset(coin), "o": oid},
        },
    }
    
    nonce := time.Now().UnixMilli()
    signature, err := c.signer.SignCancel(action, nonce)
    if err != nil {
        return err
    }
    
    reqPayload := map[string]interface{}{
        "action":    action,
        "nonce":     nonce,
        "signature": signature,
    }
    
    var result OrderResponse
    return c.post(ctx, "/exchange", reqPayload, &result)
}

// UpdateLeverage 更新杠杆
func (c *Client) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error {
    action := map[string]interface{}{
        "type":     "updateLeverage",
        "asset":    asset,
        "isCross":  isCross,
        "leverage": leverage,
    }
    
    nonce := time.Now().UnixMilli()
    signature, err := c.signer.SignAction(action, nonce)
    if err != nil {
        return err
    }
    
    reqPayload := map[string]interface{}{
        "action":    action,
        "nonce":     nonce,
        "signature": signature,
    }
    
    var result OrderResponse
    return c.post(ctx, "/exchange", reqPayload, &result)
}

// post 发送 POST 请求
func (c *Client) post(ctx context.Context, path string, payload interface{}, result interface{}) error {
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
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
    
    return json.Unmarshal(body, result)
}
```

### 3.4 WebSocket 订阅器 (subscriber.go)

```go
package hyperliquid

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/fachebot/omni-grid-bot/internal/config"
    "github.com/fachebot/omni-grid-bot/internal/exchange"
    "github.com/fachebot/omni-grid-bot/internal/logger"
)

// Subscriber WebSocket 订阅器
type Subscriber struct {
    ctx          context.Context
    cancel       context.CancelFunc
    stopChan     chan struct{}
    
    url          string
    conn         *websocket.Conn
    proxy        config.Sock5Proxy
    
    mutex        sync.Mutex
    accounts     map[string]*Signer  // address -> signer
    
    subMsgChan   chan exchange.SubMessage
}

// NewSubscriber 创建订阅器
func NewSubscriber(signer *Signer, proxy config.Sock5Proxy, isMainnet bool) *Subscriber {
    ctx, cancel := context.WithCancel(context.Background())
    
    url := "wss://api.hyperliquid.xyz/ws"
    if !isMainnet {
        url = "wss://api.hyperliquid-testnet.xyz/ws"
    }
    
    return &Subscriber{
        ctx:      ctx,
        cancel:   cancel,
        url:      url,
        proxy:    proxy,
        accounts: make(map[string]*Signer),
    }
}

// Start 启动订阅器
func (s *Subscriber) Start() {
    if s.stopChan != nil {
        return
    }
    
    s.stopChan = make(chan struct{}, 1)
    go s.run()
}

// Stop 停止订阅器
func (s *Subscriber) Stop() {
    s.cancel()
    if s.conn != nil {
        s.conn.Close()
    }
    <-s.stopChan
}

// SubscriptionChan 获取消息通道
func (s *Subscriber) SubscriptionChan() <-chan exchange.SubMessage {
    if s.subMsgChan == nil {
        s.subMsgChan = make(chan exchange.SubMessage, 1024)
    }
    return s.subMsgChan
}

// SubscribeUser 订阅用户数据
func (s *Subscriber) SubscribeUser(signer *Signer) error {
    address := signer.GetAddress()
    
    s.mutex.Lock()
    s.accounts[address] = signer
    s.mutex.Unlock()
    
    // 订阅订单更新
    subscribeMsg := map[string]interface{}{
        "method": "subscribe",
        "subscription": map[string]interface{}{
            "type": "orderUpdates",
            "user": address,
        },
    }
    
    return s.conn.WriteJSON(subscribeMsg)
}

// SubscribeMarket 订阅市场数据
func (s *Subscriber) SubscribeMarket(coin string) error {
    subscribeMsg := map[string]interface{}{
        "method": "subscribe",
        "subscription": map[string]interface{}{
            "type": "allMids",
        },
    }
    
    return s.conn.WriteJSON(subscribeMsg)
}

// run 运行循环
func (s *Subscriber) run() {
    s.connect()
    
    for {
        select {
        case <-s.ctx.Done():
            s.stopChan <- struct{}{}
            return
        case <-time.After(30 * time.Second):
            s.sendPing()
        default:
            _, message, err := s.conn.ReadMessage()
            if err != nil {
                logger.Errorf("[HyperliquidSubscriber] 读取消息失败: %v", err)
                s.reconnect()
                continue
            }
            
            s.handleMessage(message)
        }
    }
}

// handleMessage 处理消息
func (s *Subscriber) handleMessage(data []byte) {
    var msg WsMessage
    if err := json.Unmarshal(data, &msg); err != nil {
        logger.Errorf("[HyperliquidSubscriber] 解析消息失败: %v", err)
        return
    }
    
    switch msg.Channel {
    case "orderUpdates":
        s.handleOrderUpdates(msg.Data)
    case "userFills":
        s.handleUserFills(msg.Data)
    case "userEvents":
        s.handleUserEvents(msg.Data)
    case "allMids":
        s.handleAllMids(msg.Data)
    case "pong":
        // 心跳响应
    }
}

// handleOrderUpdates 处理订单更新
func (s *Subscriber) handleOrderUpdates(data interface{}) {
    // 解析并转换为通用格式
    // 推送到 subMsgChan
}

// handleAllMids 处理市场价格
func (s *Subscriber) handleAllMids(data interface{}) {
    // 解析所有交易对价格
    // 推送到 subMsgChan
}
```

---

## 4. 与系统集成

### 4.1 修改 enum.go

```go
// internal/exchange/enum.go
const (
    Lighter     string = "lighter"
    Paradex     string = "paradex"
    Variational string = "variational"
    Hyperliquid string = "hyperliquid"  // 新增
)
```

### 4.2 修改 ServiceContext (svc/service_context.go)

```go
// 新增客户端字段
type ServiceContext struct {
    // ... 现有字段
    
    HyperliquidClient *hyperliquid.Client  // 新增
}

// 在 NewServiceContext 中初始化
// 读取配置中的 Hyperliquid 私钥
hyperliquidSigner, err := hyperliquid.NewSigner(config.Hyperliquid.PrivateKey, config.Hyperliquid.IsMainnet)
if err != nil {
    logger.Fatalf("创建 Hyperliquid 签名器失败: %v", err)
}
svcCtx.HyperliquidClient = hyperliquid.NewClient(httpClient, hyperliquidSigner, config.Hyperliquid.IsMainnet)
```

### 4.3 修改 main.go

```go
// 新增订阅器
hyperliquidSubscriber := hyperliquid.NewSubscriber(
    svcCtx.HyperliquidClient.Signer(),
    c.Sock5Proxy,
    c.Hyperliquid.IsMainnet,
)
hyperliquidSubscriber.Start()

// 传递给引擎
strategyEngine := engine.NewStrategyEngine(
    svcCtx, 
    lighterSubscriber, 
    paradexSubscriber, 
    variationalSubscriber,
    hyperliquidSubscriber,  // 新增
)
```

### 4.4 修改配置文件 (etc/config.yaml)

```yaml
Hyperliquid:
  PrivateKey: "your-private-key"  # 不提交到版本控制
  IsMainnet: true
```

---

## 5. 实施步骤

### 阶段一: 基础架构 (预计 1-2 天)

| 步骤 | 任务 | 文件 |
|------|------|------|
| 1.1 | 创建目录结构 | `internal/exchange/hyperliquid/` |
| 1.2 | 定义基础类型 | `types.go` |
| 1.3 | 实现签名模块 | `signer.go` |
| 1.4 | 实现 HTTP 客户端 | `client.go` |

### 阶段二: WebSocket 订阅 (预计 1-2 天)

| 步骤 | 任务 | 文件 |
|------|------|------|
| 2.1 | 实现 WebSocket 订阅器 | `subscriber.go` |
| 2.2 | 实现消息处理 | `subscriber.go` |
| 2.3 | 实现缓存模块 | `cache.go` |
| 2.4 | 测试 WebSocket 连接 | - |

### 阶段三: 系统集成 (预计 1 天)

| 步骤 | 任务 | 文件 |
|------|------|------|
| 3.1 | 添加交易所枚举 | `enum.go` |
| 3.2 | 更新 ServiceContext | `svc/service_context.go` |
| 3.3 | 更新主程序 | `main.go` |
| 3.4 | 添加配置项 | `config.go`, `config.yaml.sample` |

### 阶段四: 测试与调试 (预计 1-2 天)

| 步骤 | 任务 |
|------|------|
| 4.1 | 单元测试 |
| 4.2 | 集成测试 (测试网) |
| 4.3 | 订单功能测试 |
| 4.4 | 网格策略测试 |

---

## 6. API 对应关系

### 6.1 Info API

| 功能 | Hyperliquid API | 对应方法 |
|------|-----------------|----------|
| 用户持仓 | `POST /info {type: "clearinghouseState"}` | `UserState()` |
| 用户挂单 | `POST /info {type: "openOrders"}` | `OpenOrders()` |
| 成交历史 | `POST /info {type: "userFills"}` | `UserFills()` |
| 交易对元数据 | `POST /info {type: "meta"}` | `Meta()` |
| 中间价 | `POST /info {type: "allMids"}` | `AllMids()` |
| 订单簿 | `POST /info {type: "l2Book"}` | `L2Book()` |

### 6.2 Exchange API

| 功能 | Hyperliquid API | 对应方法 |
|------|-----------------|----------|
| 下单 | `POST /exchange` | `PlaceOrder()` |
| 取消订单 | `POST /exchange` | `CancelOrder()` |
| 修改订单 | `POST /exchange` | `ModifyOrder()` |
| 更新杠杆 | `POST /exchange` | `UpdateLeverage()` |

### 6.3 WebSocket 订阅

| 订阅类型 | Channel | 用途 |
|----------|---------|------|
| 订单更新 | `orderUpdates` | 用户订单状态变化 |
| 成交 | `userFills` | 用户成交推送 |
| 用户事件 | `userEvents` | 仓位/余额变化 |
| 市场价格 | `allMids` | 所有交易对中间价 |
| 订单簿 | `l2Book:{coin}` | 指定交易对订单簿 |

---

## 7. 注意事项

### 7.1 签名注意事项

1. **私钥格式**: 需要提供 Ethereum 私钥 (64字符十六进制，不带 0x 前缀)
2. **主网 vs 测试网**: 
   - 主网 chainId = 1
   - 测试网 chainId = 421613 (Arbitrum Goerli)
3. **签名类型**: 使用 EIP-712 标准的 Ethereum 签名

### 7.2 订单格式转换

| 字段 | 说明 | 示例 |
|------|------|------|
| `coin` | 交易对名称 | "BTC", "ETH", "SOL" |
| `a` | Asset ID (从 meta 获取) | BTC = 1 |
| `b` | 是否买入 | true / false |
| `p` | 价格 (8位小数字符串) | "50000.00000000" |
| `s` | 数量 (8位小数字符串) | "0.10000000" |
| `r` | 是否只减仓 | true / false |
| `t` | 订单类型 | {limit: {tif: "Gtc"}} |

### 7.3 错误处理

- **rate limit**: 返回 429 时使用指数退避重试
- **签名错误**: 检查私钥格式和签名逻辑
- **订单失败**: 解析响应中的 error 字段

---

## 8. 验收标准

1. ✅ 能够通过私钥创建签名器
2. ✅ 能够查询用户持仓和挂单
3. ✅ 能够下单和取消订单
4. ✅ WebSocket 能够接收订单更新
5. ✅ 网格策略能够正常执行
6. ✅ 配置文件能够正确加载

---

*文档版本: 1.0*  
*更新时间: 2025-03-06*
