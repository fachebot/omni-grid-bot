package nado

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

// Result API 通用响应结构
// 用于封装所有 API 请求的响应数据，包含状态、类型、数据体及错误信息
type Result struct {
	Status      string          `json:"status"`       // 响应状态（如 "success"、"error"）
	RequestType string          `json:"request_type"` // 请求类型标识
	Data        json.RawMessage `json:"data"`         // 响应数据体（延迟解析的 JSON）
	ErrMsg      *string         `json:"error"`        // 错误消息（可选）
	ErrorCode   *int            `json:"error_code"`   // 错误码（可选）
}

// Error 实现 error 接口，返回格式化的错误信息
func (r *Result) Error() string {
	if r.ErrorCode == nil || r.ErrMsg == nil {
		return ""
	}
	return fmt.Sprintf("code: %d, error: %s", *r.ErrorCode, *r.ErrMsg)
}

// NoncesRes 获取 Nonces 响应
// 返回交易和订单的 nonce 值，用于防止重放攻击
type NoncesRes struct {
	TxNonce    json.Number `json:"tx_nonce"`    // 交易 nonce，用于链上交易
	OrderNonce json.Number `json:"order_nonce"` // 订单 nonce，用于订单签名
}

// ContractsRes 获取合约信息响应
// 返回链 ID 和端点合约地址
type ContractsRes struct {
	ChainId      decimal.Decimal `json:"chain_id"`      // 区块链网络 ID
	EndpointAddr common.Address  `json:"endpoint_addr"` // 交易所端点合约地址
}

// OpenOrder 活跃订单信息
type OpenOrder struct {
	ProductID      int64           `json:"product_id"`      // 订单执行的产品ID
	Sender         Sender          `json:"sender"`          // 下单的子账户
	PriceX18       decimal.Decimal `json:"price_x18"`       // 原始订单价格（18位精度）
	Amount         decimal.Decimal `json:"amount"`          // 原始买入或卖出的基础货币数量
	Expiration     decimal.Decimal `json:"expiration"`      // 原始订单过期时间
	Nonce          decimal.Decimal `json:"nonce"`           // 原始订单随机数
	UnfilledAmount decimal.Decimal `json:"unfilled_amount"` // 未成交基础货币数量
	Digest         string          `json:"digest"`          // 订单的唯一哈希值
	PlacedAt       int64           `json:"placed_at"`       // 下单时间
	Appendix       Appendix        `json:"appendix"`        // 原始订单附加信息
	OrderType      string          `json:"order_type"`      // 订单类型
}

// OpenOrdersRes 活跃订单列表响应
// 包含订单列表的响应结构
type OpenOrdersRes struct {
	Orders []OpenOrder `json:"orders"` // 订单列表
}

// ArchiveOrder 归档订单信息
// 包含订单的完整状态和成交信息
type ArchiveOrder struct {
	Digest                string          `json:"digest"`                   // 订单的唯一哈希值
	Subaccount            Sender          `json:"subaccount"`               // 下单的子账户
	ProductID             int64           `json:"product_id"`               // 订单执行的产品ID
	SubmissionIdx         decimal.Decimal `json:"submission_idx"`           // 生成订单的区块链交易唯一标识，多笔成交订单为第一笔的submission_idx
	LastFillSubmissionIdx decimal.Decimal `json:"last_fill_submission_idx"` // 多笔成交订单为最后一笔的submission_idx，单笔成交与submission_idx相同
	Amount                decimal.Decimal `json:"amount"`                   // 原始买入或卖出的基础货币数量
	PriceX18              decimal.Decimal `json:"price_x18"`                // 原始订单价格（18位精度）
	BaseFilled            decimal.Decimal `json:"base_filled"`              // 该订单已成交的基础货币总量（如：BTC）
	QuoteFilled           decimal.Decimal `json:"quote_filled"`             // 该订单已成交的计价货币总量（如：USDT）
	Fee                   decimal.Decimal `json:"fee"`                      // 该订单支付的总手续费
	ClosedAmount          decimal.Decimal `json:"closed_amount"`            // 该订单结清金额
	RealizedPnl           decimal.Decimal `json:"realized_pnl"`             // 已实现利润
	Expiration            decimal.Decimal `json:"expiration"`               // 原始订单过期时间
	Nonce                 decimal.Decimal `json:"nonce"`                    // 原始订单随机数
	Appendix              string          `json:"appendix"`                 // 原始订单附加信息
}

