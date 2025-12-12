package lighter

import (
	lighterhttp "github.com/elliottech/lighter-go/client/http"
	"github.com/fachebot/omni-grid-bot/internal/exchange"

	"github.com/shopspring/decimal"
)

type TX_TYPE uint

const (
	TX_TYPE_CHANGE_PUB_KEY     TX_TYPE = 8  // 更改公钥的交易类型
	TX_TYPE_CREATE_SUB_ACCOUNT TX_TYPE = 9  // 创建子账户的交易类型
	TX_TYPE_CREATE_PUBLIC_POOL TX_TYPE = 10 // 创建公共池的交易类型
	TX_TYPE_UPDATE_PUBLIC_POOL TX_TYPE = 11 // 更新公共池的交易类型
	TX_TYPE_TRANSFER           TX_TYPE = 12 // 转账的交易类型
	TX_TYPE_WITHDRAW           TX_TYPE = 13 // 提现的交易类型
	TX_TYPE_CREATE_ORDER       TX_TYPE = 14 // 创建订单的交易类型
	TX_TYPE_CANCEL_ORDER       TX_TYPE = 15 // 取消订单的交易类型
	TX_TYPE_CANCEL_ALL_ORDERS  TX_TYPE = 16 // 取消所有订单的交易类型
	TX_TYPE_MODIFY_ORDER       TX_TYPE = 17 // 修改订单的交易类型
	TX_TYPE_MINT_SHARES        TX_TYPE = 18 // 发行股份的交易类型
	TX_TYPE_BURN_SHARES        TX_TYPE = 19 // 销毁股份的交易类型
	TX_TYPE_UPDATE_LEVERAGE    TX_TYPE = 20 // 更新杠杆的交易类型
)

// Position 持仓结构体
type Position struct {
	MarketID               int16           `json:"market_id"`                        // 市场ID
	Symbol                 string          `json:"symbol"`                           // 交易对符号
	InitialMarginFraction  decimal.Decimal `json:"initial_margin_fraction"`          // 初始保证金比例
	OpenOrderCount         int64           `json:"open_order_count"`                 // 开放订单数量
	PendingOrderCount      int64           `json:"pending_order_count"`              // 待处理订单数量
	PositionTiedOrderCount int64           `json:"position_tied_order_count"`        // 与持仓绑定的订单数量
	Sign                   int32           `json:"sign"`                             // 持仓方向标识（1为多头，-1为空头）
	Position               decimal.Decimal `json:"position"`                         // 持仓数量
	AvgEntryPrice          decimal.Decimal `json:"avg_entry_price"`                  // 平均入场价格
	PositionValue          decimal.Decimal `json:"position_value"`                   // 持仓价值
	UnrealizedPnl          decimal.Decimal `json:"unrealized_pnl"`                   // 未实现盈亏
	RealizedPnl            decimal.Decimal `json:"realized_pnl"`                     // 已实现盈亏
	LiquidationPrice       decimal.Decimal `json:"liquidation_price"`                // 强平价格
	TotalFundingPaidOut    decimal.Decimal `json:"total_funding_paid_out,omitempty"` // 总资金费用支出
	MarginMode             int32           `json:"margin_mode"`                      // 保证金模式（0为全仓，1为逐仓）
	AllocatedMargin        decimal.Decimal `json:"allocated_margin"`                 // 分配的保证金
}

// PoolInfo 公共池信息结构体
type PoolInfo struct {
	Status                uint8           `json:"status"`                  // 池状态
	OperatorFee           decimal.Decimal `json:"operator_fee"`            // 操作员费用
	MinOperatorShareRate  decimal.Decimal `json:"min_operator_share_rate"` // 最小操作员份额比率
	TotalShares           int64           `json:"total_shares"`            // 总份额数量
	OperatorShares        int64           `json:"operator_shares"`         // 操作员份额数量
	AnnualPercentageYield decimal.Decimal `json:"annual_percentage_yield"` // 年化收益率
	DailyReturns          []DailyReturn   `json:"daily_returns"`           // 每日收益列表
	SharePrices           []SharePrice    `json:"share_prices"`            // 份额价格列表
}

// DailyReturn 每日收益结构体
type DailyReturn struct {
	Timestamp   int64           `json:"timestamp"`    // 时间戳
	DailyReturn decimal.Decimal `json:"daily_return"` // 每日收益率
}

