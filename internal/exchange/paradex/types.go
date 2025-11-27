package paradex

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// MarginType 保证金类型
type MarginType string

const (
	MarginTypeCross    MarginType = "CROSS"
	MarginTypeIsolated MarginType = "ISOLATED"
)

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
	Account                      string          `json:"account"`                        // 账户地址
	AccountValue                 decimal.Decimal `json:"account_value"`                  // 账户价值
	FreeCollateral               decimal.Decimal `json:"free_collateral"`                // 可用抵押品
	InitialMarginRequirement     decimal.Decimal `json:"initial_margin_requirement"`     // 初始保证金要求
	MaintenanceMarginRequirement decimal.Decimal `json:"maintenance_margin_requirement"` // 维持保证金要求
	MarginCushion                string          `json:"margin_cushion"`                 // 保证金缓冲
	SeqNo                        int64           `json:"seq_no"`                         // 唯一递增序号
	SettlementAsset              string          `json:"settlement_asset"`               // 结算资产
	Status                       string          `json:"status"`                         // 账户状态
	TotalCollateral              decimal.Decimal `json:"total_collateral"`               // 总抵押品
	UpdatedAt                    int64           `json:"updated_at"`                     // 更新时间(毫秒时间戳)
}

// AccountSummary 用户账户摘要信息
type AccountSummary struct {
	Account                      string          `json:"account"`                        // 用户StarkNet账户地址
	AccountValue                 decimal.Decimal `json:"account_value"`                  // 当前账户价值[含未实现盈亏]
	FreeCollateral               decimal.Decimal `json:"free_collateral"`                // 可用自由保证金(超出初始保证金要求的账户价值)
	InitialMarginRequirement     decimal.Decimal `json:"initial_margin_requirement"`     // 开仓现有头寸所需金额
	MaintenanceMarginRequirement decimal.Decimal `json:"maintenance_margin_requirement"` // 维持现有头寸所需金额
	MarginCushion                decimal.Decimal `json:"margin_cushion"`                 // 超出维持保证金要求的账户价值
	SeqNo                        int64           `json:"seq_no"`                         // 唯一递增编号(用于去重)
	SettlementAsset              string          `json:"settlement_asset"`               // 账户结算资产
	Status                       string          `json:"status"`                         // 账户状态(如ACTIVE, LIQUIDATION)
	TotalCollateral              decimal.Decimal `json:"total_collateral"`               // 用户总保证金
	UpdatedAt                    int64           `json:"updated_at"`                     // 账户最后更新时间
}

// AccountSummaries 账户摘要列表
type AccountSummaries []*AccountSummary

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

// STPType 自成交预防类型
type STPType string

const (
	STPExpireMaker STPType = "EXPIRE_MAKER" // 使Maker订单过期
	STPExpireTaker STPType = "EXPIRE_TAKER" // 使Taker订单过期
	STPExpireBoth  STPType = "EXPIRE_BOTH"  // 使双方订单都过期
)

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
	InstructionGTC      OrderInstruction = "GTC"       // 成交或取消
	InstructionPOSTONLY OrderInstruction = "POST_ONLY" // 只做Maker
	InstructionIOC      OrderInstruction = "IOC"       // 立即成交或取消
	InstructionRPI      OrderInstruction = "RPI"       // 保留挂单
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
	Next    string   `json:"next"`    // 获取下一组记录的指针(如果没有更多记录则为null)
	Prev    string   `json:"prev"`    // 获取上一组记录的指针(如果没有更多记录则为null)
	Results []*Order `json:"results"` // 订单列表
}

// OpenOrdersRes 未结订单响应
type OpenOrdersRes struct {
	Results []*Order `json:"results"` // 订单列表
}

