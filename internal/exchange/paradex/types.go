package paradex

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// MarginType 保证金类型
type MarginType string

// VerificationType 验证类型
type VerificationType string

// 验证类型常量定义
var (
	VerificationTypeOnboarding VerificationType = "Onboarding" // 入驻验证
	VerificationTypeAuth       VerificationType = "Auth"       // 身份验证
	VerificationTypeOrder      VerificationType = "Order"      // 订单验证
)

// AuthRes 身份验证响应
type AuthRes struct {
	JwtToken string `json:"jwt_token"` // JWT令牌
}

// ErrorRes 错误响应
type ErrorRes struct {
	Code    string `json:"error"`   // 错误代码
	Message string `json:"message"` // 错误消息
}

// Error 实现error接口
func (err *ErrorRes) Error() string {
	return fmt.Sprintf("code: %s, message: %s", err.Code, err.Message)
}

// OnboardingRes 入驻响应
type OnboardingRes struct {
	PublicKey string `json:"public_key"` // 公钥
}

// BridgedToken 跨链代币信息
type BridgedToken struct {
	Name            string `json:"name"`              // 代币名称
	Symbol          string `json:"symbol"`            // 代币符号
	Decimals        int    `json:"decimals"`          // 小数位数
	L1TokenAddress  string `json:"l1_token_address"`  // L1层代币地址
	L1BridgeAddress string `json:"l1_bridge_address"` // L1层桥接地址
	L2TokenAddress  string `json:"l2_token_address"`  // L2层代币地址
	L2BridgeAddress string `json:"l2_bridge_address"` // L2层桥接地址
}

// SystemConfigRes 系统配置响应
type SystemConfigRes struct {
	GatewayUrl                string         `json:"starknet_gateway_url" example:"https://potc-testnet-02.starknet.io"`                                       // Starknet网关URL
	ChainId                   string         `json:"starknet_chain_id" example:"SN_CHAIN_ID"`                                                                  // Starknet链ID
	BlockExplorerUrl          string         `json:"block_explorer_url" example:"https://voyager.testnet.paradex.trade/"`                                      // 区块浏览器URL
	ParaclearAddress          string         `json:"paraclear_address" example:"0x4638e3041366aa71720be63e32e53e1223316c7f0d56f7aa617542ed1e7554d"`            // Paraclear合约地址
	ParaclearDecimals         int            `json:"paraclear_decimals"`                                                                                       // Paraclear小数位数
	ParaclearAccountProxyHash string         `json:"paraclear_account_proxy_hash" example:"0x3530cc4759d78042f1b543bf797f5f3d647cde0388c33734cf91b7f7b9314a9"` // Paraclear账户代理哈希
	ParaclearAccountHash      string         `json:"paraclear_account_hash" example:"0x033434ad846cdd5f23eb73ff09fe6fddd568284a0fb7d1be20ee482f044dabe2"`      // Paraclear账户哈希
	BridgedTokens             []BridgedToken `json:"bridged_tokens"`                                                                                           // 跨链代币列表
	L1CoreContractAddress     string         `json:"l1_core_contract_address" example:"0x182FE62c57461d4c5Ab1aE6F04f1D51aA1607daf"`                            // L1核心合约地址
	L1OperatorAddress         string         `json:"l1_operator_address" example:"0x63e762538C70442758Fd622116d817761c94FD6A"`                                 // L1操作员地址
	L1ChainId                 string         `json:"l1_chain_id" example:"5"`                                                                                  // L1链ID
}

// OnboardingReq 入驻请求
type OnboardingReq struct {
	PublicKey string `json:"public_key"` // 公钥
}

// AccountInfoRes 账户信息响应
type AccountInfoRes struct {
	Account                      string `json:"account"`                        // 账户地址
	AccountValue                 string `json:"account_value"`                  // 账户价值
	FreeCollateral               string `json:"free_collateral"`                // 可用抵押品
	InitialMarginRequirement     string `json:"initial_margin_requirement"`     // 初始保证金要求
	MaintenanceMarginRequirement string `json:"maintenance_margin_requirement"` // 维持保证金要求
	MarginCushion                string `json:"margin_cushion"`                 // 保证金缓冲
	SeqNo                        int64  `json:"seq_no"`                         // 唯一递增序号
	SettlementAsset              string `json:"settlement_asset"`               // 结算资产
	Status                       string `json:"status"`                         // 账户状态
	TotalCollateral              string `json:"total_collateral"`               // 总抵押品
	UpdatedAt                    int64  `json:"updated_at"`                     // 更新时间(毫秒时间戳)
}