// SharePrice 份额价格结构体
type SharePrice struct {
	Timestamp  int64           `json:"timestamp"`   // 时间戳
	SharePrice decimal.Decimal `json:"share_price"` // 份额价格
}

// Share 份额结构体
type Share struct {
	PublicPoolIndex int64           `json:"public_pool_index"` // 公共池索引
	SharesAmount    int64           `json:"shares_amount"`     // 份额数量
	EntryUsdc       decimal.Decimal `json:"entry_usdc"`        // 入金USDC金额
}

// Account 账户结构体
type Account struct {
	Code                     int32           `json:"code"`                        // 账户状态码
	Message                  string          `json:"message,omitempty"`           // 账户消息
	AccountType              uint8           `json:"account_type"`                // 账户类型
	Index                    int64           `json:"index"`                       // 账户索引
	L1Address                string          `json:"l1_address"`                  // L1地址
	CancelAllTime            int64           `json:"cancel_all_time"`             // 取消所有订单的时间戳
	TotalOrderCount          int64           `json:"total_order_count"`           // 总订单数量
	TotalIsolatedOrderCount  int64           `json:"total_isolated_order_count"`  // 总逐仓订单数量
	PendingOrderCount        int64           `json:"pending_order_count"`         // 待处理订单数量
	AvailableBalance         decimal.Decimal `json:"available_balance"`           // 可用余额
	Status                   uint8           `json:"status"`                      // 账户状态
	Collateral               decimal.Decimal `json:"collateral"`                  // 抵押品金额
	AccountIndex             int64           `json:"account_index"`               // 账户索引
	Name                     string          `json:"name"`                        // 账户名称
	Description              string          `json:"description"`                 // 账户描述
	CanInvite                bool            `json:"can_invite"`                  // 是否可以邀请
	ReferralPointsPercentage string          `json:"referral_points_percentage"`  // 推荐积分百分比（前端使用L1元数据端点后移除）
	Positions                []*Position     `json:"positions"`                   // 持仓列表
	TotalAssetValue          decimal.Decimal `json:"total_asset_value,omitempty"` // 总资产价值
	CrossAssetValue          decimal.Decimal `json:"cross_asset_value,omitempty"` // 全仓资产价值
	PoolInfo                 *PoolInfo       `json:"pool_info,omitempty"`         // 公共池信息
	Shares                   []Share         `json:"shares,omitempty"`            // 份额列表
}

// Accounts 账户列表
type Accounts struct {
	lighterhttp.ResultCode
	Total    int        `json:"total"`
	Accounts []*Account `json:"accounts"`
}

// Transaction 交易结构体
type Transaction struct {
	Hash             string `json:"hash"`              // 交易哈希
	Type             uint8  `json:"type"`              // 交易类型
	Info             string `json:"info"`              // 交易信息
	EventInfo        string `json:"event_info"`        // 事件信息
	Status           int64  `json:"status"`            // 交易状态
	TransactionIndex int64  `json:"transaction_index"` // 交易索引
	L1Address        string `json:"l1_address"`        // L1地址
	AccountIndex     int64  `json:"account_index"`     // 账户索引
	Nonce            int64  `json:"nonce"`             // 交易序号
	ExpireAt         int64  `json:"expire_at"`         // 过期时间
	BlockHeight      int64  `json:"block_height"`      // 区块高度
	QueuedAt         int64  `json:"queued_at"`         // 入队时间
	ExecutedAt       int64  `json:"executed_at"`       // 执行时间
	SequenceIndex    int64  `json:"sequence_index"`    // 序列索引
	ParentHash       string `json:"parent_hash"`       // 父交易哈希
}

// Transactions 交易列表
type Transactions struct {
	lighterhttp.ResultCode
	Txs []*Transaction `json:"txs"`
}

// OrderBookStatus 订单簿状态枚举
type OrderBookStatus string

const (
	StatusInactive OrderBookStatus = "inactive"
	StatusActive   OrderBookStatus = "active"
)