// ChainDetails 链详情
type ChainDetails struct {
	CollateralAddress    string          `json:"collateral_address"`     // 抵押品地址
	ContractAddress      string          `json:"contract_address"`       // 合约地址
	FeeAccountAddress    string          `json:"fee_account_address"`    // 费用账户地址
	FeeMaker             decimal.Decimal `json:"fee_maker"`              // Maker费用
	FeeTaker             decimal.Decimal `json:"fee_taker"`              // Taker费用
	InsuranceFundAddress string          `json:"insurance_fund_address"` // 保险基金地址
	LiquidationFee       decimal.Decimal `json:"liquidation_fee"`        // 清算费用
	OracleAddress        string          `json:"oracle_address"`         // 预言机地址
	Symbol               string          `json:"symbol"`                 // 符号
}

// Delta1CrossMarginParams Delta1交叉保证金参数
type Delta1CrossMarginParams struct {
	IMFBase   decimal.Decimal `json:"imf_base"`   // 初始保证金因子基础
	IMFFactor decimal.Decimal `json:"imf_factor"` // 初始保证金因子
	IMFShift  decimal.Decimal `json:"imf_shift"`  // 初始保证金因子偏移
	MMFFactor decimal.Decimal `json:"mmf_factor"` // 维持保证金因子
}

// FeeConfig 费用配置
type FeeConfig struct {
	APIFee         FeeDetail `json:"api_fee"`         // API费用
	InteractiveFee FeeDetail `json:"interactive_fee"` // 交互式费用
	RPIFee         FeeDetail `json:"rpi_fee"`         // RPI费用
}

// FeeDetail 费用详情
type FeeDetail struct {
	MakerFee FeeRate `json:"maker_fee"` // Maker费率
	TakerFee FeeRate `json:"taker_fee"` // Taker费率
}

// FeeRate 费率
type FeeRate struct {
	Fee      decimal.Decimal `json:"fee"`       // 费率
	FeeCap   decimal.Decimal `json:"fee_cap"`   // 费率上限
	FeeFloor decimal.Decimal `json:"fee_floor"` // 费率下限
}

// OptionCrossMarginParams 期权交叉保证金参数
type OptionCrossMarginParams struct {
	IMF *MarginFactors `json:"imf"` // 初始保证金因子
	MMF *MarginFactors `json:"mmf"` // 维持保证金因子
}

// MarginFactors 保证金因子
type MarginFactors struct {
	LongITM           decimal.Decimal `json:"long_itm"`           // 多头实值
	PremiumMultiplier decimal.Decimal `json:"premium_multiplier"` // 权利金乘数
	ShortITM          decimal.Decimal `json:"short_itm"`          // 空头实值
	ShortOTM          decimal.Decimal `json:"short_otm"`          // 空头虚值
	ShortPutCap       decimal.Decimal `json:"short_put_cap"`      // 空头看跌上限
}