// ArchiveOrdersRes 归档订单列表响应
// 包含订单列表的响应结构
type ArchiveOrdersRes struct {
	Orders []ArchiveOrder `json:"orders"` // 订单列表
}

// HealthInfo 健康度信息
// 用于评估账户的风险状况和清算风险
type HealthInfo struct {
	Assets      decimal.Decimal `json:"assets"`      // 资产总额
	Liabilities decimal.Decimal `json:"liabilities"` // 负债总额
	Health      decimal.Decimal `json:"health"`      // 健康度数值（>0 表示安全，<=0 可能被清算）
}

// SpotBalance 现货余额
// 表示某个现货产品的持仓信息
type SpotBalance struct {
	ProductID int           `json:"product_id"` // 产品ID
	Balance   BalanceAmount `json:"balance"`    // 余额信息
}

// PerpBalance 永续合约余额
// 表示某个永续合约产品的持仓信息，包含资金费率相关数据
type PerpBalance struct {
	ProductID                int             `json:"product_id"`                  // 产品ID
	Balance                  BalanceAmount   `json:"balance"`                     // 余额信息（正数为多头，负数为空头）
	VQuoteBalance            decimal.Decimal `json:"v_quote_balance"`             // 虚拟报价余额，用于计算盈亏
	LastCumulativeFundingX18 decimal.Decimal `json:"last_cumulative_funding_x18"` // 最后累计资金费率（18位精度）
}

// BalanceAmount 余额金额
// 通用的金额结构
type BalanceAmount struct {
	Amount decimal.Decimal `json:"amount"` // 金额数值
}

// SpotProduct 现货产品信息
// 包含现货交易对的完整配置和状态
type SpotProduct struct {
	ProductID      int             `json:"product_id"`       // 产品ID
	OraclePriceX18 decimal.Decimal `json:"oracle_price_x18"` // 预言机价格（18位精度）
	Risk           RiskInfo        `json:"risk"`             // 风险参数
	Config         ProductConfig   `json:"config"`           // 产品配置
	State          ProductState    `json:"state"`            // 产品状态
	BookInfo       BookInfo        `json:"book_info"`        // 订单簿信息
}

// PerpProduct 永续合约产品信息
// 包含永续合约的完整配置和状态
type PerpProduct struct {
	ProductID      int              `json:"product_id"`       // 产品ID
	OraclePriceX18 decimal.Decimal  `json:"oracle_price_x18"` // 预言机价格（18位精度）
	Risk           RiskInfo         `json:"risk"`             // 风险参数
	State          PerpProductState `json:"state"`            // 产品状态
	BookInfo       BookInfo         `json:"book_info"`        // 订单簿信息
}

// RiskInfo 风险参数信息
// 定义产品的保证金和杠杆相关参数
type RiskInfo struct {
	LongWeightInitialX18      decimal.Decimal `json:"long_weight_initial_x18"`      // 多头初始权重（用于计算初始保证金）
	ShortWeightInitialX18     decimal.Decimal `json:"short_weight_initial_x18"`     // 空头初始权重（用于计算初始保证金）
	LongWeightMaintenanceX18  decimal.Decimal `json:"long_weight_maintenance_x18"`  // 多头维持权重（用于计算维持保证金）
	ShortWeightMaintenanceX18 decimal.Decimal `json:"short_weight_maintenance_x18"` // 空头维持权重（用于计算维持保证金）
	PriceX18                  decimal.Decimal `json:"price_x18"`                    // 风险计算使用的价格
}

