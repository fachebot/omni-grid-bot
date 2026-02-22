// Package paradex 提供Paradex交易所的StarkNet签名相关功能
// 支持StarkNet ECDSA签名、TypedData构造和消息哈希计算
package paradex

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/stark-curve/fp"
	"github.com/dontpanicdao/caigo"
	"github.com/dontpanicdao/caigo/types"
	"github.com/shopspring/decimal"

	pedersenhash "github.com/consensys/gnark-crypto/ecc/stark-curve/pedersen-hash"
)

var scaleX8Decimal = decimal.RequireFromString("100000000")
var snMessageBigInt = types.UTF8StrToBig("StarkNet Message")

// OnboardingPayload 入驻请求数据
// 用于用户首次入驻时的StarkNet身份验证
type OnboardingPayload struct {
	Action string // 操作类型
}

func (o *OnboardingPayload) FmtDefinitionEncoding(field string) (fmtEnc []*big.Int) {
	if field == "action" {
		fmtEnc = append(fmtEnc, types.StrToFelt(o.Action).Big())
	}

	return
}

// AuthPayload 身份验证请求数据
// 用于JWT令牌的StarkNet签名认证
type AuthPayload struct {
	Method     string // HTTP方法
	Path       string // 请求路径
	Body       string // 请求体
	Timestamp  string // 时间戳
	Expiration string // 过期时间
}

func (o *AuthPayload) FmtDefinitionEncoding(field string) (fmtEnc []*big.Int) {
	switch field {
	case "method":
		fmtEnc = append(fmtEnc, types.StrToFelt(o.Method).Big())
	case "path":
		fmtEnc = append(fmtEnc, types.StrToFelt(o.Path).Big())
	case "body":
		// this is required as types.StrToFelt("") returns nil, which seems to be an SN bug
		fmtEnc = append(fmtEnc, big.NewInt(0))
	case "timestamp":
		fmtEnc = append(fmtEnc, types.StrToFelt(o.Timestamp).Big())
	case "expiration":
		if o.Expiration != "" {
			fmtEnc = append(fmtEnc, types.StrToFelt(o.Expiration).Big())
		}
	}
	return fmtEnc
}

// OrderPayload 订单签名数据
// 用于订单提交的StarkNet签名，包含时间戳、市场、方向、类型、数量和价格
type OrderPayload struct {
	Timestamp int64  // 签名创建时的Unix时间戳(毫秒)
	Market    string // 市场名称，如 ETH-USD-PERP
	Side      string // 方向：1=买入，2=卖出
	OrderType string // 订单类型：MARKET 或 LIMIT
	Size      string // 数量，乘以1e8精度
	Price     string // 价格，乘以1e8精度（市价单为0）
}

// GetScaledSize 获取缩放后的数量
// 将数量乘以1e8精度，如0.2转换为20_000_000 (0.2 * 10^8)
func (o *OrderPayload) GetScaledSize() string {
	return decimal.RequireFromString(o.Size).Mul(scaleX8Decimal).String()
}

// GetScaledPrice 获取缩放后的价格
// 将价格乘以1e8精度，如3_309.33转换为330_933_000_000 (3_309.33 * 10^8)
func (o *OrderPayload) GetScaledPrice() string {
	price := o.Price
	if OrderType(o.OrderType) == OrderTypeMarket {
		return "0"
	} else {
		return decimal.RequireFromString(price).Mul(scaleX8Decimal).String()
	}
}

func (o *OrderPayload) FmtDefinitionEncoding(field string) (fmtEnc []*big.Int) {
	switch field {
	case "timestamp":
		fmtEnc = append(fmtEnc, big.NewInt(o.Timestamp))
	case "market":
		fmtEnc = append(fmtEnc, types.StrToFelt(o.Market).Big())
	case "side":
		side := OrderSide(o.Side).Get()
		fmtEnc = append(fmtEnc, types.StrToFelt(side).Big())
	case "orderType":
		fmtEnc = append(fmtEnc, types.StrToFelt(o.OrderType).Big())
	case "size":
		size := o.GetScaledSize()
		fmtEnc = append(fmtEnc, types.StrToFelt(size).Big())
	case "price":
		price := o.GetScaledPrice()
		fmtEnc = append(fmtEnc, types.StrToFelt(price).Big())
	}

	return fmtEnc
}

// domainDefinition 定义StarkNet域验证类型
func domainDefinition() *caigo.TypeDef {
	return &caigo.TypeDef{Definitions: []caigo.Definition{
		{Name: "name", Type: "felt"},
		{Name: "chainId", Type: "felt"},
		{Name: "version", Type: "felt"}}}
}

// domain 创建StarkNet域对象
// 包含Paradex名称、版本和链ID
func domain(chainId string) *caigo.Domain {
	return &caigo.Domain{
		Name:    "Paradex",
		Version: "1",
		ChainId: chainId,
	}
}

// onboardingPayloadDefinition 定义入驻请求的TypedData结构
func onboardingPayloadDefinition() *caigo.TypeDef {
	return &caigo.TypeDef{Definitions: []caigo.Definition{
		{Name: "action", Type: "felt"}}}
}