// Market 市场信息
type Market struct {
	AssetKind               string                  `json:"asset_kind"`                 // 资产类型: PERP, PERP_OPTION
	BaseCurrency            string                  `json:"base_currency"`              // 市场基础货币
	ChainDetails            ChainDetails            `json:"chain_details"`              // 链详情
	ClampRate               decimal.Decimal         `json:"clamp_rate"`                 // 钳制率
	Delta1CrossMarginParams Delta1CrossMarginParams `json:"delta1_cross_margin_params"` // Delta1交叉保证金参数
	ExpiryAt                int64                   `json:"expiry_at"`                  // 市场到期时间
	FeeConfig               FeeConfig               `json:"fee_config"`                 // 费用配置
	FundingMultiplier       float64                 `json:"funding_multiplier"`         // 资金费率乘数
	FundingPeriodHours      float64                 `json:"funding_period_hours"`       // 资金费率周期(小时)
	InterestRate            decimal.Decimal         `json:"interest_rate"`              // 利率
	IVBandsWidth            decimal.Decimal         `json:"iv_bands_width"`             // IV带宽
	MarketKind              string                  `json:"market_kind"`                // 市场保证金模式: cross, isolated, isolated_margin
	MaxFundingRate          decimal.Decimal         `json:"max_funding_rate"`           // 最大资金费率
	MaxFundingRateChange    decimal.Decimal         `json:"max_funding_rate_change"`    // 最大资金费率变化
	MaxOpenOrders           int                     `json:"max_open_orders"`            // 最大挂单数
	MaxOrderSize            decimal.Decimal         `json:"max_order_size"`             // 最大订单大小(基础货币)
	MaxSlippage             decimal.Decimal         `json:"max_slippage"`               // 默认最大滑点
	MaxTobSpread            decimal.Decimal         `json:"max_tob_spread"`             // 最大TOB价差
	MinNotional             decimal.Decimal         `json:"min_notional"`               // 最小订单名义价值(USD)
	OpenAt                  int64                   `json:"open_at"`                    // 市场开放时间(毫秒)
	OptionCrossMarginParams OptionCrossMarginParams `json:"option_cross_margin_params"` // 期权交叉保证金参数
	OptionType              string                  `json:"option_type"`                // 期权类型: PUT, CALL
	OracleEwmaFactor        decimal.Decimal         `json:"oracle_ewma_factor"`         // 预言机EWMA因子
	OrderSizeIncrement      decimal.Decimal         `json:"order_size_increment"`       // 订单大小最小增量
	PositionLimit           decimal.Decimal         `json:"position_limit"`             // 持仓限制(基础货币)
	PriceBandsWidth         decimal.Decimal         `json:"price_bands_width"`          // 价格带宽
	PriceFeedID             string                  `json:"price_feed_id"`              // 价格源ID
	PriceTickSize           decimal.Decimal         `json:"price_tick_size"`            // 最小价格变动单位(USD)
	QuoteCurrency           string                  `json:"quote_currency"`             // 报价货币
	SettlementCurrency      string                  `json:"settlement_currency"`        // 结算货币
	StrikePrice             decimal.Decimal         `json:"strike_price"`               // 行权价格
	Symbol                  string                  `json:"symbol"`                     // 市场符号
	Tags                    []string                `json:"tags"`                       // 市场标签
}

// MarketRes 市场响应
type MarketRes struct {
	Results []*Market `json:"results"`
}

// Greeks 希腊值
type Greeks struct {
	Delta decimal.Decimal `json:"delta"` // Delta值
	Gamma decimal.Decimal `json:"gamma"` // Gamma值
	Rho   decimal.Decimal `json:"rho"`   // Rho值
	Vanna decimal.Decimal `json:"vanna"` // Vanna值
	Vega  decimal.Decimal `json:"vega"`  // Vega值
	Volga decimal.Decimal `json:"volga"` // Volga值
}

// MarketSummary 市场概要信息
type MarketSummary struct {
	Ask                decimal.Decimal `json:"ask"`                   // 最佳卖单价
	AskIv              decimal.Decimal `json:"ask_iv"`                // 卖单隐含波动率（期权）
	Bid                decimal.Decimal `json:"bid"`                   // 最佳买单价
	BidIv              decimal.Decimal `json:"bid_iv"`                // 买单隐含波动率（期权）
	CreatedAt          int64           `json:"created_at"`            // 市场概要创建时间
	FundingRate        decimal.Decimal `json:"funding_rate"`          // 原始资金费率（对应实际资金周期）
	FutureFundingRate  decimal.Decimal `json:"future_funding_rate"`   // 期权平滑后的期货资金费率
	Greeks             Greeks          `json:"greeks"`                // 希腊值（Delta、Gamma、Vega等）
	LastIv             decimal.Decimal `json:"last_iv"`               // 最后成交价隐含波动率（期权）
	LastTradedPrice    decimal.Decimal `json:"last_traded_price"`     // 最后成交价格
	MarkIv             decimal.Decimal `json:"mark_iv"`               // 标记价格隐含波动率（期权）
	MarkPrice          decimal.Decimal `json:"mark_price"`            // 标记价格
	OpenInterest       decimal.Decimal `json:"open_interest"`         // 未平仓合约量（基础货币）
	PriceChangeRate24h decimal.Decimal `json:"price_change_rate_24h"` // 24小时价格变化率
	Symbol             string          `json:"symbol"`                // 市场交易对符号
	TotalVolume        decimal.Decimal `json:"total_volume"`          // 生命周期总交易量（USD）
	UnderlyingPrice    decimal.Decimal `json:"underlying_price"`      // 标的资产价格（现货价格）
	Volume24h          decimal.Decimal `json:"volume_24h"`            // 24小时交易量（USD）
}

