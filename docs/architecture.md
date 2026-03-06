# OmniGrid Bot 架构设计文档

## 1. 系统概述

OmniGrid Bot 是一个用 Golang 编写的网格交易 Telegram 机器人框架，用于连接多家 Perp DEX，并可配置策略自动执行网格交易。

### 1.1 核心目标

- 统一多交易所对接层
- 可配置、可扩展的网格策略管理
- 可靠的持久化与审计记录
- 通过 Telegram 进行监控和管理

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           用户层 (Telegram)                              │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  /start │ /create │ /list │ /settings │ /details │ /delete    │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      TeleBot (telebot.go)                               │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  • 消息接收与分发  • 路由管理  • 白名单验证  • Handler处理       │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     ServiceContext (svc)                                │
│  ┌─────────────┬─────────────┬─────────────┬─────────────────────┐    │
│  │   Config    │    Cache    │   Clients   │     Model Layer    │    │
│  │  配置管理    │   多级缓存   │ 交易所API   │     数据访问层      │    │
│  └─────────────┴─────────────┴─────────────┴─────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       Strategy Engine                                   │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  • 策略管理  • 订单处理  • 仓位管理  • 重试队列  • 事件分发      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│   Lighter       │ │  Paradex    │ │Variational  │ │   Market    │
│  WebSocket      │ │  WebSocket  │ │  WebSocket  │ │   Data      │
└─────────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
```

### 2.2 组件关系

```
┌─────────────┐     创建      ┌─────────────┐
│  main.go    │ ──────────▶  │ServiceContext│
└─────────────┘              └──────┬──────┘
                                    │
         ┌──────────┬───────────────┼───────────────┬──────────┐
         ▼          ▼               ▼               ▼          ▼
   ┌─────────┐┌─────────┐   ┌──────────┐  ┌──────────┐ ┌────────┐
   │TeleBot  ││ Engine  │   │ParadexCli│  │LighterCli│ │VariCli │
   └────┬────┘└────┬────┘   └────┬─────┘  └────┬─────┘ └────┬───┘
        │          │              │              │             │
        │          │              │              │             │
        ▼          ▼              ▼              ▼             ▼
   ┌─────────┐┌─────────┐   ┌──────────┐  ┌──────────┐ ┌────────┐
   │ Handlers││Strategy │   │ Subscriber│  │ Subscriber││Subscriber
   └─────────┘└─────────┘   └──────────┘  └──────────┘ └────────┘
```

---

## 3. 核心模块

### 3.1 配置管理 (internal/config)

**职责**: 加载和管理应用配置

**主要结构**:

```go
type Config struct {
    Log          LogConfig        // 日志配置
    Sock5Proxy   Sock5ProxyConfig // SOCKS5代理
    TelegramBot  TelegramBotConfig // Telegram机器人配置
}
```

**功能**:

- 从 `etc/config.yaml` 读取配置
- 支持 SOCKS5 代理配置
- 白名单用户管理

---

### 3.2 服务上下文 (internal/svc)

**职责**: 依赖注入容器，管理全局共享资源

**核心组件**:

```go
type ServiceContext struct {
    Config             *config.Config   // 配置
    Bot                *tele.Bot        // Telegram机器人
    DbClient           *ent.Client      // 数据库客户端
    TransportProxy     *http.Transport  // HTTP代理
    MessageCache       *cache.MessageCache   // 消息缓存
    LighterCache       *cache.LighterCache    // Lighter缓存
    ParadexCache       *cache.ParadexCache    // Paradex缓存
    PendingOrdersCache *cache.PendingOrdersCache
    
    ParadexClient     *paradex.Client  // Paradex API客户端
    LighterClient     *lighter.Client  // Lighter API客户端
    VariationalClient *variational.Client
    
    GridModel         *model.GridModel
    OrderModel        *model.OrderModel
    StrategyModel     *model.StrategyModel
    MatchedTradeModel *model.MatchedTradeModel
}
```

---

### 3.3 策略引擎 (internal/engine)

**职责**: 网格交易策略的执行引擎

**核心接口**:

```go
type Strategy interface {
    Get() *ent.Strategy           // 获取策略实体
    Update(s *ent.Strategy)       // 更新策略
    OnTicker(ctx context.Context, price decimal.Decimal)  // 价格更新回调
    OnOrdersChanged(ctx context.Context) error             // 订单变化回调
}
```

**核心功能**:

| 功能 | 说明 |
|------|------|
| 策略注册 | 启动/停止/更新策略 |
| 订单订阅 | 订阅用户订单状态 |
| 市场订阅 | 订阅市场行情数据 |
| 重试队列 | 失败订单自动重试 |
| 事件分发 | 处理WebSocket消息 |

**重试机制**:

- 使用最小堆实现优先级队列
- 支持策略级别的失败重试
- 可配置重试间隔

---

### 3.4 策略实现 (internal/strategy)

**GridStrategy**: 网格交易策略实现

**核心逻辑**:

```
价格更新 (OnTicker):
    │
    ▼