// authPayloadDefinition 定义身份验证请求的TypedData结构
func authPayloadDefinition() *caigo.TypeDef {
	return &caigo.TypeDef{Definitions: []caigo.Definition{
		{Name: "method", Type: "felt"},
		{Name: "path", Type: "felt"},
		{Name: "body", Type: "felt"},
		{Name: "timestamp", Type: "felt"},
		{Name: "expiration", Type: "felt"}}}
}

// orderPayloadDefinition 定义订单请求的TypedData结构
func orderPayloadDefinition() *caigo.TypeDef {
	return &caigo.TypeDef{Definitions: []caigo.Definition{
		{Name: "timestamp", Type: "felt"},
		{Name: "market", Type: "felt"},
		{Name: "side", Type: "felt"},
		{Name: "orderType", Type: "felt"},
		{Name: "size", Type: "felt"},
		{Name: "price", Type: "felt"}}}
}

// onboardingTypes 返回入驻验证的TypedData类型映射
func onboardingTypes() map[string]caigo.TypeDef {
	return map[string]caigo.TypeDef{
		"StarkNetDomain": *domainDefinition(),
		"Constant":       *onboardingPayloadDefinition(),
	}
}

// authTypes 返回身份验证的TypedData类型映射
func authTypes() map[string]caigo.TypeDef {
	return map[string]caigo.TypeDef{
		"StarkNetDomain": *domainDefinition(),
		"Request":        *authPayloadDefinition(),
	}
}

// orderTypes 返回订单验证的TypedData类型映射
func orderTypes() map[string]caigo.TypeDef {
	return map[string]caigo.TypeDef{
		"StarkNetDomain": *domainDefinition(),
		"Order":          *orderPayloadDefinition(),
	}
}

// NewVerificationTypedData 创建验证类型的TypedData
// 根据验证类型(Onboarding/Auth/Order)创建对应的TypedData结构
func NewVerificationTypedData(vType VerificationType, chainId string) (*caigo.TypedData, error) {
	if vType == VerificationTypeOnboarding {
		return NewTypedData(onboardingTypes(), domain(chainId), "Constant")
	}
	if vType == VerificationTypeAuth {
		return NewTypedData(authTypes(), domain(chainId), "Request")
	}
	if vType == VerificationTypeOrder {
		return NewTypedData(orderTypes(), domain(chainId), "Order")
	}
	return nil, errors.New("invalid validation type")
}

// NewTypedData 创建TypedData结构
// 用于构造StarkNet签名消息，与前端MetaMask签名保持一致
func NewTypedData(types map[string]caigo.TypeDef, domain *caigo.Domain, pType string) (*caigo.TypedData, error) {
	typedData, err := caigo.NewTypedData(
		types,
		pType,
		*domain,
	)

	if err != nil {
		return nil, errors.New("failed to create typed data with caigo")
	}

	return &typedData, nil
}

// PedersenArray 使用Pedersen哈希算法计算数组哈希
// 将多个大整数转换为Field元素并进行Pedersen哈希
func PedersenArray(elems []*big.Int) *big.Int {
	fpElements := make([]*fp.Element, len(elems))
	for i, elem := range elems {
		fpElements[i] = new(fp.Element).SetBigInt(elem)
	}
	hash := pedersenhash.PedersenArray(fpElements...)
	return hash.BigInt(new(big.Int))
}

// GetMessageHash 计算StarkNet消息哈希
// 构造包含StarkNet消息前缀、域编码、账户地址和消息编码的完整哈希
func GetMessageHash(td *caigo.TypedData, domEnc *big.Int, account *big.Int, msg caigo.TypedMessage, sc caigo.StarkCurve) (hash *big.Int, err error) {
	elements := []*big.Int{snMessageBigInt, domEnc, account, nil}

	msgEnc, err := td.GetTypedMessageHash(td.PrimaryType, msg, sc)
	if err != nil {
		return hash, fmt.Errorf("could not hash message: %w", err)
	}
	elements[3] = msgEnc
	hash, err = sc.ComputeHashOnElements(elements)
	return hash, err
}

// GnarkGetMessageHash 使用Gnark库计算StarkNet消息哈希
// 使用Pedersen哈希算法替代caigo的默认实现
func GnarkGetMessageHash(td *caigo.TypedData, domEnc *big.Int, account *big.Int, msg caigo.TypedMessage, sc caigo.StarkCurve) (hash *big.Int, err error) {
	msgEnc, err := GnarkGetTypedMessageHash(td, td.PrimaryType, msg)
	if err != nil {
		return nil, fmt.Errorf("could not hash message: %w", err)
	}
	elements := []*big.Int{snMessageBigInt, domEnc, account, msgEnc}
	hash = PedersenArray(elements)
	return hash, err
}

// GnarkGetTypedMessageHash 使用Gnark计算TypedData消息哈希
// 将类型定义和消息编码后使用Pedersen数组哈希
func GnarkGetTypedMessageHash(td *caigo.TypedData, inType string, msg caigo.TypedMessage) (hash *big.Int, err error) {
	prim := td.Types[inType]
	elements := make([]*big.Int, 0, len(prim.Definitions)+1)
	elements = append(elements, prim.Encoding)

	for _, def := range prim.Definitions {
		if def.Type == "felt" {
			fmtDefinitions := msg.FmtDefinitionEncoding(def.Name)
			elements = append(elements, fmtDefinitions...)
		} else {
			panic("not felt")
		}
	}
	hash = PedersenArray(elements)
	return hash, err
}