// ProductConfig 产品配置
// 定义现货产品的利率模型和费用参数
type ProductConfig struct {
	Token                     string          `json:"token"`                        // 代币合约地址
	InterestInflectionUtilX18 decimal.Decimal `json:"interest_inflection_util_x18"` // 利率拐点利用率（利率曲线转折点）
	InterestFloorX18          decimal.Decimal `json:"interest_floor_x18"`           // 利率下限
	InterestSmallCapX18       decimal.Decimal `json:"interest_small_cap_x18"`       // 低利用率时的利率上限
	InterestLargeCapX18       decimal.Decimal `json:"interest_large_cap_x18"`       // 高利用率时的利率上限
	WithdrawFeeX18            decimal.Decimal `json:"withdraw_fee_x18"`             // 提现手续费率
	MinDepositRateX18         decimal.Decimal `json:"min_deposit_rate_x18"`         // 最小存款利率
}

// ProductState 产品状态
// 记录现货产品的存借款累计状态
type ProductState struct {
	CumulativeDepositsMultiplierX18 decimal.Decimal `json:"cumulative_deposits_multiplier_x18"` // 累计存款乘数（用于计算实际存款利息）
	CumulativeBorrowsMultiplierX18  decimal.Decimal `json:"cumulative_borrows_multiplier_x18"`  // 累计借款乘数（用于计算实际借款利息）
	TotalDepositsNormalized         decimal.Decimal `json:"total_deposits_normalized"`          // 标准化总存款量
	TotalBorrowsNormalized          decimal.Decimal `json:"total_borrows_normalized"`           // 标准化总借款量
}

// PerpProductState 永续合约产品状态
// 记录永续合约的资金费率和未平仓量
type PerpProductState struct {
	CumulativeFundingLongX18  decimal.Decimal `json:"cumulative_funding_long_x18"`  // 多头累计资金费率
	CumulativeFundingShortX18 decimal.Decimal `json:"cumulative_funding_short_x18"` // 空头累计资金费率
	AvailableSettle           decimal.Decimal `json:"available_settle"`             // 可用结算金额
	OpenInterest              decimal.Decimal `json:"open_interest"`                // 未平仓合约量
}

// BookInfo 订单簿信息
// 定义订单簿的交易规则和参数
type BookInfo struct {
	SizeIncrement     decimal.Decimal `json:"size_increment"`      // 数量增量（最小下单数量步长）
	PriceIncrementX18 decimal.Decimal `json:"price_increment_x18"` // 价格增量（最小价格步长，18位精度）
	MinSize           decimal.Decimal `json:"min_size"`            // 最小下单数量
	CollectedFees     decimal.Decimal `json:"collected_fees"`      // 已收集的手续费总额
}

// PreState 交易前状态
// 用于记录交易执行前的账户快照，便于对比交易影响
type PreState struct {
	Healths             []HealthInfo  `json:"healths"`              // 交易前健康度信息
	HealthContributions [][]string    `json:"health_contributions"` // 交易前各产品对健康度的贡献
	SpotBalances        []SpotBalance `json:"spot_balances"`        // 交易前现货余额
	PerpBalances        []PerpBalance `json:"perp_balances"`        // 交易前永续合约余额
}

// SubaccountData 子账户详细数据
// 包含子账户的完整状态信息，用于账户查询接口
type SubaccountData struct {
	Subaccount          Sender              `json:"subaccount"`           // 子账户标识（32字节十六进制）
	Exists              bool                `json:"exists"`               // 账户是否存在
	Healths             []HealthInfo        `json:"healths"`              // 健康度信息数组（通常包含初始和维持两种）
	HealthContributions [][]decimal.Decimal `json:"health_contributions"` // 各产品对健康度的贡献明细
	SpotCount           int                 `json:"spot_count"`           // 持有的现货产品数量
	PerpCount           int                 `json:"perp_count"`           // 持有的永续合约产品数量
	SpotBalances        []SpotBalance       `json:"spot_balances"`        // 现货余额列表
	PerpBalances        []PerpBalance       `json:"perp_balances"`        // 永续合约余额列表
	SpotProducts        []SpotProduct       `json:"spot_products"`        // 现货产品详情列表
	PerpProducts        []PerpProduct       `json:"perp_products"`        // 永续合约产品详情列表
	PreState            *PreState           `json:"pre_state,omitempty"`  // 交易前状态快照（可选）
}