检查止盈/止损条件
    │
    ├─ 触发止损 → 停止策略 + 平仓
    ├─ 触发止盈 → 停止策略 + 平仓
    │
    ▼
无触发 → 检查网格状态

订单变化 (OnOrdersChanged):
    │
    ▼
加载网格状态 (LoadGridStrategyState)
    │
    ▼
网格再平衡 (Rebalance)
    │
    ├─ 买单成交 → 检查是否需要开空
    ├─ 卖单成交 → 检查是否需要开多
    └─ 全部成交 → 挂单等待
```

**网格参数**:

| 参数 | 说明 |
|------|------|
| GridCount | 网格数量 |
| GridSpacing | 网格间距 (%) |
| PositionSize | 仓位大小 |
| Leverage | 杠杆倍数 |
| Mode | Long/Short |
| TriggerTakeProfitPrice | 止盈价格 |
| TriggerStopLossPrice | 止损价格 |

---

### 3.5 交易所适配层 (internal/exchange)

**统一接口类型**:

```go
// Order 订单信息
type Order struct {
    Symbol            string
    OrderID           string
    ClientOrderID     string
    Side              order.Side
    Price             decimal.Decimal
    BaseAmount        decimal.Decimal
    FilledBaseAmount  decimal.Decimal
    FilledQuoteAmount decimal.Decimal
    Timestamp         int64
    Status            order.Status
}

// Position 持仓信息
type Position struct {
    Symbol              string
    Side                PositionSide    // 1=多头, -1=空头
    Position            decimal.Decimal
    AvgEntryPrice       decimal.Decimal
    UnrealizedPnl       decimal.Decimal
    RealizedPnl         decimal.Decimal
    LiquidationPrice    decimal.Decimal
}

// MarketStats 市场数据
type MarketStats struct {
    Symbol    string
    Price     decimal.Decimal
    MarkPrice decimal.Decimal
}
```

**已支持的交易所**:

| 交易所 | 特点 | 目录 |
|--------|------|------|
| Lighter | 去中心化永续合约 | lighter/ |
| Paradex | Starknet 永续合约 | paradex/ |
| Variational | 前端 API 对接 | variational/ |

**交易所模块结构** (以 Lighter 为例):

```
lighter/
├── client.go       # HTTP API 客户端 (下单/查询/撤单)
├── subscriber.go  # WebSocket 订阅器
├── signer.go       # 签名工具
├── types.go        # 类型定义
├── utils.go        # 工具函数
└── cache.go        # 数据缓存
```

---

### 3.6 Telegram 机器人 (internal/telebot)

**架构**:

```
┌─────────────────────────────────────────────────────────────┐
│  TeleBot (telebot.go)                                       │
│  • 消息接收  • 路由分发  • 白名单验证                         │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  PathRouter (pathrouter/)                                   │
│  • 路径匹配  • 参数解析  • Handler 调用                      │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Handlers (handler/)                                        │
├─────────────────────────────────────────────────────────────┤
│ create_strategy_handler.go    - 创建策略                     │
│ strategy_list_handler.go      - 策略列表                     │
│ strategy_details_handler.go   - 策略详情                     │
│ strategy_settings_handler.go  - 策略设置                     │
│ strategy_switch_handler.go    - 策略启停                     │
│ exchange_settings_handler.go  - 交易所设置                   │
│ matched_trades_handler.go     - 成交记录                     │
│ close_position_handler.go     - 平仓操作                     │
│ delete_strategy_handler.go    - 删除策略                     │
└─────────────────────────────────────────────────────────────┘
```

**命令列表**:

| 命令 | 说明 |
|------|------|
| /start | 显示主菜单 |
| /create | 创建新策略 |
| /list | 查看策略列表 |
| /details | 查看策略详情 |
| /settings | 修改策略设置 |
| /exchange | 交易所配置 |
| /trades | 成交记录 |

---

### 3.7 数据访问层 (internal/model)

**职责**: 封装对数据库的 CRUD 操作

**核心 Model**:

| Model | 职责 |
|-------|------|
| StrategyModel | 策略的增删改查 |
| OrderModel | 订单的增删改查 |
| GridModel | 网格信息的增删改查 |
| MatchedTradeModel | 成交记录的增删改查 |
| SyncProgressModel | 同步进度管理 |

---

### 3.8 缓存层 (internal/cache)

**职责**: 降低数据库和 API 调用频率

**缓存实现**:

| 缓存 | 用途 |
|------|------|
| MessageCache | Telegram 消息路由缓存 |
| LighterCache | Lighter 账户/持仓缓存 |
| ParadexCache | Paradex 账户/持仓缓存 |
| PendingOrdersCache | 待处理订单缓存 |

---

## 4. 数据库设计

### 4.1 实体关系

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Strategy   │────▶│    Order     │◀────│    Grid      │
├──────────────┤     ├──────────────┤     ├──────────────┤
│ GUID (PK)   │     │ OrderID (PK) │     │ ID (PK)      │
│ Owner        │     │ ClientOrderID│     │ StrategyID   │
│ Symbol       │     │ StrategyID   │     │ Level        │
│ Exchange     │     │ Symbol       │     │ Price        │
│ Account      │     │ Side         │     │ Quantity     │
│ Mode         │     │ Price        │     │ Status       │
│ GridCount    │     │ Quantity     │     │ OrderID      │
│ GridSpacing  │     │ Status       │     │ CreatedAt    │
│ PositionSize │     │ CreatedAt    │     └──────────────┘
│ Leverage     │     └──────────────┘
│ Active       │     ┌──────────────┐
│ CreatedAt    │     │ MatchedTrade │
└──────────────┘     ├──────────────┤
                    │ ID (PK)      │
                    │ StrategyID   │
                    │ OrderID      │
                    │ GridLevel    │
                    │ Symbol       │
                    │ Side         │
                    │ Price        │
                    │ Quantity     │
                    │ MatchedAt    │
                    └──────────────┘
```

