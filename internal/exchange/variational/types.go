package variational

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ErrorRes 错误响应
type ErrorRes struct {
	Message string `json:"error_message"` // 错误消息
}

// Error 实现error接口
func (err *ErrorRes) Error() string {
	return fmt.Sprintf("message: %s", err.Message)
}

// LoginRes 登录响应
type LoginRes struct {
	Token string `json:"token"`
}

// 生成签名数据响应
type GenerateSigningDataRes struct {
	Message string `json:"message"`
}

// Instrument 交易品种信息
type Instrument struct {
	InstrumentType   string `json:"instrument_type"`    // 品种类型：永续期货
	Underlying       string `json:"underlying"`         // 标的资产：BTC
	FundingIntervalS int64  `json:"funding_interval_s"` // 资金费率间隔（秒）
	SettlementAsset  string `json:"settlement_asset"`   // 结算资产：USDC
}

// PriceInfo 价格相关信息
type PriceInfo struct {
	Price           decimal.Decimal `json:"price"`            // 当前价格
	NativePrice     decimal.Decimal `json:"native_price"`     // 本地价格
	Delta           decimal.Decimal `json:"delta"`            // Delta值
	Gamma           decimal.Decimal `json:"gamma"`            // Gamma值
	Theta           decimal.Decimal `json:"theta"`            // Theta值
	Vega            decimal.Decimal `json:"vega"`             // Vega值
	Rho             decimal.Decimal `json:"rho"`              // Rho值
	Iv              decimal.Decimal `json:"iv"`               // 隐含波动率
	UnderlyingPrice decimal.Decimal `json:"underlying_price"` // 标的资产价格
	InterestRate    decimal.Decimal `json:"interest_rate"`    // 利率
	Timestamp       time.Time       `json:"timestamp"`        // 价格时间戳
}

// PositionInfo 持仓详细信息
type PositionInfo struct {
	Company           string           `json:"company"`              // 公司标识
	Counterparty      string           `json:"counterparty"`         // 交易对手标识
	Instrument        Instrument       `json:"instrument"`           // 交易品种信息
	PoolLocation      string           `json:"pool_location"`        // 资金池位置
	UpdatedAt         time.Time        `json:"updated_at"`           // 最后更新时间
	OpenedAt          time.Time        `json:"opened_at"`            // 开仓时间
	Qty               decimal.Decimal  `json:"qty"`                  // 当前持仓数量
	AvgEntryPrice     decimal.Decimal  `json:"avg_entry_price"`      // 平均开仓价格
	PrevAvgEntryPrice *decimal.Decimal `json:"prev_avg_entry_price"` // 上一次平均开仓价格
	PrevQty           *decimal.Decimal `json:"prev_qty"`             // 上一次持仓数量
	TakerQty          decimal.Decimal  `json:"taker_qty"`            // Taker持仓数量
	LastLocalSequence int64            `json:"last_local_sequence"`  // 最后本地序列号
}

// Position 持仓信息
type Position struct {
	PositionInfo              PositionInfo           `json:"position_info"`               // 持仓基本信息
	PendingOrderCounts        map[string]interface{} `json:"pending_order_counts"`        // 待处理订单计数
	PriceInfo                 PriceInfo              `json:"price_info"`                  // 价格相关信息
	Value                     decimal.Decimal        `json:"value"`                       // 持仓价值
	Upnl                      decimal.Decimal        `json:"upnl"`                        // 未实现盈亏
	Rpnl                      decimal.Decimal        `json:"rpnl"`                        // 已实现盈亏
	CumFunding                decimal.Decimal        `json:"cum_funding"`                 // 累计资金费用
	EstimatedLiquidationPrice decimal.Decimal        `json:"estimated_liquidation_price"` // 预估强平价格
}

// MarginUsage 保证金信息
type MarginUsage struct {
	InitialMargin     string `json:"initial_margin"`     // 初始保证金要求
	MaintenanceMargin string `json:"maintenance_margin"` // 维持保证金要求
}