// MarketSummaryRes 市场概要响应
type MarketSummaryRes struct {
	Results []*MarketSummary `json:"results"` // 市场概要结果列表
}

// Discord Discord账户信息
type Discord struct {
	ID       string `json:"id"`        // Discord ID
	ImageURL string `json:"image_url"` // 头像URL
	Username string `json:"username"`  // 用户名
}

// Twitter Twitter账户信息
type Twitter struct {
	ID       string `json:"id"`        // Twitter ID
	ImageURL string `json:"image_url"` // 头像URL
	Username string `json:"username"`  // 用户名
}

// NFT NFT信息
type NFT struct {
	CollectionAddress string   `json:"collection_address"` // 集合地址
	CollectionName    string   `json:"collection_name"`    // 集合名称
	Description       string   `json:"description"`        // 描述
	ID                string   `json:"id"`                 // NFT ID
	ImageURL          string   `json:"image_url"`          // 图片URL
	Name              string   `json:"name"`               // NFT名称
	Price             NFTPrice `json:"price"`              // 价格信息
}

// NFTPrice NFT价格信息
type NFTPrice struct {
	Currency string          `json:"currency"` // 货币类型
	Decimals int             `json:"decimals"` // 小数位数
	Value    decimal.Decimal `json:"value"`    // 价格值
}

// Notifications 通知设置
type Notifications struct {
	Announcements bool `json:"announcements"` // 公告通知
	FillSound     bool `json:"fill_sound"`    // 成交声音
	Fills         bool `json:"fills"`         // 成交通知
	Orders        bool `json:"orders"`        // 订单通知
	Transfers     bool `json:"transfers"`     // 转账通知
}

// Referral 推荐计划信息
type Referral struct {
	CommissionRate       decimal.Decimal `json:"commission_rate"`         // 推荐人佣金比例
	CommissionVolumeCap  decimal.Decimal `json:"commission_volume_cap"`   // 佣金交易量上限
	DiscountRate         decimal.Decimal `json:"discount_rate"`           // 被推荐人折扣比例
	DiscountVolumeCap    decimal.Decimal `json:"discount_volume_cap"`     // 折扣交易量上限
	MinimumVolume        decimal.Decimal `json:"minimum_volume"`          // 参与计划所需最小交易量
	Name                 string          `json:"name"`                    // 推荐计划名称
	PointsBonusRate      decimal.Decimal `json:"points_bonus_rate"`       // 被推荐人积分奖励比例
	PointsBonusVolumeCap decimal.Decimal `json:"points_bonus_volume_cap"` // 积分奖励交易量上限
	ReferralType         string          `json:"referral_type"`           // 推荐类型
}

// ShareRate 分成比例信息
type ShareRate struct {
	LastUpdatedAt int64           `json:"last_updated_at"` // 最后更新时间(毫秒时间戳)
	ShareRate     decimal.Decimal `json:"share_rate"`      // 分成比例
}

// UserProfile 用户配置信息
type UserProfile struct {
	Discord             *Discord          `json:"discord"`               // Discord信息
	IsUsernamePrivate   bool              `json:"is_username_private"`   // 用户名是否私有
	MarketMaxSlippage   map[string]string `json:"market_max_slippage"`   // 市场最大滑点配置
	MarketedBy          string            `json:"marketed_by"`           // 营销来源
	NFTs                []*NFT            `json:"nfts"`                  // NFT列表
	Notifications       Notifications     `json:"notifications"`         // 通知设置
	Referral            Referral          `json:"referral"`              // 推荐计划信息
	ReferralCode        string            `json:"referral_code"`         // 推荐码
	ReferredBy          string            `json:"referred_by"`           // 推荐人
	SizeCurrencyDisplay string            `json:"size_currency_display"` // 数量货币显示方式
	TapShareRate        ShareRate         `json:"tap_share_rate"`        // TAP分成比例
	TapStatus           string            `json:"tap_status"`            // TAP联盟状态: NONE, ACTIVE, INACTIVE
	Twitter             *Twitter          `json:"twitter"`               // Twitter信息
	TwitterFollowing    map[string]bool   `json:"twitter_following"`     // Twitter关注列表
	Username            string            `json:"username"`              // 用户名
	XPShareRate         ShareRate         `json:"xp_share_rate"`         // XP分成比例
}