// OrderBookMetadata 订单簿元数据
type OrderBookMetadata struct {
	Symbol                 string          `json:"symbol"`                   // 交易对符号
	MarketID               int16           `json:"market_id"`                // 市场ID
	Status                 OrderBookStatus `json:"status"`                   // 状态：inactive/active
	TakerFee               decimal.Decimal `json:"taker_fee"`                // 吃单手续费
	MakerFee               decimal.Decimal `json:"maker_fee"`                // 挂单手续费
	LiquidationFee         decimal.Decimal `json:"liquidation_fee"`          // 清算手续费
	MinBaseAmount          decimal.Decimal `json:"min_base_amount"`          // 最小基础货币数量
	MinQuoteAmount         decimal.Decimal `json:"min_quote_amount"`         // 最小计价货币数量
	OrderQuoteLimit        string          `json:"order_quote_limit"`        // 订单计价限制（可选字段）
	SupportedSizeDecimals  uint8           `json:"supported_size_decimals"`  // 支持的数量小数位数
	SupportedPriceDecimals uint8           `json:"supported_price_decimals"` // 支持的价格小数位数
	SupportedQuoteDecimals uint8           `json:"supported_quote_decimals"` // 支持的计价小数位数
}

// OrderBooksMetadata 订单簿元数据列表
type OrderBooksMetadata struct {
	lighterhttp.ResultCode
	OrderBooks []*OrderBookMetadata `json:"order_books"`
}

// OrderType 订单类型
type OrderType string

const (
	OrderTypeLimit           OrderType = "limit"             // 限价单
	OrderTypeMarket          OrderType = "market"            // 市价单
	OrderTypeStopLoss        OrderType = "stop-loss"         // 止损单
	OrderTypeStopLossLimit   OrderType = "stop-loss-limit"   // 限价止损单
	OrderTypeTakeProfit      OrderType = "take-profit"       // 止盈单
	OrderTypeTakeProfitLimit OrderType = "take-profit-limit" // 限价止盈单
	OrderTypeTwap            OrderType = "twap"              // 时间加权平均价格单
	OrderTypeTwapSub         OrderType = "twap-sub"          // TWAP子订单
	OrderTypeLiquidation     OrderType = "liquidation"       // 强制平仓单
)

// TimeInForce 订单有效期类型
type TimeInForce string

const (
	TimeInForceGoodTillTime      TimeInForce = "good-till-time"      // 指定时间前有效
	TimeInForceImmediateOrCancel TimeInForce = "immediate-or-cancel" // 立即成交或取消
	TimeInForcePostOnly          TimeInForce = "post-only"           // 只做maker
	TimeInForceUnknown           TimeInForce = "Unknown"             // 未知
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusInProgress                 OrderStatus = "in-progress"                   // 进行中
	OrderStatusPending                    OrderStatus = "pending"                       // 待处理
	OrderStatusOpen                       OrderStatus = "open"                          // 已开启
	OrderStatusFilled                     OrderStatus = "filled"                        // 已成交
	OrderStatusCanceled                   OrderStatus = "canceled"                      // 已取消
	OrderStatusCanceledPostOnly           OrderStatus = "canceled-post-only"            // 因post-only取消
	OrderStatusCanceledReduceOnly         OrderStatus = "canceled-reduce-only"          // 因reduce-only取消
	OrderStatusCanceledPositionNotAllowed OrderStatus = "canceled-position-not-allowed" // 因持仓不允许取消
	OrderStatusCanceledMarginNotAllowed   OrderStatus = "canceled-margin-not-allowed"   // 因保证金不足取消
	OrderStatusCanceledTooMuchSlippage    OrderStatus = "canceled-too-much-slippage"    // 因滑点过大取消
	OrderStatusCanceledNotEnoughLiquidity OrderStatus = "canceled-not-enough-liquidity" // 因流动性不足取消
	OrderStatusCanceledSelfTrade          OrderStatus = "canceled-self-trade"           // 因自成交取消
	OrderStatusCanceledExpired            OrderStatus = "canceled-expired"              // 因过期取消
	OrderStatusCanceledOco                OrderStatus = "canceled-oco"                  // 因OCO取消
	OrderStatusCanceledChild              OrderStatus = "canceled-child"                // 因子订单取消
	OrderStatusCanceledLiquidation        OrderStatus = "canceled-liquidation"          // 因强制平仓取消
)

// TriggerStatus 触发状态
type TriggerStatus string