### 4.2 Schema 定义 (Ent ORM)

```go
// Strategy 策略实体
type Strategy struct {
    GUID                  string  // 主键
    Owner                 int64   // 用户ID
    Symbol                string  // 交易对
    Exchange              string  // 交易所
    Account               string  // 账户
    Mode                  string  // Long/Short
    GridCount             int     // 网格数量
    GridSpacing           float64 // 网格间距
    PositionSize          float64 // 仓位大小
    Leverage              int     // 杠杆
    TriggerTakeProfitPrice *float64 // 止盈价格
    TriggerStopLossPrice   *float64 // 止损价格
    Active                bool    // 是否活跃
    CreatedAt             time.Time
}
```

---

## 5. 数据流设计

### 5.1 用户创建策略流程

```
1. User → Telegram Bot
   │
   ▼
2. /create 命令
   │
   ▼
3. DisplayExchangeSelector
   │
   ▼
4. User 选择交易所 → DisplayGridForm
   │
   ▼
5. User 提交参数 → ValidateAndSave
   │
   ▼
6. StrategyModel.Create() → DB
   │
   ▼
7. NewGridStrategy() → 创建策略实例
   │
   ▼
8. StrategyEngine.StartStrategy() 注册策略
   │
   ├─ 订阅用户订单
   ├─ 订阅市场数据
   └─ 添加到引擎
   │
   ▼
9. 完成 → 发送通知
```

### 5.2 交易执行流程

```
1. Exchange WebSocket → 价格更新
   │
   ▼
2. Subscriber → Push SubMessage
   │
   ▼
3. Engine.run() → 接收消息
   │
   ▼
4. processMarketStats() → 分发到策略
   │
   ▼
5. GridStrategy.OnTicker() 
   │
   ├─ 检查止盈/止损
   ├─ 更新最新价格
   │
   ▼
6. Exchange WebSocket → 订单更新
   │
   ▼
7. processOrders() → 分发到策略
   │
   ▼
8. GridStrategy.OnOrdersChanged()
   │
   ├─ LoadGridStrategyState()
   ├─ Rebalance() → 网格再平衡
   │   ├─ 检查是否需要开单
   │   ├─ 提交新订单
   │   └─ 更新网格状态
   │
   ▼
9. 订单成交 → 重复步骤6
```

---

## 6. WebSocket 架构

### 6.1 订阅者模式

```
┌─────────────────────────────────────────────────────────────┐
│                    Exchange Subscribers                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   LighterSubscriber    ParadexSubscriber   VariationalSub   │
│   ┌────────────┐     ┌────────────┐      ┌────────────┐    │
│   │ WebSocket │     │ WebSocket  │      │ WebSocket  │    │
│   │ Connect   │     │ Connect    │      │ Connect    │    │
│   └─────┬──────┘     └─────┬──────┘      └─────┬──────┘    │
│         │                  │                   │           │
│         ▼                  ▼                   ▼           │
│   ┌─────────────────────────────────────────────────┐      │
│   │              SubMessage Channel                  │      │
│   │   • UserOrders  • MarketStats  • Errors         │      │
│   └─────────────────────┬───────────────────────────┘      │
│                         │                                   │
│                         ▼                                   │
│                  StrategyEngine                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 订阅内容

| 类型 | 说明 |
|------|------|
| 用户订单 | 实时订单状态变化 |
| 市场数据 | 实时价格/标记价格 |

---

## 7. 错误处理与重试

### 7.1 重试队列

```go
type retryItem struct {
    strategyID string
    retryTime  time.Time
    index      int // 堆索引
}
```

**重试流程**:

```
订单失败
    │
    ▼