// Portfolio 投资组合信息
type Portfolio struct {
	MarginUsage MarginUsageDecimal `json:"margin_usage"` // 保证金使用情况
	Balance     decimal.Decimal    `json:"balance"`      // 账户余额
	Upnl        decimal.Decimal    `json:"upnl"`         // 未实现盈亏
}

// MarginUsageDecimal 保证金信息
type MarginUsageDecimal struct {
	InitialMargin     decimal.Decimal `json:"initial_margin"`     // 初始保证金要求
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"` // 维持保证金要求
}

// PortfolioRes 投资组合响应
type PortfolioRes struct {
	MarginUsage MarginUsage     `json:"margin_usage"` // 保证金使用情况
	Balance     decimal.Decimal `json:"balance"`      // 账户余额
	Upnl        decimal.Decimal `json:"upnl"`         // 未实现盈亏
}

// CreateOrderRes 创建订单响应
type CreateOrderRes struct {
	RfqId           string  `json:"rfq_id"`
	TakeProfitRfqId *string `json:"take_profit_rfq_id"`
	StopLossRfqId   *string `json:"stop_loss_rfq_id"`
}

// OrderStatus 订单状态枚举
type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "pending"  // 订单待处理
	OrderStatusCanceled OrderStatus = "canceled" // 订单已取消
	OrderStatusCleared  OrderStatus = "cleared"  // 订单已清算
	OrderStatusRejected OrderStatus = "rejected" // 订单已驳回
)

// OrderType 订单类型枚举
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"  // 限价订单
	OrderTypeMarket OrderType = "market" // 市价订单
)

// OrderSide 订单方向枚举
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"  // 买入
	OrderSideSell OrderSide = "sell" // 卖出
)

// Pagination 分页信息
type Pagination struct {
	LastPage    *PageInfo `json:"last_page"`    // 最后一页信息
	NextPage    *PageInfo `json:"next_page"`    // 下一页信息
	ObjectCount int       `json:"object_count"` // 对象总数
}

// PageInfo 页面信息
type PageInfo struct {
	Limit  string `json:"limit"`  // 每页限制数量
	Offset string `json:"offset"` // 偏移量
}

// Order 订单信息
type Order struct {
	CancelReason       string           `json:"cancel_reason"`       // 取消原因
	ClearingStatus     *string          `json:"clearing_status"`     // 清算状态
	Company            string           `json:"company"`             // 公司标识
	CreatedAt          time.Time        `json:"created_at"`          // 创建时间
	ExecutionTimestamp *time.Time       `json:"execution_timestamp"` // 执行时间戳
	FailedRiskChecks   []string         `json:"failed_risk_checks"`  // 失败的风险检查
	Instrument         Instrument       `json:"instrument"`          // 交易工具信息
	IsAutoResize       bool             `json:"is_auto_resize"`      // 是否自动调整大小
	IsReduceOnly       bool             `json:"is_reduce_only"`      // 是否仅减仓
	LimitPrice         *decimal.Decimal `json:"limit_price"`         // 限价价格
	MarkPrice          *decimal.Decimal `json:"mark_price"`          // 标记价格
	OrderID            string           `json:"order_id"`            // 订单ID
	OrderType          OrderType        `json:"order_type"`          // 订单类型
	PoolLocation       string           `json:"pool_location"`       // 资金池位置
	Price              *decimal.Decimal `json:"price"`               // 实际成交价格
	Qty                decimal.Decimal  `json:"qty"`                 // 数量
	RfqID              string           `json:"rfq_id"`              // RFQ ID
	Side               OrderSide        `json:"side"`                // 买卖方向
	SlippageLimit      *decimal.Decimal `json:"slippage_limit"`      // 滑点限制
	Status             OrderStatus      `json:"status"`              // 订单状态
	Tif                string           `json:"tif"`                 // 订单有效时间
	TriggerPrice       *decimal.Decimal `json:"trigger_price"`       // 触发价格
	UseMarkPrice       bool             `json:"use_mark_price"`      // 是否使用标记价格
}

// OrdersRes 订单列表响应
type OrdersRes struct {
	Pagination Pagination `json:"pagination"` // 分页信息
	Result     []*Order   `json:"result"`     // 订单结果列表
}