const (
	TriggerStatusNA          TriggerStatus = "na"           // 不适用
	TriggerStatusReady       TriggerStatus = "ready"        // 准备就绪
	TriggerStatusMarkPrice   TriggerStatus = "mark-price"   // 标记价格触发
	TriggerStatusTwap        TriggerStatus = "twap"         // TWAP触发
	TriggerStatusParentOrder TriggerStatus = "parent-order" // 父订单触发
)

// Order 表示订单信息
type Order struct {
	OrderIndex          int64           `json:"order_index"`           // 订单索引，系统内部唯一标识符
	ClientOrderIndex    int64           `json:"client_order_index"`    // 客户端订单索引
	OrderID             string          `json:"order_id"`              // 订单ID
	ClientOrderID       string          `json:"client_order_id"`       // 客户端订单ID
	MarketIndex         int16           `json:"market_index"`          // 市场索引，标识交易市场
	OwnerAccountIndex   int64           `json:"owner_account_index"`   // 账户拥有者索引
	InitialBaseAmount   decimal.Decimal `json:"initial_base_amount"`   // 初始基础资产数量
	Price               decimal.Decimal `json:"price"`                 // 订单价格
	Nonce               int64           `json:"nonce"`                 // 随机数，防重放攻击
	RemainingBaseAmount decimal.Decimal `json:"remaining_base_amount"` // 剩余基础资产数量
	IsAsk               bool            `json:"is_ask"`                // 是否为卖单
	BaseSize            int64           `json:"base_size"`             // 基础资产规模
	BasePrice           int64           `json:"base_price"`            // 基础价格（整数形式）
	FilledBaseAmount    decimal.Decimal `json:"filled_base_amount"`    // 已成交的基础资产数量
	FilledQuoteAmount   decimal.Decimal `json:"filled_quote_amount"`   // 已成交的计价资产数量
	Type                OrderType       `json:"type"`                  // 订单类型
	TimeInForce         TimeInForce     `json:"time_in_force"`         // 订单有效期类型
	ReduceOnly          bool            `json:"reduce_only"`           // 是否为只减仓订单
	TriggerPrice        decimal.Decimal `json:"trigger_price"`         // 触发价格
	OrderExpiry         int64           `json:"order_expiry"`          // 订单过期时间戳
	Status              OrderStatus     `json:"status"`                // 订单状态
	TriggerStatus       TriggerStatus   `json:"trigger_status"`        // 触发状态
	TriggerTime         int64           `json:"trigger_time"`          // 触发时间戳
	ParentOrderIndex    int64           `json:"parent_order_index"`    // 父订单索引
	ParentOrderID       string          `json:"parent_order_id"`       // 父订单ID
	ToTriggerOrderID0   string          `json:"to_trigger_order_id_0"` // 要触发的订单ID 0
	ToTriggerOrderID1   string          `json:"to_trigger_order_id_1"` // 要触发的订单ID 1
	ToCancelOrderID0    string          `json:"to_cancel_order_id_0"`  // 要取消的订单ID 0
	BlockHeight         int64           `json:"block_height"`          // 区块高度
	Timestamp           int64           `json:"timestamp"`             // 时间戳
	CreatedAt           int64           `json:"created_at"`            // 创建时间戳
	UpdatedAt           int64           `json:"updated_at"`            // 更新时间戳
}

// Orders 订单列表
type Orders struct {
	lighterhttp.ResultCode
	NextCursor string   `json:"next_cursor,omitempty"`
	Orders     []*Order `json:"orders"`
}

// ORDER_TYPE 定义订单类型
type ORDER_TYPE uint8

const (
	ORDER_TYPE_LIMIT             ORDER_TYPE = 0 // 限价单：指定价格买入或卖出
	ORDER_TYPE_MARKET            ORDER_TYPE = 1 // 市价单：按市场当前价格立即执行
	ORDER_TYPE_STOP_LOSS         ORDER_TYPE = 2 // 止损单：价格达到止损价时按市价执行
	ORDER_TYPE_STOP_LOSS_LIMIT   ORDER_TYPE = 3 // 止损限价单：价格达到止损价时按限价执行
	ORDER_TYPE_TAKE_PROFIT       ORDER_TYPE = 4 // 止盈单：价格达到止盈价时按市价执行
	ORDER_TYPE_TAKE_PROFIT_LIMIT ORDER_TYPE = 5 // 止盈限价单：价格达到止盈价时按限价执行
	ORDER_TYPE_TWAP              ORDER_TYPE = 6 // TWAP订单：时间加权平均价格订单
)