addToRetryQueue() → 加入最小堆
    │
    ▼
run() 循环检查
    │
    ├─ 未到重试时间 → 等待
    │
    └─ 到达重试时间
        │
        ▼
    executeStrategy() → 重新执行
        │
        ├─ 成功 → removeFromRetryQueue()
        │
        └─ 失败 → 重新加入队列 (指数退避)
```

---

## 8. 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.20+ |
| 数据库 | SQLite3 (WAL 模式) |
| ORM | Ent |
| WebSocket | gorilla/websocket |
| Telegram | telebot.v4 |
| 日志 | logrus |
| 配置 | YAML |
| 密码学 | Starknet (caigo), Ethereum |

---

## 9. 目录结构

```
omni-grid-bot/
├── cmd/
│   └── test/                      # 测试代码
├── data/
│   └── sqlite.db                  # SQLite 数据库
├── etc/
│   ├── config.yaml                # 主配置
│   └── config.yaml.sample         # 配置样例
├── internal/
│   ├── cache/                     # 缓存实现
│   │   ├── lighter_cache.go
│   │   ├── paradex_cache.go
│   │   ├── message_cache.go
│   │   └── pending_orders_cache.go
│   ├── config/
│   │   └── config.go
│   ├── engine/
│   │   ├── engine.go              # 策略引擎核心
│   │   ├── orders.go
│   │   ├── subscriptions.go
│   │   ├── market_stats.go
│   │   └── reptyheap.go          # 重试堆
│   ├── exchange/
│   │   ├── lighter/               # Lighter 适配器
│   │   ├── paradex/               # Paradex 适配器
│   │   ├── variational/           # Variational 适配器
│   │   ├── types.go               # 通用类型
│   │   └── enum.go                # 枚举定义
│   ├── logger/
│   │   └── logger.go
│   ├── model/                     # 数据访问层
│   │   ├── strategy.go
│   │   ├── order.go
│   │   ├── grid.go
│   │   └── matched_trade.go
│   ├── service/
│   │   └── matched_trade_service.go
│   ├── strategy/
│   │   ├── grid_strategy.go       # 网格策略
│   │   ├── grid_state.go          # 网格状态
│   │   └── create.go              # 策略工厂
│   ├── svc/
│   │   └── service_context.go     # 服务上下文
│   ├── telebot/
│   │   ├── telebot.go             # 机器人主类
│   │   ├── handler/               # 处理器
│   │   │   ├── routes.go
│   │   │   ├── create_strategy_handler.go
│   │   │   ├── strategy_list_handler.go
│   │   │   └── ...
│   │   └── pathrouter/            # 路由
│   └── util/                      # 工具函数
├── main.go                         # 程序入口
├── go.mod
└── README.md
```

---

## 10. 启动流程

```
1. main()
   │
   ▼
2. LoadConfig() - 加载配置
   │
   ▼
3. 创建 data 目录
   │
   ▼
4. NewServiceContext() - 初始化服务
   │  ├─ 连接数据库
   │  ├─ 创建代理
   │  ├─ 初始化客户端
   │  └─ 创建缓存
   │
   ▼
5. 启动 Subscribers
   │  ├─ LighterSubscriber.Start()
   │  ├─ ParadexSubscriber.Start()
   │  └─ VariationalSubscriber.Start()
   │
   ▼
6. NewStrategyEngine() - 创建引擎
   │
   ▼
7. startAllStrategy() - 恢复活跃策略
   │
   ▼
8. NewTeleBot() - 启动机器人
   │
   ▼
9. 等待信号 → 优雅停止
```

---

## 11. 设计模式

| 模式 | 应用 |
|------|------|
| 依赖注入 | ServiceContext 作为 DI 容器 |
| 策略模式 | Strategy 接口支持多策略 |
| 订阅者模式 | WebSocket 事件订阅与分发 |
| 工厂模式 | NewGridStrategy() 工厂 |
| 缓存模式 | 多级缓存降低 DB 压力 |
| 重试模式 | 最小堆实现失败重试 |

---

*文档版本: 1.0*  
*更新时间: 2025-03-06*