// Subaccount 子账户信息
// 子账户的基本元数据
type Subaccount struct {
	ID             string `json:"id"`              // 内部子账户ID（数据库主键）
	Subaccount     Sender `json:"subaccount"`      // 子账户的十六进制字符串（钱包地址+子账户名称，32字节）
	Address        string `json:"address"`         // 钱包地址的十六进制字符串（20字节）
	SubaccountName string `json:"subaccount_name"` // 子账户标识符（12字节字符串）
	CreatedAt      string `json:"created_at"`      // 子账户创建时间（ISO 8601格式）
	Isolated       bool   `json:"isolated"`        // 是否为隔离仓位的子账户
}

// SubaccountsRes 子账户列表响应
type SubaccountsRes struct {
	Subaccounts []Subaccount `json:"subaccounts"` // 子账户列表
}

// OrderParams 订单参数
// 构建订单签名所需的核心参数
type OrderParams struct {
	Sender     string `json:"sender"`     // 交易发送者的子账户32字节十六进制字符串（地址+子账户名）
	PriceX18   string `json:"priceX18"`   // 订单价格乘以1e18后的值，使用字符串保证精度
	Amount     string `json:"amount"`     // 订单数量乘以1e18后的值（正数买入，负数卖出）
	Expiration string `json:"expiration"` // 订单自动取消的时间戳（Unix纪元后的秒数）
	Nonce      string `json:"nonce"`      // 用于区分多次相同订单的非重复值（通常使用时间戳）
	Appendix   string `json:"appendix"`   // 编码订单属性（执行类型、隔离头寸、TWAP参数、触发类型等）
}

// PlaceOrderInfo 下单信息
// 单个订单的完整提交信息
type PlaceOrderInfo struct {
	ProductID    int         `json:"product_id"`              // 现货或永续产品ID
	Order        OrderParams `json:"order"`                   // 订单参数对象
	Signature    string      `json:"signature"`               // 订单签名的十六进制字符串（EIP-712签名）
	Digest       *string     `json:"digest,omitempty"`        // 订单哈希的十六进制字符串（可选，服务端会自动计算）
	SpotLeverage *bool       `json:"spot_leverage,omitempty"` // 是否使用杠杆，false时若导致子账户借款则下单失败（可选，默认为true）
	ID           *int64      `json:"id,omitempty"`            // 客户端自定义ID，在Fill和OrderUpdate流事件中返回（可选）
}

// PlaceOrders 下单列表
// 批量下单的订单容器
type PlaceOrders struct {
	Orders []*PlaceOrderInfo `json:"orders"` // 订单列表
}

// PlaceOrdersReq 批量下单请求
type PlaceOrdersReq struct {
	PlaceOrders PlaceOrders `json:"place_orders"` // 订单列表包装
}

// PlaceOrderResult 下单结果
// 单个订单的提交结果
type PlaceOrderResult struct {
	Error  *string `json:"error"`  // 错误信息（nil表示成功）
	Digest *string `json:"digest"` // 订单Hash（成功时返回）
}

// PlaceOrdersRes 批量下单响应
// 返回每个订单的提交结果，顺序与请求中的订单一一对应
type PlaceOrdersRes []PlaceOrderResult

// CancelProductOrdersTx 按产品取消订单的交易参数
// 用于构建取消订单的签名数据
type CancelProductOrdersTx struct {
	Sender     Sender `json:"sender"`     // 发送者子账户标识
	ProductIds []int  `json:"productIds"` // 要取消订单的产品ID列表
	Nonce      string `json:"nonce"`      // 交易nonce，防止重放
}

// CancelProductOrders 按产品取消订单
// 包含签名后的取消订单请求
type CancelProductOrders struct {
	Tx        CancelProductOrdersTx `json:"tx"`        // 交易参数
	Signature string                `json:"signature"` // 交易签名
}

// CancelProductOrdersReq 取消多个产品订单请求
type CancelProductOrdersReq struct {
	CancelProductOrders CancelProductOrders `json:"cancel_product_orders"` // 取消订单请求体
}

// CancelProductOrdersRes 取消多个产品订单响应
type CancelProductOrdersRes struct{}