// ORDER_TIME_IN_FORCE 定义订单有效期类型
type ORDER_TIME_IN_FORCE uint8

const (
	ORDER_TIME_IN_FORCE_IMMEDIATE_OR_CANCEL ORDER_TIME_IN_FORCE = 0 // IOC：立即执行或取消，不能完全成交的部分立即取消
	ORDER_TIME_IN_FORCE_GOOD_TILL_TIME      ORDER_TIME_IN_FORCE = 1 // GTT：在指定时间前有效
	ORDER_TIME_IN_FORCE_POST_ONLY           ORDER_TIME_IN_FORCE = 2 // POST_ONLY：只能作为挂单，不能立即成交
)

// CreateOrderTxReq 创建订单交易请求
type CreateOrderTxReq struct {
	MarketIndex      int16               // 市场索引，标识交易对
	ClientOrderIndex int64               // 客户端订单索引，用于客户端跟踪订单
	BaseAmount       int64               // 基础资产数量（通常以最小单位表示）
	Price            uint32              // 订单价格（对于限价单）
	IsAsk            uint8               // 订单方向：1表示卖单(ask)，0表示买单(bid)
	Type             ORDER_TYPE          // 订单类型
	TimeInForce      ORDER_TIME_IN_FORCE // 订单有效期类型
	ReduceOnly       uint8               // 是否为减仓单：1表示只能减少持仓，0表示可以增加持仓
	TriggerPrice     uint32              // 触发价格（用于止损/止盈订单）
	OrderExpiry      int64               // 订单过期时间戳
}

// OrderBookDetail 订单簿详情
type OrderBookDetail struct {
	LastTradePrice decimal.Decimal `json:"last_trade_price"` // 最后交易价格
}

// OrderBookDetails 订单簿详情响应
type OrderBookDetails struct {
	lighterhttp.ResultCode
	OrderBookDetails []OrderBookDetail `json:"order_book_details"`
}

// UpdateLeverageTxReq 更新杠杆交易请求
type UpdateLeverageTxReq struct {
	MarketIndex int16               // 市场索引，标识交易对
	Leverage    uint                // 杠杆级别
	MarginMode  exchange.MarginMode // 保证金模式
}

// MarketStats 市场状态数据
type MarketStats struct {
	MarketID              int16           `json:"market_id"`                // 市场ID
	IndexPrice            decimal.Decimal `json:"index_price"`              // 指数价格
	MarkPrice             decimal.Decimal `json:"mark_price"`               // 标记价格
	OpenInterest          decimal.Decimal `json:"open_interest"`            // 持仓量
	LastTradePrice        decimal.Decimal `json:"last_trade_price"`         // 最后交易价格
	CurrentFundingRate    decimal.Decimal `json:"current_funding_rate"`     // 当前资金费率
	FundingRate           decimal.Decimal `json:"funding_rate"`             // 资金费率
	FundingTimestamp      int64           `json:"funding_timestamp"`        // 资金费时间戳
	DailyBaseTokenVolume  decimal.Decimal `json:"daily_base_token_volume"`  // 日基础代币交易量
	DailyQuoteTokenVolume decimal.Decimal `json:"daily_quote_token_volume"` // 日计价代币交易量
	DailyPriceLow         decimal.Decimal `json:"daily_price_low"`          // 日最低价
	DailyPriceHigh        decimal.Decimal `json:"daily_price_high"`         // 日最高价
	DailyPriceChange      decimal.Decimal `json:"daily_price_change"`       // 日价格变化
}

// TxHash 交易hash
type TxHash struct {
	lighterhttp.ResultCode
	TxHash                   string `json:"tx_hash"`
	PredictedExecutionTimeMs int64  `json:"predicted_execution_time_ms"`
}

// RespSendTxBatch 批量发送交易响应
type RespSendTxBatch struct {
	lighterhttp.ResultCode
	TxHash []string `json:"tx_hash"`
}

// WebSocketMessage 推送消息
type WebSocketMessage struct {
	Type        string              `json:"type"`
	Channel     string              `json:"channel"`
	Orders      map[string][]*Order `json:"orders,omitempty"`
	MarketStats *MarketStats        `json:"market_stats,omitempty"`
}