// MarginConfig 杠杆配置
type MarginConfig struct {
	Account    string `json:"account"`     // 账户ID
	Leverage   int    `json:"leverage"`    // 杠杆倍数
	MarginType string `json:"margin_type"` // 保证金类型: CROSS(全仓)/ISOLATED(逐仓)
	Market     string `json:"market"`      // 市场符号
}

// FillType 成交类型
type FillType string

const (
	FillTypeFill         FillType = "FILL"
	FillTypeLiquidation  FillType = "LIQUIDATION"
	FillTypeTransfer     FillType = "TRANSFER"
	FillTypeSettleMarket FillType = "SETTLE_MARKET"
	FillTypeRPI          FillType = "RPI"
	FillTypeBlockTrade   FillType = "BLOCK_TRADE"
)

// FillFlag 成交标志
type FillFlag string

const (
	FillFlagInteractive FillFlag = "interactive"
	FillFlagRPI         FillFlag = "rpi"
	FillFlagFastfill    FillFlag = "fastfill"
)

// Liquidity 流动性类型
type Liquidity string

const (
	LiquidityTaker Liquidity = "TAKER"
	LiquidityMaker Liquidity = "MAKER"
)

// Fill 成交记录
type Fill struct {
	Account         string          `json:"account"`          // 产生成交的账户
	ClientID        string          `json:"client_id"`        // 客户端分配的唯一订单ID
	CreatedAt       int64           `json:"created_at"`       // 成交时间(毫秒时间戳)
	Fee             decimal.Decimal `json:"fee"`              // 用户支付的手续费
	FeeCurrency     string          `json:"fee_currency"`     // 手续费币种
	FillType        FillType        `json:"fill_type"`        // 成交类型
	Flags           []FillFlag      `json:"flags"`            // 成交标志,指示特殊属性
	ID              string          `json:"id"`               // 每个FillType的唯一字符串ID
	Liquidity       Liquidity       `json:"liquidity"`        // Maker或Taker
	Market          string          `json:"market"`           // 市场名称
	OrderID         string          `json:"order_id"`         // 订单ID
	OrderbookSeqNo  int64           `json:"orderbook_seq_no"` // 交易执行后的订单簿序列号
	Price           decimal.Decimal `json:"price"`            // 订单成交价格
	RealizedFunding decimal.Decimal `json:"realized_funding"` // 成交的已实现资金费用
	RealizedPnl     decimal.Decimal `json:"realized_pnl"`     // 成交的已实现盈亏
	RemainingSize   decimal.Decimal `json:"remaining_size"`   // 订单剩余数量
	Side            OrderSide       `json:"side"`             // Taker方向
	Size            decimal.Decimal `json:"size"`             // 成交数量
	UnderlyingPrice decimal.Decimal `json:"underlying_price"` // 标的资产价格(现货价格)
}

// FillRes 成交记录响应
type FillRes struct {
	Next    string  `json:"next"`    // 获取下一组记录的指针(如果没有更多记录则为null)
	Prev    string  `json:"prev"`    // 获取上一组记录的指针(如果没有更多记录则为null)
	Results []*Fill `json:"results"` // 成交记录列表
}

// MarketConfig 市场配置信息
type MarkettMarginConfig struct {
	IsolatedMarginLeverage int        `json:"isolated_margin_leverage"` // 隔离保证金杠杆，可为空
	Leverage               int        `json:"leverage"`                 // 杠杆值，可为空
	MarginType             MarginType `json:"margin_type"`              // 保证金类型(CROSS/ISOLATED)，可为空
	Market                 string     `json:"market"`                   // 市场交易对符号，可为空
}