// PositionSide 持仓方向
type PositionSide string

const (
	PositionSideShort PositionSide = "SHORT" // 空头
	PositionSideLong  PositionSide = "LONG"  // 多头
)

// PositionStatus 持仓状态
type PositionStatus string

const (
	PositionStatusOpen   PositionStatus = "OPEN"   // 开仓
	PositionStatusClosed PositionStatus = "CLOSED" // 平仓
)

// Position 持仓信息
type Position struct {
	Account                      string          `json:"account"`                         // 账户地址
	AverageEntryPrice            decimal.Decimal `json:"average_entry_price"`             // 平均入场价格
	AverageEntryPriceUSD         decimal.Decimal `json:"average_entry_price_usd"`         // 平均入场价格(美元)
	AverageExitPrice             decimal.Decimal `json:"average_exit_price"`              // 平均出场价格
	CachedFundingIndex           decimal.Decimal `json:"cached_funding_index"`            // 缓存的资金费率指数
	ClosedAt                     int64           `json:"closed_at"`                       // 平仓时间(毫秒时间戳)
	Cost                         decimal.Decimal `json:"cost"`                            // 成本
	CostUSD                      decimal.Decimal `json:"cost_usd"`                        // 成本(美元)
	CreatedAt                    int64           `json:"created_at"`                      // 创建时间(毫秒时间戳)
	ID                           string          `json:"id"`                              // 持仓ID
	LastFillID                   string          `json:"last_fill_id"`                    // 最后成交ID
	LastUpdatedAt                int64           `json:"last_updated_at"`                 // 最后更新时间(毫秒时间戳)
	Leverage                     string          `json:"leverage"`                        // 杠杆倍数
	LiquidationPrice             string          `json:"liquidation_price"`               // 强平价格
	Market                       string          `json:"market"`                          // 市场
	RealizedPositionalFundingPnl decimal.Decimal `json:"realized_positional_funding_pnl"` // 已实现资金费盈亏
	RealizedPositionalPnl        decimal.Decimal `json:"realized_positional_pnl"`         // 已实现持仓盈亏
	SeqNo                        int64           `json:"seq_no"`                          // 唯一递增序号
	Side                         PositionSide    `json:"side"`                            // 持仓方向
	Size                         decimal.Decimal `json:"size"`                            // 持仓数量
	Status                       PositionStatus  `json:"status"`                          // 持仓状态
	UnrealizedFundingPnl         decimal.Decimal `json:"unrealized_funding_pnl"`          // 未实现资金费盈亏
	UnrealizedPnl                decimal.Decimal `json:"unrealized_pnl"`                  // 未实现盈亏
}

// PositionRes 持仓响应
type PositionRes struct {
	Results []*Position `json:"results"` // 持仓列表
}

// OrderFlag 订单标志
type OrderFlag string

const (
	OrderFlagReduceOnly                OrderFlag = "REDUCE_ONLY"                  // 只减仓
	OrderFlagStopConditionBelowTrigger OrderFlag = "STOP_CONDITION_BELOW_TRIGGER" // 止损条件低于触发价
	OrderFlagStopConditionAboveTrigger OrderFlag = "STOP_CONDITION_ABOVE_TRIGGER" // 止损条件高于触发价
	OrderFlagInteractive               OrderFlag = "INTERACTIVE"                  // 交互式订单
)

// OrderInstruction 订单执行指令
type OrderInstruction string

const (
	OrderInstructionGTC      OrderInstruction = "GTC"       // Good Till Cancel - 取消前有效
	OrderInstructionPostOnly OrderInstruction = "POST_ONLY" // Post Only - 只做挂单方
	OrderInstructionIOC      OrderInstruction = "IOC"       // Immediate or Cancel - 立即成交或取消
	OrderInstructionRPI      OrderInstruction = "RPI"       // Reduce Position Immediately - 立即减仓
)

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"  // 买入
	OrderSideSell OrderSide = "SELL" // 卖出
)