// AccountConfig 账户配置信息
type AccounttMarginConfig struct {
	Account           string                 `json:"account"`            // 账户ID，可为空
	Configs           []*MarkettMarginConfig `json:"configs"`            // 各市场保证金配置列表，可为空
	MarginMethodology string                 `json:"margin_methodology"` // 保证金计算方法(cross_margin/portfolio_margin)，可为空
}

// CreateOrderReq 创建订单请求
type CreateOrderReq struct {
	Instruction        OrderInstruction `json:"instruction"`                    // 订单指令，GTC、IOC、RPI 或 POST_ONLY，如果为空则默认为 GTC
	Market             string           `json:"market"`                         // 订单创建的市场
	Price              string           `json:"price"`                          // 订单价格
	Side               OrderSide        `json:"side"`                           // 订单方向，BUY 或 SELL
	Signature          string           `json:"signature"`                      // 订单签名，格式为 "[r,s]"，由账户的 paradex 私钥签名
	SignatureTimestamp int64            `json:"signature_timestamp"`            // 订单创建的 Unix 时间戳（毫秒），用于签名验证
	Size               decimal.Decimal  `json:"size"`                           // 订单数量
	Type               OrderType        `json:"type"`                           // 订单类型
	ClientID           *string          `json:"client_id,omitempty"`            // 客户端分配的唯一订单 ID，长度不超过 64 个字符
	Flags              []OrderFlag      `json:"flags"`                          // 订单标志，允许的标志：REDUCE_ONLY
	MaxSlippagePrice   *decimal.Decimal `json:"max_slippage_price,omitempty"`   // 市价单的最大滑点价格，如果大于价格带，则使用价格带
	OnBehalfOfAccount  *string          `json:"on_behalf_of_account,omitempty"` // 对应的隔离保证金账户 ID，仅适用于隔离保证金订单
	RecvWindow         *int64           `json:"recv_window,omitempty"`          // 订单接收窗口（毫秒），从签名时间戳起在此时间内被 API 接收则创建订单，最小为 10 毫秒
	SignedImpactPrice  *decimal.Decimal `json:"signed_impact_price,omitempty"`  // [已弃用] 市价单的签名影响价格（base64 编码）
	Stp                *STPType         `json:"stp,omitempty"`                  // 自成交预防，EXPIRE_MAKER、EXPIRE_TAKER 或 EXPIRE_BOTH，如果为空则默认为 EXPIRE_TAKER
	TriggerPrice       string           `json:"trigger_price"`                  // 止损单的触发价格
}

// CreateOrderErrorType 错误类型枚举
type CreateOrderErrorType string

const (
	ValidationError                     CreateOrderErrorType = "VALIDATION_ERROR"
	BindingError                        CreateOrderErrorType = "BINDING_ERROR"
	InternalError                       CreateOrderErrorType = "INTERNAL_ERROR"
	NotFound                            CreateOrderErrorType = "NOT_FOUND"
	ServiceUnavailable                  CreateOrderErrorType = "SERVICE_UNAVAILABLE"
	InvalidRequestParameter             CreateOrderErrorType = "INVALID_REQUEST_PARAMETER"
	OrderIDNotFound                     CreateOrderErrorType = "ORDER_ID_NOT_FOUND"
	OrderIsClosed                       CreateOrderErrorType = "ORDER_IS_CLOSED"
	OrderIsNotOpen                      CreateOrderErrorType = "ORDER_IS_NOT_OPEN"
	InvalidOrderSize                    CreateOrderErrorType = "INVALID_ORDER_SIZE"
	ClientOrderIDNotFound               CreateOrderErrorType = "CLIENT_ORDER_ID_NOT_FOUND"
	DuplicatedClientID                  CreateOrderErrorType = "DUPLICATED_CLIENT_ID"
	InvalidPricePrecision               CreateOrderErrorType = "INVALID_PRICE_PRECISION"
	InvalidSymbol                       CreateOrderErrorType = "INVALID_SYMBOL"
	InvalidToken                        CreateOrderErrorType = "INVALID_TOKEN"
	InvalidEthereumAddress              CreateOrderErrorType = "INVALID_ETHEREUM_ADDRESS"
	InvalidEthereumSignature            CreateOrderErrorType = "INVALID_ETHEREUM_SIGNATURE"
	InvalidStarknetAddress              CreateOrderErrorType = "INVALID_STARKNET_ADDRESS"
	InvalidStarknetSignature            CreateOrderErrorType = "INVALID_STARKNET_SIGNATURE"
	StarknetSignatureVerificationFailed CreateOrderErrorType = "STARKNET_SIGNATURE_VERIFICATION_FAILED"
	BadStarknetRequest                  CreateOrderErrorType = "BAD_STARKNET_REQUEST"
	EthereumSignerMismatch              CreateOrderErrorType = "ETHEREUM_SIGNER_MISMATCH"
	EthereumHashMismatch                CreateOrderErrorType = "ETHEREUM_HASH_MISMATCH"
	NotOnboarded                        CreateOrderErrorType = "NOT_ONBOARDED"
	InvalidTimestamp                    CreateOrderErrorType = "INVALID_TIMESTAMP"
	InvalidBlockExpiration              CreateOrderErrorType = "INVALID_BLOCK_EXPIRATION"
	AccountNotFound                     CreateOrderErrorType = "ACCOUNT_NOT_FOUND"
	InvalidOrderSignature               CreateOrderErrorType = "INVALID_ORDER_SIGNATURE"
	PublicKeyInvalid                    CreateOrderErrorType = "PUBLIC_KEY_INVALID"
	UnauthorizedEthereumAddress         CreateOrderErrorType = "UNAUTHORIZED_ETHEREUM_ADDRESS"
	UnauthorizedError                   CreateOrderErrorType = "UNAUTHORIZED_ERROR"
	EthereumAddressAlreadyOnboarded     CreateOrderErrorType = "ETHEREUM_ADDRESS_ALREADY_ONBOARDED"
	MarketNotFound                      CreateOrderErrorType = "MARKET_NOT_FOUND"
	AllowlistEntryNotFound              CreateOrderErrorType = "ALLOWLIST_ENTRY_NOT_FOUND"
	UsernameInUse                       CreateOrderErrorType = "USERNAME_IN_USE"
	GeoIPBlock                          CreateOrderErrorType = "GEO_IP_BLOCK"
	EthereumAddressBlocked              CreateOrderErrorType = "ETHEREUM_ADDRESS_BLOCKED"
	ProgramNotFound                     CreateOrderErrorType = "PROGRAM_NOT_FOUND"
	ProgramNotSupported                 CreateOrderErrorType = "PROGRAM_NOT_SUPPORTED"
	InvalidDashboard                    CreateOrderErrorType = "INVALID_DASHBOARD"
	MarketNotOpen                       CreateOrderErrorType = "MARKET_NOT_OPEN"
	InvalidReferralCode                 CreateOrderErrorType = "INVALID_REFERRAL_CODE"
	RequestNotAllowed                   CreateOrderErrorType = "REQUEST_NOT_ALLOWED"
	ParentAddressAlreadyOnboarded       CreateOrderErrorType = "PARENT_ADDRESS_ALREADY_ONBOARDED"
	InvalidParentAccount                CreateOrderErrorType = "INVALID_PARENT_ACCOUNT"
	InvalidVaultOperatorChain           CreateOrderErrorType = "INVALID_VAULT_OPERATOR_CHAIN"
	VaultOperatorAlreadyOnboarded       CreateOrderErrorType = "VAULT_OPERATOR_ALREADY_ONBOARDED"
	VaultNameInUse                      CreateOrderErrorType = "VAULT_NAME_IN_USE"
	VaultNotFound                       CreateOrderErrorType = "VAULT_NOT_FOUND"
	VaultStrategyNotFound               CreateOrderErrorType = "VAULT_STRATEGY_NOT_FOUND"
	VaultLimitReached                   CreateOrderErrorType = "VAULT_LIMIT_REACHED"
	BatchSizeOutOfRange                 CreateOrderErrorType = "BATCH_SIZE_OUT_OF_RANGE"
	IsolatedMarketAccountMismatch       CreateOrderErrorType = "ISOLATED_MARKET_ACCOUNT_MISMATCH"
	NoAccessToMarket                    CreateOrderErrorType = "NO_ACCESS_TO_MARKET"
	PointsSummaryNotFound               CreateOrderErrorType = "POINTS_SUMMARY_NOT_FOUND"
	AlgoIDNotFound                      CreateOrderErrorType = "ALGO_ID_NOT_FOUND"
	InvalidDerivationPath               CreateOrderErrorType = "INVALID_DERIVATION_PATH"
	ProfileStatsNotFound                CreateOrderErrorType = "PROFILE_STATS_NOT_FOUND"
	InvalidChain                        CreateOrderErrorType = "INVALID_CHAIN"
	InvalidLayerswapSwap                CreateOrderErrorType = "INVALID_LAYERSWAP_SWAP"
	InvalidRhinofiRequest               CreateOrderErrorType = "INVALID_RHINOFI_REQUEST"
	InvalidRhinofiQuote                 CreateOrderErrorType = "INVALID_RHINOFI_QUOTE"
	InvalidRhinofiQuoteCommit           CreateOrderErrorType = "INVALID_RHINOFI_QUOTE_COMMIT"
	SocialUsernameInUse                 CreateOrderErrorType = "SOCIAL_USERNAME_IN_USE"
	InvalidOauthRequest                 CreateOrderErrorType = "INVALID_OAUTH_REQUEST"
	RpiAccountNotWhitelisted            CreateOrderErrorType = "RPI_ACCOUNT_NOT_WHITELISTED"
	InvalidMarketingCode                CreateOrderErrorType = "INVALID_MARKETING_CODE"
	InvalidJoinWaitlistRequest          CreateOrderErrorType = "INVALID_JOIN_WAITLIST_REQUEST"
	TransactionNotFound                 CreateOrderErrorType = "TRANSACTION_NOT_FOUND"
	OfferNotFound                       CreateOrderErrorType = "OFFER_NOT_FOUND"
	MarketMarginRestricted              CreateOrderErrorType = "MARKET_MARGIN_RESTRICTED"
	NotUnique                           CreateOrderErrorType = "NOT_UNIQUE"
	AccountAlreadyReferred              CreateOrderErrorType = "ACCOUNT_ALREADY_REFERRED"
	OnboardingPeriodExpired             CreateOrderErrorType = "ONBOARDING_PERIOD_EXPIRED"
	OnboardingRateLimited               CreateOrderErrorType = "ONBOARDING_RATE_LIMITED"
	SubaccountsLimitExceeded            CreateOrderErrorType = "SUBACCOUNTS_LIMIT_EXCEEDED"
	InsufficientMinChainBalance         CreateOrderErrorType = "INSUFFICIENT_MIN_CHAIN_BALANCE"
	InvalidSubkey                       CreateOrderErrorType = "INVALID_SUBKEY"
	TokenLimitReached                   CreateOrderErrorType = "TOKEN_LIMIT_REACHED"
	InvalidTokenScope                   CreateOrderErrorType = "INVALID_TOKEN_SCOPE"
	InsufficientTransferrableXP         CreateOrderErrorType = "INSUFFICIENT_TRANSFERRABLE_XP"
	TransferLimitReached                CreateOrderErrorType = "TRANSFER_LIMIT_REACHED"
)

// CreateOrderError 错误信息
type CreateOrderError struct {
	Code    CreateOrderErrorType `json:"error"`   // 错误类型枚举值
	Message string               `json:"message"` // 错误消息
}

func (err CreateOrderError) Error() string {
	return fmt.Sprintf("code: %s, message: %s", err.Code, err.Message)
}

// CreateBatchOrdersRes 创建批量订单响应
type CreateBatchOrdersRes struct {
	Orders []*Order            `json:"orders"` // 订单列表
	Errors []*CreateOrderError `json:"errors"` // 错误列表
}

// JsonRpcMessage JSON-RPC消息
type JsonRpcMessage struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	UsIn    int64           `json:"usIn,omitempty"`
	UsDiff  int64           `json:"usDiff,omitempty"`
}

// SubscriptionPayload 订阅消息
type SubscriptionPayload struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}