// Get 获取订单方向的数字表示
func (s OrderSide) Get() string {
	if s == OrderSideBuy {
		return "1" // 买入返回1
	} else {
		return "2" // 卖出返回2
	}
}

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusNew         OrderStatus = "NEW"         // 新建
	OrderStatusUntriggered OrderStatus = "UNTRIGGERED" // 未触发
	OrderStatusOpen        OrderStatus = "OPEN"        // 开启
	OrderStatusClosed      OrderStatus = "CLOSED"      // 关闭
)

// STPMode 自成交防范模式(Self-Trade Prevention)
type STPMode string

const (
	STPModeExpireMaker STPMode = "EXPIRE_MAKER" // 取消挂单方
	STPModeExpireTaker STPMode = "EXPIRE_TAKER" // 取消吃单方
	STPModeExpireBoth  STPMode = "EXPIRE_BOTH"  // 取消双方
)

// OrderType 订单类型
type OrderType string

const (
	OrderTypeMarket           OrderType = "MARKET"             // 市价单
	OrderTypeLimit            OrderType = "LIMIT"              // 限价单
	OrderTypeStopLimit        OrderType = "STOP_LIMIT"         // 止损限价单
	OrderTypeStopMarket       OrderType = "STOP_MARKET"        // 止损市价单
	OrderTypeTakeProfitLimit  OrderType = "TAKE_PROFIT_LIMIT"  // 止盈限价单
	OrderTypeTakeProfitMarket OrderType = "TAKE_PROFIT_MARKET" // 止盈市价单
	OrderTypeStopLossMarket   OrderType = "STOP_LOSS_MARKET"   // 止损市价单
	OrderTypeStopLossLimit    OrderType = "STOP_LOSS_LIMIT"    // 止损限价单
)

// RequestInfo 请求信息
type RequestInfo struct {
	ID          string `json:"id"`           // 请求ID
	Message     string `json:"message"`      // 消息
	RequestType string `json:"request_type"` // 请求类型
	Status      string `json:"status"`       // 状态
}

// Order 订单信息
type Order struct {
	Account       string           `json:"account"`         // Paradex账户地址
	AvgFillPrice  string           `json:"avg_fill_price"`  // 订单平均成交价格
	CancelReason  string           `json:"cancel_reason"`   // 订单取消原因
	ClientID      string           `json:"client_id"`       // 客户端订单ID
	CreatedAt     int64            `json:"created_at"`      // 订单创建时间(毫秒时间戳)
	Flags         []OrderFlag      `json:"flags"`           // 订单标志
	ID            string           `json:"id"`              // Paradex生成的唯一订单标识符
	Instruction   OrderInstruction `json:"instruction"`     // 订单执行指令
	LastUpdatedAt int64            `json:"last_updated_at"` // 订单最后更新时间(毫秒时间戳)
	Market        string           `json:"market"`          // 市场
	Price         decimal.Decimal  `json:"price"`           // 订单价格(市价单为0)
	PublishedAt   int64            `json:"published_at"`    // 订单发送给客户端的时间戳(毫秒)
	ReceivedAt    int64            `json:"received_at"`     // API服务接收订单的时间戳(毫秒)
	RemainingSize decimal.Decimal  `json:"remaining_size"`  // 订单剩余数量
	RequestInfo   *RequestInfo     `json:"request_info"`    // 订单的附加请求信息
	SeqNo         int64            `json:"seq_no"`          // 唯一递增序号
	Side          OrderSide        `json:"side"`            // 订单方向
	Size          decimal.Decimal  `json:"size"`            // 订单数量
	Status        OrderStatus      `json:"status"`          // 订单状态
	STP           STPMode          `json:"stp"`             // 自成交防范模式
	Timestamp     int64            `json:"timestamp"`       // 订单签名时间戳(毫秒)
	TriggerPrice  decimal.Decimal  `json:"trigger_price"`   // 止损单触发价格
	Type          OrderType        `json:"type"`            // 订单类型
}

// OrdersRes 订单响应
type OrdersRes struct {
	Next    *string  `json:"next"`    // 获取下一组记录的指针(如果没有更多记录则为null)
	Prev    *string  `json:"prev"`    // 获取上一组记录的指针(如果没有更多记录则为null)
	Results []*Order `json:"results"` // 订单列表
}

// OpenOrdersRes 未结订单响应
type OpenOrdersRes struct {
	Results []*Order `json:"results"` // 订单列表
}
